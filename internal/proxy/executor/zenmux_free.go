package executor

// ZenMux Free (Web) — zenmux.ai's free tier over session cookies. ZenMux's
// only browser-reachable chat endpoint is an Anthropic-compatible SSE lane at
// https://zenmux.ai/api/anthropic/v1/messages that reads a `ctoken` from the
// query string and the session cookies as the Cookie header; the response is
// Anthropic-shaped SSE, re-emitted here as OpenAI chunks. Ported from
// cheaprouter's zenmux-free.ts.

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	json "encoding/json/v2"

	"9router/proxy/internal/log"
	"9router/proxy/internal/proxy"
)

var zenmuxFreeChatURL = "https://zenmux.ai/api/anthropic/v1/messages"

// zenmuxExtractCtoken pulls the ctoken cookie out of whatever the operator
// pasted — a bare ctoken value, a ctoken=pair, or a full DevTools blob.
func zenmuxExtractCtoken(raw string) string {
	cookie := webCookieStripPrefix(raw)
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if value, ok := strings.CutPrefix(part, "ctoken="); ok {
			return value
		}
	}
	// A bare token paste (no cookie name at all) is the ctoken itself.
	if !strings.Contains(cookie, "=") {
		return cookie
	}
	return ""
}

// ForwardZenmuxFree relays zenmux.ai's Anthropic-compatible lane.
func ForwardZenmuxFree(w http.ResponseWriter, req *Request) error {
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	rawCookie := webCookieStripPrefix(req.APIKey)
	ctoken := zenmuxExtractCtoken(rawCookie)
	if ctoken == "" {
		return proxy.UpstreamFailure(http.StatusUnauthorized, "zenmux-free: ctoken not found in cookies — export all cookies from zenmux.ai and paste them as the credential")
	}

	var body map[string]any
	if err := json.Unmarshal(req.Body, &body); err != nil || body == nil {
		return proxy.UpstreamFailure(http.StatusBadRequest, "zenmux-free: request body is not a JSON object")
	}
	messages, _ := body["messages"].([]any)
	if len(messages) == 0 {
		return proxy.UpstreamFailure(http.StatusBadRequest, "zenmux-free: missing or empty messages array")
	}

	model, _ := body["model"].(string)
	if strings.TrimSpace(model) == "" {
		model = "deepseek/deepseek-chat"
	}
	maxTokens, _ := body["max_tokens"].(float64)
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	// ZenMux's lane flattens the transcript to a single user turn, like the
	// upstream web client does.
	var systemParts, userParts []string
	for _, raw := range messages {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		role, _ := m["role"].(string)
		text := webCookieMessageText(m["content"])
		if text == "" {
			continue
		}
		switch role {
		case "system":
			systemParts = append(systemParts, text)
		case "user", "assistant":
			userParts = append(userParts, text)
		}
	}
	fullText := strings.Join(userParts, "\n\n")
	if len(systemParts) > 0 {
		fullText = strings.Join(systemParts, "\n\n") + "\n\n" + fullText
	}
	if strings.TrimSpace(fullText) == "" {
		return proxy.UpstreamFailure(http.StatusBadRequest, "zenmux-free: no message text found in the request")
	}

	anthropicBody := map[string]any{
		"model":      model,
		"max_tokens": int64(maxTokens),
		"messages":   []map[string]any{{"role": "user", "content": []map[string]any{{"type": "text", "text": fullText}}}},
		"stream":     true,
	}
	if temperature, ok := body["temperature"].(float64); ok {
		anthropicBody["temperature"] = temperature
	}

	encoded, err := json.Marshal(anthropicBody)
	if err != nil {
		return proxy.UpstreamFailure(http.StatusInternalServerError, "zenmux-free: marshal payload: "+err.Error())
	}

	url := zenmuxFreeChatURL + "?ctoken=" + ctoken
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(encoded)))
	if err != nil {
		return proxy.UpstreamFailure(http.StatusInternalServerError, "zenmux-free: build request: "+err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", webCookieUserAgent)
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Origin", "https://zenmux.ai")
	httpReq.Header.Set("Referer", "https://zenmux.ai/platform/chat")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("chat-request-id", zenmuxRandomHex(16))
	httpReq.Header.Set("x-zenmux-accept-processing", "true, true")
	httpReq.Header.Set("x-zenmux-apikey-source", "subscription")
	httpReq.Header.Set("Cookie", rawCookie)

	log.Info("zenmux-free", "query", "model", model, "stream", req.IsStream, "chars", len(encoded))

	var response *http.Response
	if req.Client != nil {
		response, err = req.Client.Do(httpReq)
	} else {
		response, err = http.DefaultClient.Do(httpReq)
	}
	if err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, "zenmux-free: connection failed: "+err.Error())
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden {
		_, _ = io.Copy(io.Discard, response.Body)
		return proxy.UpstreamFailure(http.StatusUnauthorized, "zenmux-free: cookies expired or invalid — re-export cookies from zenmux.ai")
	}
	if response.StatusCode == http.StatusPaymentRequired {
		_, _ = io.Copy(io.Discard, response.Body)
		return proxy.UpstreamFailure(http.StatusPaymentRequired, "zenmux-free: free-tier quota exhausted")
	}
	if response.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		text := strings.TrimSpace(string(snippet))
		if len(text) > 300 {
			text = text[:300]
		}
		return proxy.UpstreamFailure(response.StatusCode, fmt.Sprintf("zenmux-free returned HTTP %d: %s", response.StatusCode, text))
	}

	responseID := "chatcmpl-zmf-" + zenmuxRandomHex(6)
	created := time.Now().Unix()

	if !req.IsStream {
		text := zenmuxCollectText(response.Body)
		bodyOut := map[string]any{
			"id":      responseID,
			"object":  "chat.completion",
			"created": created,
			"model":   model,
			"choices": []map[string]any{{"index": 0, "message": map[string]any{"role": "assistant", "content": text}, "finish_reason": "stop"}},
			"usage":   map[string]any{"prompt_tokens": 0, "completion_tokens": (len(text) + 3) / 4, "total_tokens": 0},
		}
		encodedOut, err := json.Marshal(bodyOut)
		if err != nil {
			return proxy.UpstreamFailure(http.StatusInternalServerError, "zenmux-free: marshal response: "+err.Error())
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(encodedOut)
		return nil
	}

	hw := proxy.NewHeartbeatWriter(req.Ctx, w, 0)
	defer func() { _ = hw.Close() }()
	flusher := proxy.WriteSSEHeaders(hw)

	emit := func(delta map[string]any, finish any) error {
		chunk := map[string]any{
			"id":      responseID,
			"object":  "chat.completion.chunk",
			"created": created,
			"model":   model,
			"choices": []map[string]any{{"index": 0, "delta": delta, "finish_reason": finish}},
		}
		encodedChunk, err := json.Marshal(chunk)
		if err != nil {
			return err
		}
		if _, err := hw.Write(fmt.Appendf(nil, "data: %s\n\n", encodedChunk)); err != nil {
			return err
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}

	if err := emit(map[string]any{"role": "assistant"}, nil); err != nil {
		return err
	}

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		delta, _ := event["delta"].(map[string]any)
		switch event["type"] {
		case "content_block_delta":
			text, _ := delta["text"].(string)
			if text == "" {
				text, _ = delta["thinking"].(string)
			}
			if text == "" {
				continue
			}
			if err := emit(map[string]any{"content": text}, nil); err != nil {
				return err
			}
		case "message_delta":
			finish, _ := delta["stop_reason"].(string)
			if finish == "" {
				finish = "stop"
			}
			if err := emit(map[string]any{}, finish); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, "zenmux-free: read stream: "+err.Error())
	}

	_, _ = hw.Write([]byte("data: [DONE]\n\n"))
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

// zenmuxCollectText drains an Anthropic-format SSE body and returns the
// concatenated text deltas.
func zenmuxCollectText(body io.Reader) string {
	var sb strings.Builder
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		delta, _ := event["delta"].(map[string]any)
		if event["type"] != "content_block_delta" || delta == nil {
			continue
		}
		text, _ := delta["text"].(string)
		if text == "" {
			text, _ = delta["thinking"].(string)
		}
		sb.WriteString(text)
	}
	return sb.String()
}

func zenmuxRandomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
