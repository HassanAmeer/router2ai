### Web & cookie providers from cheaprouter + cheaprouter sidebar flow

- **Providers tab — Web & Cookie section completed.** Every web/cookie and
  no-auth provider in cheaprouter's catalog (`WEB_COOKIE_PROVIDERS` /
  `NOAUTH_PROVIDERS`, `web-cookie.ts` / `noauth.ts`) is now listed under
  **Web Cookie Providers** / **Free (No Auth)** in the Providers tab, with the
  cheaprouter auth hints, websites, aliases and static model catalogs ported
  into `internal/providers/provider_registry.go`, `providers.go`
  (`KnownProviders`), `registry_models.go` (`ProviderModels`),
  `registry_aliases.go` and `aliases.go`.
- **Executors.** Five providers gained real Go executors
  (`internal/proxy/executor/{webcookie_simple,poe_web,tencent_aistudio_web,zenmux_free}.go`):
  `venice-web`, `v0-vercel-web` (shared OpenAI-shaped cookie-replay helper),
  `poe-web` (GraphQL `ChatViewQuery`), `tencent-aistudio-web` (Hunyuan
  codename lane, raw stream passthrough) and `zenmux-free` (Anthropic SSE lane
  re-framed as OpenAI chunks). The no-auth OpenAI-compatible lanes
  (`aihorde` with the documented anonymous `0000000000` key,
  `pollinations-text`, `uncloseai`, `antseed`) route through the shared
  passthrough with no executor code. DeepSeek Web, Gemini Web, Grok Web and
  Perplexity Web implementations are unchanged.
- **Pending executors answer 503, never a misdirected replay.** The remaining
  cookie/no-auth providers (`chatgpt-web`, `claude-web`, `copilot-web`,
  `copilot-m365-web`, `t3-web`, `inner-ai`, `adapta-web`, `lmarena`,
  `yuanbao-web`, `blackbox-web`, `huggingchat`, `kimi-web`, `tinycms-web`,
  `zai-web`, `notion-web`, `hyperagent`, `conol-web`, `twinmind`, `syntx`,
  `maxai`, `uc`, `chatplayground`, `promptql`, `genspark-web`,
  `adobe-firefly`, `duckduckgo-web`, `cloudflare-playground`, `auggie`) are
  visible in the catalog with a notice on the card ("the Go executor is not
  ported yet") and resolve to `ForwardWebCookiePending`
  (`webcookie_pending.go`), which answers HTTP 503 with an explicit message
  instead of leaking the pasted credential to the wrong lane. Each needs its
  own port tracked individually — several require Playwright/TLS-impersonation/
  PoW/WebSocket transports that do not translate cheaply (see
  `open-sse/executors/*.ts` sizes, e.g. muse-spark-web 1.4k lines).
- **CheapRouter providers, complete.** Every LLM provider catalogued in
  cheaprouter (359 registry entries; aiohttp-excluded media lanes and the
  internal `auto`/`9router`/`cliproxyapi` system entries excluded) is now in
  the Providers tab — 261 wired entries (OpenAI-compatible passthrough
  against the provider's real chat endpoint, or the gitlab-duo/glm(t)/
  amazon-q families already here) plus 53 catalog entries whose wire protocol
  is not portable (oauth-only flows, app-server bridges, search/audio media
  lanes, …) marked with a Notice and routed to a 503 executor. Display
  metadata (name, alias, icon name, color, auth hint, service kinds, static
  model lists for the antigravity/kiro families) comes from cheaprouter's
  catalog data; alias collisions with 9router-go's existing aliases were
  skipped; 74 provider SVGs were copied from cheaprouter and the rest got
  generated initials badges. Delete this file's detail section when the
  503-lane executors land.
- **CheapRouter sidebar parity.** The dashboard sidebar now mirrors the
  cheaprouter flow: a headerless top-level **Analytics** entry (new
  `HomeView.svelte` overview landing page — stat cards, provider health,
  quick actions; `/` and `/dashboard` land here), then the **CheapProxy** group
  (Providers, Combos, **Pipe Models** — new `ManageModelsView.svelte` grouped
  provider/model catalog with search and filters; **Limits Bank** — the Plans
  page, relabeled `Limits Bank` and now visible by default; Quota Tracker;
  **API Keys** — new standalone `ApiKeysView.svelte` hosting `ApiKeysTable`
  plus an inline single-key create form; API Endpoints; Usage; Token Saver),
  the **API** group (**API Docs** — new `ApiDocsView.svelte` interactive
  reference: base URL, auth, `/v1` endpoint table, curl examples), the
  **Dev Tools** group (**Logs** — Usage → Details; **Proxy Pools**;
  **Console Logs** (relabeled from Console Log); CLI Tools; Skills;
  **Playground** — new `PlaygroundView.svelte` live chat against
  `/v1/chat/completions` with a model picker fed by `/v1/models`), an
  **Advanced** group (Security, Persona), the existing **System** group (Media
  Providers accordion), and a standalone bottom **Appearance** entry —
  relabeled from Settings to match cheaprouter's bottom item. Note: 9router's
  settings page covers more than appearance (SSO, retention, whitelabeling…),
  unlike cheaprouter's — the label follows cheaprouter per the request; the
  sidebar structure, icons and order are otherwise unchanged from the request.
- **Routing.** Tab order is stable for existing bookmarks: `/` and
  `/dashboard` now land on Home, `/dashboard/keys` lands on the standalone
  API Keys page, `/dashboard/logs` opens the request log view, and new routes
  `/dashboard/{manage-models,api-keys,api-reference,playground}` were added.
  Sidebar badge moved from the Endpoint entry to API Keys
  (`navBadges.ts`); `Limits Bank` (Plans) is no longer default-hidden.
  `e2e/sidebarVisibility.test.ts` and `e2e/commandPalette.test.ts` were
  updated for the `Settings → Appearance` label change.
- **Deliberately not ported:** cheaprouter's **Users** page (reseller
  multi-tenant: plans/BYOK/per-user keys — 9router-go is a single-operator
  gateway with one master password, no RBAC), **Pipe Models activation**
  toggles (per-key model ACLs already live in Plans/Security), and
  **Proxy Logs** (no per-request proxy log exists in the Go backend yet).
  Cheaprouter's per-site scoping (`?site=`) is not ported either. These are
  documented here instead of being faked with placeholder pages.
- Ported behavior was implemented from the cheaprouter source
  (`/home/hasan/Documents/reactjs/cheaprouter`, read-only), not from upstream
  9router; the cheaprouter catalog entries are a fork of the same
  decolua/9router lineage, so the port is contract-level (auth hints, model
  ids, base URLs), not a transliteration.

### Verification

- `go build ./...` passes (Go 1.27.4 downloaded into the toolchain dir for
  this work — the host had no `go` on PATH) and `go vet` clean for the
  touched packages.
- `go test ./internal/providers/ ./internal/proxy/executor/` passes,
  including `webcookie_catalog_test.go` (all catalog contracts) and the
  dashboard catalog parity test — **156 providers checked** in both catalogs.
- `web/src/lib/models.ts` gained the static model catalogs for the new
  providers (generated from `ProviderModels`, names from the cheaprouter spec),
  and each new provider ships a generated SVG badge under
  `web/public/providers/` (the `ships artwork for every provider` test).
- `bun run build` passes; `bun run ratchet:svelte` — **0 unresolved
  identifiers**, svelte-check baseline unchanged (74 pre-existing errors);
  `bun test` — **843 pass / 36 fail**, every failure is the pre-existing
  "gateway binary not found" e2e environment issue (present on a clean
  checkout too, confirmed via `git stash`).
- e2e suite (real binary + Chromium): the first full runs exposed **six
  failures caused by this change** — a duplicated `<ApiKeysTable>` mount on
  the standalone API Keys page (strict-mode "2 elements"), a `Key Name`
  label/id missing from the inline create form (the spec that drove it
  expects `getByLabel('Key Name')`), a "Create Key" accessible-name clash,
  the `Advanced Features` group heading rename, the Sidebar badge moving
  from `Endpoint & Key` to `API Keys`, and the `Settings → Appearance`
  label change — **all six are fixed and verified**, the four affected
  spec files pass 16/16 in isolation (~62 s). The remaining full-suite
  failures are 20–60 s-budget tests that lose against CPU contention in
  this machine (the same specs time out on a pristine checkout — verified
  with `git stash`; they pass in isolation).
- Cheaprouter reference: `/home/hasan/Documents/reactjs/cheaprouter` at
  `bd63f95` (read-only); web-cookie catalog extraction in
  `src/shared/constants/providers/web-cookie*.ts` and `noauth.ts`.
- **Not verified against live provider endpoints** — the cookie/no-auth
  transports replay browser sessions whose endpoints rotate; only the catalog
  wiring is statically asserted (`webcookie_catalog_test.go`).
