package executor

// Web & cookie providers whose cheaprouter executors are not ported to Go
// yet. The catalog entry, auth flow and model catalog are in place; the
// request path returns a clear 503 instead of silently forwarding the
// credential to an endpoint it does not belong to, or hanging in the generic
// OpenAI passthrough. Each id is registered once in init.go.
//
// Ported providers (venice-web, v0-vercel-web, poe-web, tencent-aistudio-web,
// zenmux-free) and providers already served by the shared OpenAI passthrough
// (the noauth OpenAI-compatible lanes) are NOT listed here.

import (
	"net/http"

	"9router/proxy/internal/proxy"
)

// webCookiePendingProviders lists the catalogued providers awaiting their Go
// executor port. Keep this in sync with the catalog entries flagged as
// pending in provider_registry.go.
var webCookiePendingProviders = []string{
	// Web-cookie chat lanes (Playwright / TLS-impersonation / PoW / WS / signed
	// requests — ports tracked individually).
	"chatgpt-web",
	"chatgpt-web-codex",
	"muse-spark-web",
	"claude-web",
	"copilot-web",
	"copilot-m365-web",
	"t3-web",
	"inner-ai",
	"adapta-web",
	"lmarena",
	"yuanbao-web",
	"doubao-web",
	"blackbox-web",
	"huggingchat",
	"kimi-web",
	"tinycms-web",
	"zai-web",
	"notion-web",
	"hyperagent",
	"conol-web",
	"twinmind",
	"syntx",
	"maxai",
	"uc",
	"chatplayground",
	"promptql",
	"genspark-web",
	// Media lane pending: Adobe Firefly (image/video).
	"adobe-firefly",
	// Anonymous/noauth lanes that need non-HTTP-replay transports
	// (JS-VM challenge, Playwright browser protocol, local stdio bridge).
	"duckduckgo-web",
	"cloudflare-playground",
	"auggie",
}

// ForwardWebCookiePending returns the executor that reports the provider as
// not ported yet.
func ForwardWebCookiePending(providerID string) Executor {
	return func(w http.ResponseWriter, req *Request) error {
		return proxy.UpstreamFailure(http.StatusServiceUnavailable,
			providerID+": the web-cookie executor is not ported to Go yet — the catalog entry and credential storage are in place, but requests fail with 503 until the port lands")
	}
}
