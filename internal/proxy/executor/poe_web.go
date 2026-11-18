package executor

// Poe Web (Subscription) — poe.com chat over the session cookie (`p-b`).
// Poe exposes no public chat HTTP lane; the web app talks to a GraphQL
// endpoint (ChatViewQuery) that answers one message object, so the stream is
// always a single chunk. Ported from cheaprouter's poe-web.ts.

import (
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

var poeWebGQLURL = "https://www.poe.com/api/gql_POST"

const poeWebOrigin = "https://www.poe.com"

// poeWebModelMap translates CheapRouter-style model ids onto Poe bot names.
var poeWebModelMap = map[string]string{
	"gpt-4o":            "GPT-4o",
	"gpt-4-turbo":       "GPT-4-Turbo",
	"claude-3.5-sonnet": "Claude-3.5-Sonnet",
	"claude-3-opus":     "Claude-3-Opus",
	"gemini-2.0-flash":  "Gemini-2.0-Flash",
	"llama-3-70b":       "Llama-3-70B",
	"mixtral-8x22b":     "Mixtral-8x22B",
	"poe-default":       "Assistant",
}

// poeExtractPBCookie pulls the p-b cookie value out of whatever the operator
// pasted — a bare token, a p-b=pair, or a full DevTools blob.
func poeExtractPBCookie(raw string) string {
	cookie := webCookieStripPrefix(raw)
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if value, ok := strings.CutPrefix(part, "p-b="); ok {
			return value
		}
	}
	if !strings.Contains(cookie, "=") {
		return cookie
	}
	return cookie
}

// ForwardPoeWeb relays poe.com's GraphQL chat lane.
func ForwardPoeWeb(w http.ResponseWriter, req *Request) error {
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	pb := poeExtractPBCookie(req.APIKey)
	if pb == "" {
		return proxy.UpstreamFailure(http.StatusUnauthorized, "poe-web: a p-b cookie is required — paste the p-b cookie value from poe.com (DevTools → Application → Cookies → p-b) into the connection")
	}

	var body map[string]any
	if err := json.Unmarshal(req.Body, &body); err != nil || body == nil {
		return proxy.UpstreamFailure(http.StatusBadRequest, "poe-web: request body is not a JSON object")
	}
	messages, _ := body["messages"].([]any)
	if len(messages) == 0 {
		return proxy.UpstreamFailure(http.StatusBadRequest, "poe-web: missing or empty messages array")
	}

	model, _ := body["model"].(string)
	if strings.TrimSpace(model) == "" {
		model = "poe-default"
	}
	botName := poeWebModelMap[model]
	if botName == "" {
		botName = model
	}

	prompt := ""
	for _, raw := range messages {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if role, _ := m["role"].(string); role == "user" {
			prompt = webCookieMessageText(m["content"])
		}
	}
	if strings.TrimSpace(prompt) == "" {
		return proxy.UpstreamFailure(http.StatusBadRequest, "poe-web: no user message found in the request")
	}

	gqlBody := map[string]any{
		"operationName": "ChatViewQuery",
		"query":         "query ChatViewQuery($bot: String!, $query: String!) {\n  chatWithBot(bot: $bot, query: $query) {\n    messageId\n    text\n    state\n  }\n}",
		"variables":     map[string]any{"bot": botName, "query": prompt},
	}

	encoded, err := json.Marshal(gqlBody)
	if err != nil {
		return proxy.UpstreamFailure(http.StatusInternalServerError, "poe-web: marshal payload: "+err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, poeWebGQLURL, strings.NewReader(string(encoded)))
	if err != nil {
		return proxy.UpstreamFailure(http.StatusInternalServerError, "poe-web: build request: "+err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", webCookieUserAgent)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Referer", poeWebOrigin+"/")
	httpReq.Header.Set("Origin", poeWebOrigin)
	httpReq.Header.Set("Cookie", "p-b="+pb)

	log.Info("poe-web", "query", "model", model, "bot", botName, "chars", len(prompt))

	var response *http.Response
	if req.Client != nil {
		response, err = req.Client.Do(httpReq)
	} else {
		response, err = http.DefaultClient.Do(httpReq)
	}
	if err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, "poe-web: connection failed: "+err.Error())
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		text := strings.TrimSpace(string(snippet))
		if len(text) > 300 {
			text = text[:300]
		}
		return proxy.UpstreamFailure(response.StatusCode, fmt.Sprintf("poe-web returned HTTP %d: %s", response.StatusCode, text))
	}

	raw, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, "poe-web: read response: "+err.Error())
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, "poe-web: response is not JSON")
	}
	text := ""
	if inner, ok := data["data"].(map[string]any); ok {
		if chat, ok := inner["chatWithBot"].(map[string]any); ok {
			text, _ = chat["text"].(string)
		}
	}

	responseID := "chatcmpl-poe-" + zenmuxRandomHex(6)
	created := time.Now().Unix()

	if !req.IsStream {
		bodyOut := map[string]any{
			"id":      responseID,
			"object":  "chat.completion",
			"created": created,
			"model":   model,
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": text}, "finish_reason": "stop"}},
		}
		encodedOut, err := json.Marshal(bodyOut)
		if err != nil {
			return proxy.UpstreamFailure(http.StatusInternalServerError, "poe-web: marshal response: "+err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(encodedOut)
		return nil
	}

	hw := proxy.NewHeartbeatWriter(req.Ctx, w, 0)
	defer func() { _ = hw.Close() }()
	flusher := proxy.WriteSSEHeaders(hw)

	chunk := map[string]any{
		"id":      responseID,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{{"index": 0, "delta": map[string]any{"content": text}, "finish_reason": nil}},
	}
	encodedChunk, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	_, _ = hw.Write(fmt.Appendf(nil, "data: %s\n\n", encodedChunk))
	_, _ = hw.Write([]byte("data: [DONE]\n\n"))
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}
