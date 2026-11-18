<script lang="ts">
  // Interactive developer documentation for the gateway — the cheaprouter
  // "API Reference" page. Everything on this page is derivable by hitting the
  // gateway; no server round-trip is needed to read it.
  const base = typeof window !== 'undefined' ? window.location.origin : 'http://localhost:20130'

  interface Endpoint {
    method: 'GET' | 'POST'
    path: string
    label: string
    description: string
    sample: string
  }

  const endpoints: Endpoint[] = [
    {
      method: 'POST',
      path: '/v1/chat/completions',
      label: 'Chat Completions (OpenAI)',
      description: 'The workhorse endpoint. Streams SSE chunks with `stream: true`; replays any provider/model via the `provider/model` wire syntax or the combo resolver.',
      sample: `curl ${base}/v1/chat/completions \\
  -H "Authorization: Bearer sk-..." \\
  -H "Content-Type: application/json" \\
  -d '{"model":"anthropic/claude-opus-4","messages":[{"role":"user","content":"hi"}]}'`,
    },
    {
      method: 'POST',
      path: '/v1/messages',
      label: 'Messages (Anthropic)',
      description: 'Anthropic Messages API — the same routing pipeline, Anthropic wire format in and out.',
      sample: `curl ${base}/v1/messages \\
  -H "Authorization: Bearer sk-..." \\
  -H "Content-Type: application/json" \\
  -H "anthropic-version: 2023-06-01" \\
  -d '{"model":"anthropic/claude-opus-4","max_tokens":1024,"messages":[{"role":"user","content":"hi"}]}'`,
    },
    {
      method: 'GET',
      path: '/v1/models',
      label: 'List models',
      description: 'OpenAI-compatible model catalog across every connected provider. Use `provider/model` ids to force a lane.',
      sample: `curl ${base}/v1/models -H "Authorization: Bearer sk-..."`,
    },
    {
      method: 'POST',
      path: '/v1/embeddings',
      label: 'Embeddings',
      description: 'Vector embeddings via any connected embedding model (Google, OpenAI, Jina, Voyage…).',
      sample: `curl ${base}/v1/embeddings \\
  -H "Authorization: Bearer sk-..." \\
  -d '{"model":"google/text-embedding-004","input":"hello world"}'`,
    },
    {
      method: 'POST',
      path: '/v1/images/generations',
      label: 'Image generation',
      description: 'Text-to-image via DALL-E, Imagen, Flux, SDWebUI — or a web-cookie lane such as Adobe Firefly.',
      sample: `curl ${base}/v1/images/generations \\
  -H "Authorization: Bearer sk-..." \\
  -d '{"model":"antigravity/gemini-3.1-flash-image","prompt":"a red panda astronaut"}'`,
    },
    {
      method: 'POST',
      path: '/v1/audio/speech',
      label: 'Text to speech',
      description: 'OpenAI-compatible TTS. Works with any connected TTS provider — Edge, Google, ElevenLabs, Kokoro…',
      sample: `curl ${base}/v1/audio/speech \\
  -H "Authorization: Bearer sk-..." \\
  -d '{"model":"edge-tts/tts-1","input":"hello","voice":"en-US-GuyNeural"}' --output hello.mp3`,
    },
  ]

  const notes = [
    {
      icon: 'key',
      title: 'Auth',
      text: 'Client API keys are minted on the API Keys page (or the Endpoints tab). Send them as `Authorization: Bearer <key>`. The dashboard session cookie never works against /v1.',
    },
    {
      icon: 'route',
      title: 'Model routing',
      text: '`model` accepts `provider/model` (direct lane), a combo name (fallback/round-robin/fusion per the combo strategy), or a bare model id (auto-routed). Prefixes: ag/antigravity, oc/opencode, ds-web (DeepSeek Web), pw (Perplexity Web) …',
    },
    {
      icon: 'cookie',
      title: 'Web & cookie providers',
      text: 'Providers in the Web Cookie Providers section of the Providers tab authenticate with a browser session (cookie/userToken) instead of an API key. Paste the session on that provider’s page; the gateway replays it sealed at rest. Pending executors answer 503 by design — see the notice on the card.',
    },
    {
      icon: 'bolt',
      title: 'Streaming',
      text: 'All chat endpoints stream SSE with `stream: true`. Non-streaming clients get the fully-collected response in one JSON body.',
    },
  ]

  let copied = $state<string | null>(null)
  async function copy(text: string, id: string) {
    try {
      await navigator.clipboard.writeText(text)
      copied = id
      setTimeout(() => (copied = null), 1500)
    } catch {
      // clipboard unavailable — leave the snippet selectable
    }
  }
</script>

<div class="space-y-6">
  <div>
    <div class="flex items-center gap-3">
      <span class="material-symbols-outlined text-[28px] text-brand-500">menu_book</span>
      <h1 class="text-2xl font-bold text-text-main">API Docs</h1>
    </div>
    <p class="mt-1 max-w-2xl text-sm text-text-muted">
      The gateway exposes an OpenAI-compatible surface under <code class="rounded bg-surface-2 px-1.5 py-0.5 text-[12px]">{base}/v1</code>.
      Everything below is copy-paste ready; the machine-readable catalog is <code class="rounded bg-surface-2 px-1.5 py-0.5 text-[12px]">GET {base}/v1/models</code>.
    </p>
  </div>

  <!-- Base facts -->
  <div class="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
    {#each notes as note (note.title)}
      <div class="rounded-[10px] border border-border bg-surface p-4">
        <span class="material-symbols-outlined text-[20px] text-brand-500">{note.icon}</span>
        <h2 class="mt-2 text-sm font-semibold text-text-main">{note.title}</h2>
        <p class="mt-1 text-[12px] leading-snug text-text-muted">{note.text}</p>
      </div>
    {/each}
  </div>

  <!-- Endpoints -->
  <div class="space-y-3">
    {#each endpoints as ep (ep.path)}
      <div class="overflow-hidden rounded-[10px] border border-border bg-surface">
        <div class="flex flex-wrap items-center gap-3 border-b border-border-subtle bg-surface-2/40 px-4 py-2.5">
          <span
            class="rounded px-2 py-0.5 font-code text-[11px] font-bold {ep.method === 'GET'
              ? 'bg-green-500/15 text-green-400'
              : 'bg-sky-500/15 text-sky-400'}">{ep.method}</span
          >
          <code class="font-code text-sm text-text-main">{ep.path}</code>
          <span class="text-xs font-semibold text-text-subtle">· {ep.label}</span>
          <button
            type="button"
            class="ml-auto inline-flex size-7 items-center justify-center rounded-md border border-border text-text-subtle transition-colors hover:text-text-main"
            title="Copy curl example"
            onclick={() => copy(ep.sample, ep.path)}
          >
            <span class="material-symbols-outlined text-[16px]">{copied === ep.path ? 'check' : 'content_copy'}</span>
          </button>
        </div>
        <div class="px-4 py-3">
          <p class="text-xs text-text-muted">{ep.description}</p>
          <pre class="mt-2 overflow-x-auto rounded-[10px] border border-border-subtle bg-black/30 p-3 font-code text-[11px] leading-relaxed text-text-muted">{ep.sample}</pre>
        </div>
      </div>
    {/each}
  </div>

  <p class="text-[11px] text-text-subtle">
    Full surface: embeddings, images (generations/edits), audio (speech/transcriptions), moderations, rerank, batches, files, and the Anthropic Messages lane.
    For per-provider configuration head to <code class="rounded bg-surface-2 px-1 py-0.5">Providers</code>; for key management to <code class="rounded bg-surface-2 px-1 py-0.5">API Keys</code>.
  </p>
</div>
