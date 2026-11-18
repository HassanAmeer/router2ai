package providers

import (
	"slices"
	"testing"
)

// Every web-cookie and no-auth provider ported from cheaprouter's catalog.
// Each row pins the contract cheaprouter ships: the catalog renders, the wire
// config points somewhere, the aliases round-trip, and the model catalog is
// the provider's own.
func TestCheaprouterWebCookieAndNoAuthProviders(t *testing.T) {
	type row struct {
		id       string
		name     string
		category Category
		aliases  []string // aliases that resolve to id (both directions)
		models   []string // empty means "no static catalog is published"
		noAuth   bool
	}
	rows := []row{
		// Web & cookie — ported with Go executors.
		{"venice-web", "Venice Web (Privacy)", CategoryWebCookie, []string{"ven"}, []string{"venice-default"}, false},
		{"v0-vercel-web", "v0 Vercel Web (Code Gen)", CategoryWebCookie, nil, []string{"v0-default"}, false},
		{"poe-web", "Poe Web (Subscription)", CategoryWebCookie, []string{"poe"}, []string{"gpt-4o", "poe-default"}, false},
		{"tencent-aistudio-web", "Tencent AI Studio (Free)", CategoryWebCookie, []string{"tasw"}, []string{"hy3-g", "hunyuan-default", "hunyuan-3d"}, false},
		{"zenmux-free", "ZenMux Free (Web)", CategoryWebCookie, []string{"zmf"}, []string{"deepseek/deepseek-chat", "z-ai/glm-4.7-flash-free"}, false},
		// Web & cookie — catalog entry with a pending executor.
		{"chatgpt-web", "ChatGPT Web (Clean Room)", CategoryWebCookie, nil, []string{"gpt-5-6", "gpt-5-5-pro"}, false},
		{"chatgpt-web-codex", "ChatGPT Web (Codex)", CategoryWebCookie, []string{"cgpt-codex"}, []string{"luna", "pro"}, false},
		{"muse-spark-web", "Muse Spark Web (Meta AI)", CategoryWebCookie, []string{"ms-web"}, []string{"muse-spark", "muse-spark-thinking"}, false},
		{"claude-web", "Claude Web", CategoryWebCookie, []string{"cw"}, []string{"claude-fable-5-1", "claude-opus-5"}, false},
		{"copilot-web", "Microsoft Copilot Web", CategoryWebCookie, []string{"copilot"}, []string{"copilot-pro"}, false},
		{"copilot-m365-web", "Microsoft 365 Copilot (BizChat)", CategoryWebCookie, []string{"m365copilot"}, []string{"copilot-m365"}, false},
		{"t3-web", "t3.chat (Pro/Free)", CategoryWebCookie, []string{"t3chat"}, []string{"gpt-5", "claude-opus-4"}, false},
		{"inner-ai", "Inner.ai (Subscription)", CategoryWebCookie, []string{"in-ai"}, []string{"gpt-4o", "o3"}, false},
		{"adapta-web", "Adapta.org (Adapta One Web)", CategoryWebCookie, []string{"adp-web"}, []string{"adapta-one", "adapta-llama"}, false},
		{"lmarena", "Arena (Free)", CategoryWebCookie, []string{"lma"}, []string{"claude-sonnet-5", "grok-4.3/text"}, false},
		{"yuanbao-web", "Tencent Yuanbao (Free)", CategoryWebCookie, []string{"ybw"}, []string{"deepseek-v3", "hunyuan-t1"}, false},
		{"huggingchat", "HuggingChat (Free)", CategoryWebCookie, nil, []string{"zai-org/GLM-5.2", "deepseek-ai/DeepSeek-V4-Pro"}, false},
		{"kimi-web", "Kimi Web", CategoryWebCookie, nil, []string{"k3", "k2d6"}, false},
		{"doubao-web", "Dola Web (ByteDance)", CategoryWebCookie, []string{"db"}, []string{"dola-speed", "dola-pro"}, false},
		{"tinycms-web", "TinyCMS Web (Free/Sub)", CategoryWebCookie, []string{"tcw"}, []string{"claude-fable-5", "glm-5.2"}, false},
		{"zai-web", "Z.ai Web", CategoryWebCookie, []string{"zw"}, []string{"glm-5.3-flash", "glm-5.2"}, false},
		{"promptql", "PromptQL (Unofficial/Experimental)", CategoryWebCookie, []string{"pql"}, []string{"gpt-5.6-sol", "kimi-k3"}, false},
		{"notion-web", "Notion AI Web (Unofficial/Experimental)", CategoryWebCookie, []string{"nw"}, []string{"notion-ai", "opus-5"}, false},
		{"adobe-firefly", "Adobe Firefly (Image/Video)", CategoryWebCookie, []string{"firefly"}, nil, false},
		{"hyperagent", "HyperAgent (Unofficial/Experimental)", CategoryWebCookie, []string{"ha"}, []string{"fable-latest", "sonnet-latest"}, false},
		{"notrack-web", "NoTrack Web (Free)", CategoryWebCookie, []string{"ntw"}, []string{"notrack-c"}, false},
		{"conol-web", "Conol (Unofficial/Experimental)", CategoryWebCookie, []string{"cnl"}, []string{"claude-sonnet-5", "z-ai/glm-5.2"}, false},
		{"twinmind", "Twinmind (Unofficial/Experimental)", CategoryWebCookie, []string{"tm"}, []string{"auto", "claude-sonnet-5"}, false},
		{"syntx", "SYNTX.ai (Unofficial/Experimental)", CategoryWebCookie, []string{"stx"}, []string{"claude-sonnet-5", "glm-5.2"}, false},
		{"maxai", "MaxAI", CategoryWebCookie, []string{"mx"}, []string{"gpt-5.6-luna", "deepseek-r1"}, false},
		{"uc", "UC (uncensored.com)", CategoryWebCookie, []string{"ucn"}, []string{"gpt-5.5", "grok-4-20"}, false},
		{"chatplayground", "ChatPlayground", CategoryWebCookie, []string{"cpl"}, []string{"gpt-5.5", "sonar-pro"}, false},
		{"genspark-web", "Genspark AI Web", CategoryWebCookie, []string{"genspark"}, nil, false},
		{"blackbox-web", "Blackbox Web (Subscription)", CategoryWebCookie, []string{"bb-web"}, []string{"gpt-4-turbo", "claude-3-opus"}, false},
		// No-auth lanes.
		{"duckduckgo-web", "DuckDuckGo AI Chat", CategoryFree, []string{"ddgw"}, []string{"gpt-5.4-mini", "tinfoil/gemma4-31b"}, true},
		{"cloudflare-playground", "Cloudflare AI Playground", CategoryFree, []string{"cfp"}, []string{"zai-org/glm-5.2"}, true},
		{"aihorde", "AI Horde", CategoryFree, []string{"horde"}, []string{"google/gemma-4-31b"}, true},
		{"pollinations-text", "Pollinations AI (Text)", CategoryFree, []string{"pol-text"}, []string{"openai-fast", "deepseek"}, true},
		{"uncloseai", "UncloseAI", CategoryFree, []string{"unc"}, []string{"turboderp/Qwen3.8-27B-exl3"}, true},
		{"antseed", "Antseed CLI", CategoryFree, nil, []string{"glm-5.3-flash", "deepseek-v4"}, true},
		{"auggie", "Augment (Auggie CLI)", CategoryFree, []string{"aug"}, []string{"opus4.8", "sonnet4.6"}, true},
	}

	for _, row := range rows {
		t.Run(row.id, func(t *testing.T) {
			meta, ok := GetProviderMeta(row.id)
			if !ok {
				t.Fatalf("GetProviderMeta(%q) not found — the provider would not render in the catalog", row.id)
			}
			if meta.ID != row.id {
				t.Errorf("meta.ID = %q, want %q", meta.ID, row.id)
			}
			if meta.Category != row.category {
				t.Errorf("meta.Category = %q, want %q", meta.Category, row.category)
			}
			if meta.Name != row.name {
				t.Errorf("meta.Name = %q, want %q", meta.Name, row.name)
			}
			if meta.NoAuth != row.noAuth {
				t.Errorf("meta.NoAuth = %v, want %v", meta.NoAuth, row.noAuth)
			}
			cfg, ok := KnownProviders[row.id]
			if !ok {
				t.Fatalf("KnownProviders has no entry for %q — a connection could not route", row.id)
			}
			if cfg.BaseURL == "" {
				t.Errorf("KnownProviders[%q].BaseURL is empty", row.id)
			}

			for _, alias := range row.aliases {
				if got := RegistryAliases[row.id]; got != alias {
					t.Errorf("RegistryAliases[%q] = %q, want %q", row.id, got, alias)
				}
				if got := ResolveAlias(alias); got != row.id {
					t.Errorf("ResolveAlias(%q) = %q, want %q", alias, got, row.id)
				}
			}
			if len(row.aliases) == 0 {
				if got := GetProviderAlias(row.id); got != row.id {
					t.Errorf("GetProviderAlias(%q) = %q, want the id itself", row.id, got)
				}
			}

			models := GetProviderModels(row.id)
			if len(row.models) == 0 {
				if len(models) != 0 {
					t.Errorf("GetProviderModels(%q) = %d models, want none published", row.id, len(models))
				}
				return
			}
			for _, model := range row.models {
				if !slices.Contains(models, model) {
					t.Errorf("GetProviderModels(%q) is missing %q", row.id, model)
				}
			}
		})
	}
}

// No alias may be claimed by two providers, and none of the cheaprouter
// web-cookie aliases may collide with the pre-existing tables.
func TestCheaprouterAliasesAreUnique(t *testing.T) {
	seen := map[string]string{}
	for alias, id := range ProviderAliasMap {
		if prev, dup := seen[alias]; dup {
			t.Errorf("alias %q claimed by both %q and %q", alias, prev, id)
		}
		seen[alias] = id
	}
	for id, alias := range RegistryAliases {
		if alias == "" || alias == id {
			// Identity aliases are published as the model prefix and resolve to
			// the id itself, so ProviderAliasMap intentionally has no row.
			continue
		}
		if id2, ok := ProviderAliasMap[alias]; !ok || id2 != id {
			t.Errorf("RegistryAliases[%q] = %q but ProviderAliasMap[%q] = %q — the two alias tables disagree", id, alias, alias, id2)
		}
	}
}
