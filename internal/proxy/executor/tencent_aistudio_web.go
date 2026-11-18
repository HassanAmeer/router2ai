package executor

// Tencent AI Studio (Free) — aistudio.tencent.ai's web chat over the session
// Cookie header. The site answers on an internal lane that is not
// OpenAI-shaped; the upstream body is streamed to the client verbatim, like
// the cheaprouter executor does. Model ids map onto the site's internal
// Hunyuan codenames.

import (
	"bytes"
	"context"
	"io"
	"net/http"

	json "encoding/json/v2"

	"9router/proxy/internal/log"
	"9router/proxy/internal/proxy"
)

var tencentAIStudioBase = "https://aistudio.tencent.ai"

// tencentAIStudioModelMap translates client-facing ids onto internal model
// codenames. Unknown ids fall back to HunyuanDefault, like the TS executor.
var tencentAIStudioModelMap = map[string]string{
	"hy3-g":           "HunyuanDefault",
	"hunyuan-default": "HunyuanDefault",
	"hunyuan-3d":      "Hunyuan3D",
}

// ForwardTencentAIStudioWeb relays aistudio.tencent.ai's cookie-gated lane.
func ForwardTencentAIStudioWeb(w http.ResponseWriter, req *Request) error {
	ctx := req.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	cookie := webCookieStripPrefix(req.APIKey)
	if cookie == "" {
		return proxy.UpstreamFailure(http.StatusUnauthorized, "tencent-aistudio-web: a cookie is required — log in to aistudio.tencent.ai and paste the full Cookie header into the connection")
	}

	var body map[string]any
	if err := json.Unmarshal(req.Body, &body); err != nil || body == nil {
		return proxy.UpstreamFailure(http.StatusBadRequest, "tencent-aistudio-web: request body is not a JSON object")
	}
	messages, _ := body["messages"].([]any)
	if len(messages) == 0 {
		return proxy.UpstreamFailure(http.StatusBadRequest, "tencent-aistudio-web: missing or empty messages array")
	}

	model, _ := body["model"].(string)
	upstreamModel := tencentAIStudioModelMap[model]
	if upstreamModel == "" {
		upstreamModel = "HunyuanDefault"
	}

	upstreamBody := map[string]any{"model": upstreamModel, "messages": messages}
	encoded, err := json.Marshal(upstreamBody)
	if err != nil {
		return proxy.UpstreamFailure(http.StatusInternalServerError, "tencent-aistudio-web: marshal payload: "+err.Error())
	}

	chatURL := tencentAIStudioBase + "/api/chat/" + upstreamModel
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, chatURL, bytes.NewReader(encoded))
	if err != nil {
		return proxy.UpstreamFailure(http.StatusInternalServerError, "tencent-aistudio-web: build request: "+err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Cookie", cookie)
	httpReq.Header.Set("Origin", tencentAIStudioBase)
	httpReq.Header.Set("Referer", tencentAIStudioBase+"/")
	httpReq.Header.Set("User-Agent", webCookieUserAgent)

	log.Info("tencent-aistudio-web", "query", "model", model, "upstream", upstreamModel, "messages", len(messages))

	var response *http.Response
	if req.Client != nil {
		response, err = req.Client.Do(httpReq)
	} else {
		response, err = http.DefaultClient.Do(httpReq)
	}
	if err != nil {
		return proxy.UpstreamFailure(http.StatusBadGateway, "tencent-aistudio-web: connection failed: "+err.Error())
	}
	defer func() { _ = response.Body.Close() }()

	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	w.Header().Set("Content-Type", contentType)
	if cache := response.Header.Get("Cache-Control"); cache != "" {
		w.Header().Set("Cache-Control", cache)
	}
	w.WriteHeader(response.StatusCode)

	flusher, canFlush := w.(http.Flusher)
	buf := make([]byte, 32<<10)
	for {
		n, readErr := response.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			// The response status line is already committed; all we can do is
			// log the truncation and let the client see a cut stream.
			log.Info("tencent-aistudio-web", "stream cut", "err", readErr.Error())
			return nil
		}
	}
}
