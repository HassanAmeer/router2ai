package executor

// Executors for cheaprouter catalog entries wired into this gateway. The
// vast majority route through ForwardOpenAI against an OpenAI-compatible
// upstream; gitlab-duo/glm(t)/amazon-q ride the same families already here.
// Everything else not wired (26 providers whose wire protocol is not a plain
// OpenAI chat endpoint) resolves to ForwardCheaprouterPending — a 503 with a
// clear message rather than a misdirected replay.

import (
	"net/http"

	"9router/proxy/internal/proxy"
)

// ForwardCheaprouterPending returns the executor that reports the provider as
// unavailable in this build.
func ForwardCheaprouterPending(providerID string) Executor {
	return func(w http.ResponseWriter, req *Request) error {
		return proxy.UpstreamFailure(http.StatusServiceUnavailable,
			providerID+": the executor is not ported to Go yet — the catalog entry and credential storage are in place, but requests fail with 503 until the port lands")
	}
}

func init() {
	Register("gitlab-duo", func() Executor { return ForwardOpenAI })
	Register("glmt", func() Executor { return ForwardZai })
	Register("amazon-q", func() Executor { return ForwardKiro })
	for _, pending := range cheaprouterPendingProviders {
		provider := pending
		Register(provider, func() Executor { return ForwardCheaprouterPending(provider) })
	}
}

// cheaprouterPendingProviders are catalog entries whose wire protocol is not
// portable today (mark-and-sweep OAuth login flows, app-server bridges, media
// lanes without executors ported yet, and providers whose BaseURL+protocol
// must be wired first). Keep in sync with the Notice on their registry entries.
var cheaprouterPendingProviders = []string{
	"agy", "anysearch-search", "azure-ai", "azure-openai",
	"cheaperinference", "clova-studio", "codex-app-server", "codex-cloud",
	"command-code", "context7", "cursor-api", "datarobot",
	"deepai", "devin", "devin-cli-agentic", "devin-desktop",
	"docker-model-runner", "exa-search", "fishaudio", "ghe-copilot",
	"gladia", "google-pse-search", "haiper", "ideogram",
	"jules", "leonardo", "linkup-search", "magnific",
	"mixedbread", "moonshot", "nimble-search", "nomic",
	"oci", "opencode-bypass", "perplexity-search", "rev-ai",
	"sap", "searchapi-search", "searxng-search", "segmind",
	"serper-search", "soniox", "speechmatics", "tavily-search",
	"triton", "udio", "watsonx", "x-search",
	"xai-oauth", "xquik-search", "youcom-search", "zed-hosted",
}
