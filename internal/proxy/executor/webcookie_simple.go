package executor

// venice.ai and v0.dev share one wire shape: a session-cookie-authenticated
// POST to a site-internal chat endpoint that answers either one JSON object
// (choices[0].message.content) or an OpenAI-shaped SSE stream. Ported from
// cheaprouter's venice-web.ts / v0-vercel-web.ts. One helper drives both;
// the per-provider difference is the endpoint and the max_tokens default.

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	json "encoding/json/v2"

	"9router/proxy/internal/log"
	"9router/proxy/internal/proxy"
)

var webCookieUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/149.0.0.0 Safari/537.36"

// webCookieStripPrefix removes the decorations a pasted credential
// sometimes carries: a bearer prefix or a "cookie:" label.
func webCookieStripPrefix(raw string) string {
	trimmed := strings.TrimSpace(raw)
	lower := strings.ToLower(trimmed)
	if value, ok := strings.CutPrefix(lower, "bearer "); ok {
		_ = value
		trimmed = trimmed[len("bearer "):]
	}
	lower = strings.ToLower(strings.TrimSpace(trimmed))
	if value, ok := strings.CutPrefix(lower, "cookie:"); ok {
		_ = value
		trimmed = strings.TrimSpace(trimmed[len("cookie:"):])
	}
	return strings.TrimSpace(trimmed)
}

// webCookieChatSpec describes one cookie-session endpoint that speaks
// near-OpenAI chat JSON/SSE (venice.ai, v0.dev).
type webCookieChatSpec struct {
	Provider         string
	ChatURL          string
	Origin           string
	IDPrefix         string
	DefaultModel     string
	IncludeMaxTokens bool
}

var veniceWebSpec = webCookieChatSpec{
	Provider:         "venice-web",
	ChatURL:          "https://venice.ai/api/chat",
	Origin:           "https://venice.ai",
	IDPrefix:         "chatcmpl-ven-",
	DefaultModel:     "venice-default",
	IncludeMaxTokens: true,
}

var v0VercelWebSpec = webCookieChatSpec{
	Provider:     "v0-vercel-web",
	ChatURL:      "https://v0.dev/api/chat",
	Origin:       "https://v0.dev",
	IDPrefix:     "chatcmpl-v0-",
	DefaultModel: "v0-default",
}

// ForwardVeniceWeb relays venice.ai's cookie-gated chat endpoint.
func ForwardVeniceWeb(w http.ResponseWriter, req *Request) error {
	return forwardWebCookieOpenAIChat(veniceWebSpec, w, req)
}

// ForwardV0VercelWeb relays v0.dev's cookie-gated chat endpoint.
func ForwardV0VercelWeb(w http.ResponseWriter, req *Request) error {
	return forwardWebCookieOpenAIChat(v0VercelWebSpec, w, req)
}

func forwardWebCookieOpenAIChat(spec webCookieChatSpec, w http.ResponseWriter, req *Request) error {
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	var body map[string]any
	if err := json.Unmarshal(req.Body, &body); err != nil || body == nil {
		return proxy.UpstreamFailure(http.StatusBadRequest, spec.Provider+": request body is not a JSON object")
	}
	messages, _ := body["messages"].([]any)
	if len(messages) == 0 {
		return proxy.UpstreamFailure(http.StatusBadRequest, spec.Provider+": missing or empty messages array")
	}

	model, _ := body["model"].(string)
	if strings.TrimSpace(model) == "" {
		model = spec.DefaultModel
	}

	flat := make([]map[string]any, 0, len(messages))
	for _, raw := range messages {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		content := webCookieMessageText(m["content"])
		flat = append(flat, map[string]any{"role": role, "content": content})
	}

	upstreamBody := map[string]any{
		"messages": flat,
		"model":    model,
		"stream":   req.IsStream,
	}
	if spec.IncludeMaxTokens {
		maxTokens, _ := body["max_tokens"].(float64)
		if maxTokens <= 0 {
			maxTokens = 4096
		}
		upstreamBody["max_tokens"] = int64(maxTokens)
	}

	cookie := webCookieStripPrefix(req.APIKey)
	if cookie == "" {
		return proxy.UpstreamFailure(http.StatusUnauthorized, spec.Provider+": a session cookie is required — paste the full Cookie header from the site into the connection")
	}

	encoded, err := json.Marshal(upstreamBody)
	if err != nil {
		return proxy.UpstreamFailure(http.StatusInternalServerError, spec.Provider+": marshal payload: "+err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, spec.ChatURL, strings.NewReader(string(encoded)))
	if err != nil {
		return proxy.UpstreamFailure(http.StatusInternalServerError, spec.Provider+": build request: "+err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", webCookieUserAgent)
	httpReq.Header.Set("Referer", spec.Origin+"/")
	httpReq.Header.Set("Origin", spec.Origin)
	if req.IsStream {
		httpReq.Header.Set("Accept", "text/event-stream")
	} else {
		httpReq.Header.Set("Accept", "application/json")
	}
	httpReq.Header.Set("Cookie", cookie)

	log.Info(spec.Provider, "query", "model", model, "stream", req.IsStream, "chars", len(encoded))

	var response *http.Response
	if req.Client != nil {
		response, err = req.Client.Do(httpReq)
	} else {
		response, err = http.DefaultClient.Do(httpReq)
	}
	if err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, spec.Provider+": connection failed: "+err.Error())
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		text := strings.TrimSpace(string(snippet))
		if len(text) > 300 {
			text = text[:300]
		}
		msg := fmt.Sprintf("%s returned HTTP %d", spec.Provider, response.StatusCode)
		if text != "" {
			msg += ": " + text
		}
		return proxy.UpstreamFailure(response.StatusCode, msg)
	}

	responseID := spec.IDPrefix + fmt.Sprintf("%d", time.Now().UnixNano())
	if req.IsStream {
		return webCookieRelaySSE(spec, w, req, response.Body, model, responseID)
	}
	return webCookieCollectToClient(spec, w, req, response.Body, model, responseID)
}

// webCookieMessageText flattens an OpenAI message content (string, or the
// array-of-parts shape) into plain text.
func webCookieMessageText(content any) string {
	switch value := content.(type) {
	case string:
		return value
	case []any:
		var parts []string
		for _, raw := range value {
			part, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok && text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// webCookieCollectToClient folds the upstream JSON body into a chat.completion.
func webCookieCollectToClient(spec webCookieChatSpec, w http.ResponseWriter, req *Request, upstream io.Reader, model string, responseID string) error {
	raw, err := io.ReadAll(io.LimitReader(upstream, 4<<20))
	if err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, spec.Provider+": read response: "+err.Error())
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, spec.Provider+": response is not JSON")
	}
	message := webCookieAssistantMessage(data)
	body := map[string]any{
		"id":      responseID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []map[string]any{{"index": 0, "message": message, "finish_reason": "stop"}},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return proxy.UpstreamFailure(http.StatusInternalServerError, spec.Provider+": marshal response: "+err.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
	return nil
}

// webCookieAssistantMessage extracts the assistant message (content and, when
// present, reasoning_content) from both upstream response shapes.
func webCookieAssistantMessage(data map[string]any) map[string]any {
	var message map[string]any
	if choices, ok := data["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			message, _ = choice["message"].(map[string]any)
		}
	}
	if message == nil {
		message = map[string]any{}
	}
	content, _ := message["content"].(string)
	if content == "" {
		content, _ = data["content"].(string)
	}
	out := map[string]any{"role": "assistant", "content": content}
	reasoning, _ := message["reasoning_content"].(string)
	if reasoning == "" {
		reasoning, _ = data["reasoning_content"].(string)
	}
	if reasoning != "" {
		out["reasoning_content"] = reasoning
	}
	return out
}

// webCookieRelaySSE re-emits the upstream SSE stream as OpenAI chunks.
func webCookieRelaySSE(spec webCookieChatSpec, w http.ResponseWriter, req *Request, upstream io.Reader, model string, responseID string) error {
	hw := proxy.NewHeartbeatWriter(req.Ctx, w, 0)
	defer func() { _ = hw.Close() }()
	flusher := proxy.WriteSSEHeaders(hw)

	created := time.Now().Unix()
	scanner := bufio.NewScanner(upstream)
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)

	emit := func(delta map[string]any) error {
		chunk := map[string]any{
			"id":      responseID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []map[string]any{{"index": 0, "delta": delta, "finish_reason": nil}},
		}
		encoded, err := json.Marshal(chunk)
		if err != nil {
			return err
		}
		if _, err := hw.Write(fmt.Appendf(nil, "data: %s\n\n", encoded)); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(data), &parsed); err != nil {
			continue // unparseable chunk — skip, like the TS executors do
		}
		var delta map[string]any
		if choices, ok := parsed["choices"].([]any); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]any); ok {
				delta, _ = choice["delta"].(map[string]any)
			}
		}
		if delta == nil {
			continue
		}
		outDelta := map[string]any{}
		if reasoning, ok := delta["reasoning_content"].(string); ok && reasoning != "" {
			outDelta["reasoning_content"] = reasoning
		}
		if content, ok := delta["content"].(string); ok && content != "" {
			outDelta["content"] = content
		}
		if len(outDelta) == 0 {
			continue
		}
		if err := emit(outDelta); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, spec.Provider+": read stream: "+err.Error())
	}

	_, _ = hw.Write([]byte("data: [DONE]\n\n"))
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}
