<script lang="ts">
  // Playground — a live testbench against the gateway's own /v1 lane.
  // Auth: a client API key (the same keys the API Keys page mints). The key is
  // persisted in this browser's localStorage only; it never leaves the machine.
  interface ChatMessage {
    role: 'user' | 'assistant'
    content: string
    pending?: boolean
    error?: boolean
  }

  const storedKey = typeof localStorage !== 'undefined' ? (localStorage.getItem('9router_key') ?? '') : ''

  let apiKey = $state(storedKey)
  let models = $state<string[]>([])
  let model = $state('')
  let modelsLoading = $state(false)
  let modelsError = $state<string | null>(null)

  let input = $state('')
  let messages = $state<ChatMessage[]>([])
  let sending = $state(false)
  let lastLatencyMs = $state<number | null>(null)
  let lastUsage = $state<{ prompt_tokens?: number; completion_tokens?: number; total_tokens?: number } | null>(null)

  const canSend = $derived(!!model && input.trim().length > 0 && !sending)

  async function fetchModels() {
    const key = apiKey.trim()
    if (!key) {
      modelsError = 'Enter a client API key first'
      return
    }
    modelsLoading = true
    modelsError = null
    try {
      const res = await fetch('/v1/models', { headers: { Authorization: `Bearer ${key}` } })
      const body = await res.json().catch(() => null)
      if (!res.ok) throw new Error(body?.error?.message || `HTTP ${res.status}`)
      const ids: string[] = (body?.data ?? []).map((m: { id: string }) => m.id).filter(Boolean)
      models = ids
      if (ids.length > 0 && !ids.includes(model)) {
        model = ids.includes('anthropic/claude-opus-4') ? 'anthropic/claude-opus-4' : ids[0]
      }
    } catch (err) {
      modelsError = err instanceof Error ? err.message : String(err)
      models = []
    } finally {
      modelsLoading = false
    }
  }

  function saveKey() {
    try {
      localStorage.setItem('9router_key', apiKey.trim())
    } catch {
      // storage unavailable — the key still works for this session
    }
  }

  async function send() {
    const text = input.trim()
    if (!canSend || !text) return
    saveKey()
    const key = apiKey.trim()
    const history = messages.filter((m) => !m.pending && !m.error).map((m) => ({ role: m.role, content: m.content }))
    messages = [...messages, { role: 'user', content: text }, { role: 'assistant', content: '', pending: true }]
    input = ''
    sending = true
    const started = performance.now()
    try {
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: { Authorization: `Bearer ${key}`, 'Content-Type': 'application/json' },
        body: JSON.stringify({ model, messages: [...history, { role: 'user', content: text }] }),
      })
      const body = await res.json().catch(() => null)
      if (!res.ok) throw new Error(body?.error?.message || `HTTP ${res.status}`)
      const reply: string = body?.choices?.[0]?.message?.content ?? '(empty response)'
      lastUsage = body?.usage ?? null
      lastLatencyMs = Math.round(performance.now() - started)
      messages = messages.map((m) => (m.pending ? { role: 'assistant' as const, content: reply } : m))
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      messages = messages.map((m) => (m.pending ? { role: 'assistant' as const, content: `Error: ${msg}`, error: true } : m))
    } finally {
      sending = false
    }
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault()
      void send()
    }
  }

  function clearChat() {
    messages = []
    lastUsage = null
    lastLatencyMs = null
  }
</script>

<div class="flex h-[calc(100vh-7rem)] flex-col gap-4 lg:flex-row">
  <!-- Controls -->
  <div class="w-full shrink-0 space-y-4 lg:w-80">
    <div>
      <div class="flex items-center gap-3">
        <span class="material-symbols-outlined text-[28px] text-brand-500">science</span>
        <h1 class="text-2xl font-bold text-text-main">Playground</h1>
      </div>
      <p class="mt-1 text-sm text-text-muted">
        Live testbench against <code class="rounded bg-surface-2 px-1 py-0.5 text-[12px]">/v1/chat/completions</code>.
        Bring a client API key — it stays in this browser.
      </p>
    </div>

    <label class="block space-y-1.5">
      <span class="text-[10px] font-semibold uppercase tracking-wide text-text-subtle">Client API key</span>
      <input
        type="password"
        bind:value={apiKey}
        onchange={saveKey}
        placeholder="sk-..."
        autocomplete="off"
        class="w-full rounded-[10px] border border-border bg-surface-2 px-3 py-2 font-code text-sm text-text-main placeholder:text-text-subtle focus:outline-none focus:ring-1 focus:ring-brand-500"
      />
    </label>

    <div class="flex items-center justify-between gap-2">
      <span class="text-[10px] font-semibold uppercase tracking-wide text-text-subtle">Model</span>
      <button
        type="button"
        onclick={fetchModels}
        disabled={modelsLoading}
        class="inline-flex items-center gap-1 rounded-[8px] border border-border bg-surface-2 px-2 py-1 text-[11px] font-semibold text-text-muted transition-colors hover:text-text-main disabled:opacity-50"
      >
        <span class="material-symbols-outlined text-[14px] {modelsLoading ? 'animate-spin' : ''}">{modelsLoading ? 'progress_activity' : 'sync'}</span>
        {modelsLoading ? 'Loading…' : models.length > 0 ? `${models.length} models` : 'Fetch models'}
      </button>
    </div>
    {#if modelsError}
      <p class="text-[11px] text-red-400">{modelsError}</p>
    {:else if models.length > 0}
      <select
        bind:value={model}
        class="w-full rounded-[10px] border border-border bg-surface-2 px-3 py-2 font-code text-xs text-text-main focus:outline-none focus:ring-1 focus:ring-brand-500"
        aria-label="Model"
      >
        {#each models as id (id)}
          <option value={id}>{id}</option>
        {/each}
      </select>
    {:else}
      <input
        type="text"
        bind:value={model}
        placeholder="provider/model — e.g. anthropic/claude-opus-4"
        class="w-full rounded-[10px] border border-border bg-surface-2 px-3 py-2 font-code text-xs text-text-main placeholder:text-text-subtle focus:outline-none focus:ring-1 focus:ring-brand-500"
        aria-label="Model id"
      />
    {/if}

    <div class="rounded-[10px] border border-border-subtle bg-surface p-3 text-[11px] text-text-subtle">
      <div class="flex items-center gap-1.5 font-semibold uppercase tracking-wide text-text-muted">
        <span class="material-symbols-outlined text-[14px] text-brand-500">bolt</span>
        Notes
      </div>
      <ul class="mt-1.5 list-disc space-y-1 pl-4">
        <li>Combo names route through the combo strategy (fallback / round-robin / fusion).</li>
        <li>Non-streaming — the full response is shown at once.</li>
      </ul>
      {#if lastLatencyMs !== null}
        <div class="mt-2 border-t border-border-subtle pt-2 font-code text-[11px] text-text-muted">
          {lastLatencyMs} ms{lastUsage?.total_tokens ? ` · ${lastUsage.total_tokens} tokens` : ''}
        </div>
      {/if}
    </div>

    {#if messages.length > 0}
      <button type="button" onclick={clearChat} class="text-[11px] font-semibold text-text-muted underline underline-offset-2 hover:text-text-main">
        Clear conversation
      </button>
    {/if}
  </div>

  <!-- Transcript -->
  <div class="flex min-h-0 flex-1 flex-col rounded-[10px] border border-border bg-surface">
    <div class="flex-1 space-y-3 overflow-y-auto p-4">
      {#if messages.length === 0}
        <div class="flex h-full flex-col items-center justify-center gap-2 text-center">
          <span class="material-symbols-outlined text-[36px] text-text-subtle">forum</span>
          <p class="text-sm text-text-subtle">Ask the gateway anything. Responses come from whatever lane you pick.</p>
        </div>
      {:else}
        {#each messages as message, i (i)}
          <div class="flex {message.role === 'user' ? 'justify-end' : 'justify-start'}">
            <div
              class="max-w-[85%] whitespace-pre-wrap rounded-[14px] px-4 py-2 text-sm leading-relaxed {message.role === 'user'
                ? 'bg-brand-500/15 text-text-main'
                : message.error
                  ? 'border border-red-500/30 bg-red-500/10 text-red-300'
                  : 'bg-surface-2 text-text-main'}"
            >{#if message.pending}<span class="animate-pulse text-text-muted">Thinking…</span>{:else}{message.content}{/if}</div>
          </div>
        {/each}
      {/if}
    </div>
    <div class="border-t border-border-subtle p-3">
      <div class="flex items-end gap-2">
        <textarea
          bind:value={input}
          onkeydown={onKeydown}
          rows="2"
          placeholder="Type a prompt… (Enter to send, Shift+Enter for a new line)"
          class="max-h-48 min-h-[2.75rem] flex-1 resize-y rounded-[10px] border border-border bg-surface-2 px-3 py-2 text-sm text-text-main placeholder:text-text-subtle focus:outline-none focus:ring-1 focus:ring-brand-500"
        ></textarea>
        <button
          type="button"
          onclick={() => void send()}
          disabled={!canSend}
          class="inline-flex h-11 items-center gap-1.5 rounded-[10px] bg-primary px-4 text-sm font-semibold text-white transition-colors hover:bg-primary-hover disabled:opacity-40"
        >
          {sending ? 'Sending…' : 'Send'}
          <span class="material-symbols-outlined text-[18px]">send</span>
        </button>
      </div>
    </div>
  </div>
</div>
