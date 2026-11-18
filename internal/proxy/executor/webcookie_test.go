package executor

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func webCookieTestRequest(t *testing.T, body string, stream bool, client *http.Client) *Request {
	t.Helper()
	ttft := int64(0)
	return &Request{
		Ctx:      t.Context(),
		Body:     []byte(body),
		APIKey:   "session=abc123; sso=def456",
		IsStream: stream,
		Client:   client,
		TTFT:     &ttft,
	}
}

// swapSpecURL repoints a web-cookie spec's chat URL at the fake server for the
// duration of the test.
func swapVeniceSpec(t *testing.T, url string) {
	t.Helper()
	original := veniceWebSpec
	v := original
	v.ChatURL = url
	veniceWebSpec = v
	t.Cleanup(func() { veniceWebSpec = original })
}

func swapV0Spec(t *testing.T, url string) {
	t.Helper()
	original := v0VercelWebSpec
	v := original
	v.ChatURL = url
	v0VercelWebSpec = v
	t.Cleanup(func() { v0VercelWebSpec = original })
}

func swapZenmuxURL(t *testing.T, url string) {
	t.Helper()
	original := zenmuxFreeChatURL
	zenmuxFreeChatURL = url
	t.Cleanup(func() { zenmuxFreeChatURL = original })
}

func swapPoeURL(t *testing.T, url string) {
	t.Helper()
	original := poeWebGQLURL
	poeWebGQLURL = url
	t.Cleanup(func() { poeWebGQLURL = original })
}

func swapTencentBase(t *testing.T, base string) {
	t.Helper()
	original := tencentAIStudioBase
	tencentAIStudioBase = base
	t.Cleanup(func() { tencentAIStudioBase = original })
}

func TestForwardVeniceWebCollectSendsCookieAndBuildsCompletion(t *testing.T) {
	var gotCookie, gotAccept string
	var gotPayload string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotAccept = r.Header.Get("Accept")
		body, _ := io.ReadAll(r.Body)
		gotPayload = string(body)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ven hello"}}]}`)
	}))
	defer srv.Close()
	swapVeniceSpec(t, srv.URL)

	rec := httptest.NewRecorder()
	req := webCookieTestRequest(t, `{"messages":[{"role":"user","content":"hi"}],"model":"venice-default"}`, false, srv.Client())
	if err := ForwardVeniceWeb(rec, req); err != nil {
		t.Fatalf("ForwardVeniceWeb: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	if gotCookie != "session=abc123; sso=def456" {
		t.Errorf("Cookie = %q, want the pasted header", gotCookie)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
	if !strings.Contains(gotPayload, `"venice-default"`) {
		t.Errorf("payload = %s, want the client model passed through", gotPayload)
	}
	if !strings.Contains(rec.Body.String(), `"content":"ven hello"`) {
		t.Errorf("body = %s, want it to carry the upstream message content", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"object":"chat.completion"`) {
		t.Errorf("body = %s, want a chat.completion envelope", rec.Body.String())
	}
}

func TestForwardVeniceWebRelaysSSEAsOpenAIChunks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"he\"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"llo\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()
	swapVeniceSpec(t, srv.URL)

	rec := httptest.NewRecorder()
	req := webCookieTestRequest(t, `{"messages":[{"role":"user","content":"hi"}]}`, true, srv.Client())
	if err := ForwardVeniceWeb(rec, req); err != nil {
		t.Fatalf("ForwardVeniceWeb: %v", err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"content":"he"`) || !strings.Contains(body, `"content":"llo"`) {
		t.Errorf("stream = %q, want both deltas re-emitted", body)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Errorf("stream = %q, want a terminal [DONE]", body)
	}
	if !strings.Contains(body, `"object":"chat.completion.chunk"`) {
		t.Errorf("stream = %q, want chat.completion.chunk objects", body)
	}
}

func TestForwardV0VercelWebPropagatesReasoningContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"answer","reasoning_content":"thinking"}}]}`)
	}))
	defer srv.Close()
	swapV0Spec(t, srv.URL)

	rec := httptest.NewRecorder()
	req := webCookieTestRequest(t, `{"messages":[{"role":"user","content":"hi"}]}`, false, srv.Client())
	if err := ForwardV0VercelWeb(rec, req); err != nil {
		t.Fatalf("ForwardV0VercelWeb: %v", err)
	}
	if !strings.Contains(rec.Body.String(), `"reasoning_content":"thinking"`) {
		t.Errorf("body = %s, want reasoning_content to survive the relay", rec.Body.String())
	}
}

func TestForwardPoeWebMapsModelAndExtractsBotText(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		if got := r.Header.Get("Cookie"); got != "p-b=tok123" {
			t.Errorf("Cookie = %q, want p-b=tok123", got)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":{"chatWithBot":{"text":"poe says hi","messageId":"m-1","state":"complete"}}}`)
	}))
	defer srv.Close()
	swapPoeURL(t, srv.URL)

	rec := httptest.NewRecorder()
	req := webCookieTestRequest(t, `{"messages":[{"role":"user","content":"hello"}],"model":"claude-3-opus"}`, false, srv.Client())
	req.APIKey = "p-b=tok123"
	if err := ForwardPoeWeb(rec, req); err != nil {
		t.Fatalf("ForwardPoeWeb: %v", err)
	}
	if !strings.Contains(gotBody, `"Claude-3-Opus"`) {
		t.Errorf("GraphQL variables = %s, want the mapped bot name Claude-3-Opus", gotBody)
	}
	if !strings.Contains(rec.Body.String(), `"content":"poe says hi"`) {
		t.Errorf("body = %s, want the bot text in the completion", rec.Body.String())
	}
}

func TestForwardTencentAIStudioWebPassesThroughUpstreamBody(t *testing.T) {
	var gotPath, gotCookie string
	var gotPayload string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotCookie = r.Header.Get("Cookie")
		raw, _ := io.ReadAll(r.Body)
		gotPayload = string(raw)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"answer":"hunyuan says hi"}`)
	}))
	defer srv.Close()
	swapTencentBase(t, srv.URL)

	rec := httptest.NewRecorder()
	req := webCookieTestRequest(t, `{"messages":[{"role":"user","content":"hi"}],"model":"hunyuan-3d"}`, true, srv.Client())
	if err := ForwardTencentAIStudioWeb(rec, req); err != nil {
		t.Fatalf("ForwardTencentAIStudioWeb: %v", err)
	}
	if gotPath != "/api/chat/Hunyuan3D" {
		t.Errorf("path = %q, want the Hunyuan3D codename lane", gotPath)
	}
	if gotCookie != "session=abc123; sso=def456" {
		t.Errorf("Cookie = %q, want the pasted header", gotCookie)
	}
	if !strings.Contains(gotPayload, `"Hunyuan3D"`) {
		t.Errorf("payload = %s, want the mapped codename", gotPayload)
	}
	if !strings.Contains(rec.Body.String(), `"hunyuan says hi"`) {
		t.Errorf("body = %s, want the upstream body streamed through verbatim", rec.Body.String())
	}
}

func TestForwardZenmuxFreeCollectsAnthropicStream(t *testing.T) {
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"zen \"}}\n\n")
		fmt.Fprint(w, "event: content_block_delta\n")
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"mux\"}}\n\n")
		fmt.Fprint(w, "event: message_delta\n")
		fmt.Fprint(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n")
	}))
	defer srv.Close()
	swapZenmuxURL(t, srv.URL)

	rec := httptest.NewRecorder()
	req := webCookieTestRequest(t, `{"messages":[{"role":"system","content":"be terse"},{"role":"user","content":"hi"}],"model":"z-ai/glm-4.7-flash-free"}`, false, srv.Client())
	req.APIKey = "ctoken=ctok123; session=xyz"
	if err := ForwardZenmuxFree(rec, req); err != nil {
		t.Fatalf("ForwardZenmuxFree: %v", err)
	}
	if !strings.Contains(gotURL, "ctoken=ctok123") {
		t.Errorf("url = %q, want the ctoken query param", gotURL)
	}
	if !strings.Contains(rec.Body.String(), `"content":"zen mux"`) {
		t.Errorf("body = %s, want the collected anthropic text folded into one completion", rec.Body.String())
	}
}

func TestForwardZenmuxFreeStreamTranslatesToOpenAIChunks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"zen\"}}\n\n")
		fmt.Fprint(w, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"}}\n\n")
	}))
	defer srv.Close()
	swapZenmuxURL(t, srv.URL)

	rec := httptest.NewRecorder()
	req := webCookieTestRequest(t, `{"messages":[{"role":"user","content":"hi"}]}`, true, srv.Client())
	req.APIKey = "ctoken=ctok123"
	if err := ForwardZenmuxFree(rec, req); err != nil {
		t.Fatalf("ForwardZenmuxFree: %v", err)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"role":"assistant"`) {
		t.Errorf("stream = %q, want the role delta first", body)
	}
	if !strings.Contains(body, `"content":"zen"`) {
		t.Errorf("stream = %q, want the content delta re-framed as OpenAI", body)
	}
	if !strings.Contains(body, `"finish_reason":"end_turn"`) {
		t.Errorf("stream = %q, want the message_delta stop reason surfaced", body)
	}
}

func TestWebCookiePendingExecutorAnswers503(t *testing.T) {
	executor := ForwardWebCookiePending("claude-web")
	rec := httptest.NewRecorder()
	req := webCookieTestRequest(t, `{"messages":[{"role":"user","content":"hi"}]}`, false, nil)
	err := executor(rec, req)
	if err == nil {
		t.Fatal("pending executor returned nil error")
	}
	if !strings.Contains(err.Error(), "not ported") {
		t.Errorf("error = %v, want it to name the missing port", err)
	}
}

func TestWebCookieStripPrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "bare cookie", in: "session=abc", want: "session=abc"},
		{name: "cookie header prefix", in: "cookie: session=abc", want: "session=abc"},
		{name: "bearer prefix", in: "Bearer session=abc", want: "session=abc"},
		{name: "whitespace", in: "   session=abc   ", want: "session=abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := webCookieStripPrefix(tt.in); got != tt.want {
				t.Errorf("webCookieStripPrefix(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestZenmuxExtractCtoken(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "full header", in: "ctoken=tok123; session=xyz", want: "tok123"},
		{name: "ctoken in the middle", in: "session=xyz; ctoken=tok123; other=1", want: "tok123"},
		{name: "bare token", in: "tok123", want: "tok123"},
		{name: "missing", in: "session=xyz", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zenmuxExtractCtoken(tt.in); got != tt.want {
				t.Errorf("zenmuxExtractCtoken(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
