<script lang="ts">
  import {
    api,
    normalizeLastError,
    type APIKey,
    type ProviderConnection,
    type ProviderNode,
    type SystemVersionInfo,
  } from '../api/client'
  import type { ActiveTab } from '../lib/router'

  interface Props {
    connections?: ProviderConnection[]
    providerNodes?: ProviderNode[]
    /** Combo rows as served by /api/combos; only the count is used here. */
    combos?: any[]
    apiKeys?: APIKey[]
    navigate?: (tab: ActiveTab) => void
  }

  let {
    connections = [],
    providerNodes = [],
    combos = [],
    apiKeys = [],
    navigate = () => {},
  }: Props = $props()

  const nf = new Intl.NumberFormat()

  // The two reads this page owns. Everything else arrives as props from the
  // app shell. Both are guarded: a failed /api/version or a 500 from
  // /api/usage/stats must never blank the landing page, so errors resolve to
  // null and the cards render their fallback instead.
  let versionInfo = $state<SystemVersionInfo | null>(null)
  let stats = $state<any>(null)
  let statsLoading = $state(true)

  $effect(() => {
    let cancelled = false
    api
      .getSystemVersion()
      .then((v) => {
        if (!cancelled && v) versionInfo = v
      })
      .catch(() => {})
    api
      .getUsageStats('today')
      .catch(() => null)
      .then((res) => {
        if (cancelled) return
        stats = res
        statsLoading = false
      })
    return () => {
      cancelled = true
    }
  })

  const totalProviders = $derived(connections.length)
  const activeProviders = $derived(connections.filter((c) => c.isActive === 1).length)
  const activeKeys = $derived(apiKeys.filter((k) => k.isActive === 1).length)

  const todayRequests = $derived(
    stats && typeof stats.totalRequests === 'number' ? stats.totalRequests : null,
  )
  const todayTokens = $derived.by(() => {
    if (!stats) return null
    const prompt = Number(stats.totalPromptTokens)
    const completion = Number(stats.totalCompletionTokens)
    if (!Number.isFinite(prompt) && !Number.isFinite(completion)) return null
    return (Number.isFinite(prompt) ? prompt : 0) + (Number.isFinite(completion) ? completion : 0)
  })

  // Active connections first so the six slots go to the rows that matter;
  // inactive rows fill in behind them.
  const healthRows = $derived(
    [...connections].sort((a, b) => (b.isActive ?? 0) - (a.isActive ?? 0)).slice(0, 6),
  )

  function connectionLabel(conn: ProviderConnection): string {
    return conn.displayName || conn.name || conn.email || conn.provider
  }

  type ErrorHint = 'Quota' | 'Error'
  function errorHint(conn: ProviderConnection): { kind: ErrorHint; text: string } | null {
    const text = normalizeLastError(conn.lastError)
    if (!text) return null
    const quota = /quota|429|rate.?limit|exhaust/i.test(text)
    return { kind: quota ? 'Quota' : 'Error', text }
  }

  const quickActions: { icon: string; label: string; desc: string; tab: ActiveTab }[] = [
    { icon: 'dns', label: 'Providers', desc: 'Connect & manage accounts', tab: 'connections' },
    { icon: 'layers', label: 'Combos', desc: 'Model fallback pipelines', tab: 'combos' },
    { icon: 'tune', label: 'Pipe Models', desc: 'Curate pool routing order', tab: 'manage-models' },
    { icon: 'vpn_key', label: 'API Keys', desc: 'Issue & rotate client keys', tab: 'api-keys' },
    { icon: 'science', label: 'Playground', desc: 'Test prompts live', tab: 'playground' },
    { icon: 'bar_chart', label: 'Usage', desc: 'Requests, tokens & cost', tab: 'analytics' },
  ]

  const cardClass =
    'bg-surface border border-border rounded-[10px] shadow-[var(--shadow-warm)] flex min-w-0 flex-col gap-1.5 px-4 py-3.5'
  const labelClass = 'text-text-subtle text-[10px] font-semibold uppercase tracking-wider'
</script>

<div class="mx-auto w-full max-w-7xl space-y-6 p-4 sm:p-6">
  <!-- Header row -->
  <div class="flex flex-wrap items-center justify-between gap-4">
    <div class="flex min-w-0 items-center gap-3">
      <div
        class="flex h-11 w-11 shrink-0 items-center justify-center rounded-[10px] bg-brand-500/10 text-brand-500 shadow-[var(--shadow-warm)]"
      >
        <span class="material-symbols-outlined text-[26px]" aria-hidden="true">home</span>
      </div>
      <div class="min-w-0">
        <h1 class="font-headline text-2xl font-bold text-text-main sm:text-3xl">Analytics</h1>
        <p class="truncate text-xs text-text-muted sm:text-sm">
          Gateway overview — providers, combos, API keys and traffic
        </p>
      </div>
    </div>
    {#if versionInfo}
      <span
        class="inline-flex items-center gap-1.5 rounded-full border border-brand-500/25 bg-brand-500/10 px-2.5 py-1 text-[11px] font-medium text-brand-500"
        title={versionInfo.hasUpdate && versionInfo.latestVersion
          ? `Update available: v${versionInfo.latestVersion}`
          : 'Gateway up to date'}
      >
        <span class="material-symbols-outlined text-[14px]" aria-hidden="true">
          {versionInfo.hasUpdate ? 'system_update_alt' : 'check_circle'}
        </span>
        v{versionInfo.hasUpdate && versionInfo.latestVersion
          ? versionInfo.latestVersion
          : versionInfo.currentVersion}
      </span>
    {/if}
  </div>

  <!-- Stat cards -->
  <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
    <div class={cardClass}>
      <div class="flex items-center justify-between gap-2">
        <span class={labelClass}>Active providers</span>
        <span class="material-symbols-outlined text-[18px] text-text-subtle" aria-hidden="true">dns</span>
      </div>
      <span class="font-code text-2xl font-semibold text-text-main">{activeProviders}</span>
      <span class="truncate text-[11px] text-text-subtle">
        of {nf.format(totalProviders)} connected{providerNodes.length > 0
          ? ` · ${nf.format(providerNodes.length)} custom node${providerNodes.length === 1 ? '' : 's'}`
          : ''}
      </span>
    </div>

    <div class={cardClass}>
      <div class="flex items-center justify-between gap-2">
        <span class={labelClass}>Combos</span>
        <span class="material-symbols-outlined text-[18px] text-text-subtle" aria-hidden="true">layers</span>
      </div>
      <span class="font-code text-2xl font-semibold text-text-main">{nf.format(combos.length)}</span>
      <span class="truncate text-[11px] text-text-subtle">model fallback pipelines</span>
    </div>

    <div class={cardClass}>
      <div class="flex items-center justify-between gap-2">
        <span class={labelClass}>API keys</span>
        <span class="material-symbols-outlined text-[18px] text-text-subtle" aria-hidden="true">vpn_key</span>
      </div>
      <span class="font-code text-2xl font-semibold text-text-main">{nf.format(activeKeys)}</span>
      <span class="truncate text-[11px] text-text-subtle">
        active of {nf.format(apiKeys.length)} total
      </span>
    </div>

    <div
      class={cardClass}
      title="Requests recorded today across all providers"
    >
      <div class="flex items-center justify-between gap-2">
        <span class={labelClass}>Requests today</span>
        <span class="material-symbols-outlined text-[18px] text-text-subtle" aria-hidden="true">bolt</span>
      </div>
      <span class="font-code text-2xl font-semibold text-text-main">
        {todayRequests !== null ? nf.format(todayRequests) : statsLoading ? '…' : '—'}
      </span>
      <span class="truncate text-[11px] text-text-subtle">
        {todayTokens !== null ? `${nf.format(todayTokens)} tokens` : 'tokens unavailable'}
      </span>
    </div>
  </div>

  <!-- Quick actions -->
  <section class="space-y-2">
    <h2 class="text-xs font-semibold uppercase tracking-wider text-text-subtle">Quick actions</h2>
    <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-6">
      {#each quickActions as action (action.tab)}
        <button
          type="button"
          onclick={() => navigate(action.tab)}
          class="group flex min-w-0 cursor-pointer flex-col items-start gap-2 rounded-[10px] border border-border bg-surface px-3 py-3 text-left shadow-[var(--shadow-warm)] transition-colors hover:border-brand-500/40 hover:bg-surface-2"
        >
          <span
            class="flex h-9 w-9 items-center justify-center rounded-[10px] bg-surface-2 text-brand-500 transition-colors group-hover:bg-brand-500/10"
          >
            <span class="material-symbols-outlined text-[20px]" aria-hidden="true">{action.icon}</span>
          </span>
          <span class="text-xs font-semibold text-text-main">{action.label}</span>
          <span class="text-[10px] leading-tight text-text-subtle">{action.desc}</span>
        </button>
      {/each}
    </div>
  </section>

  <!-- Provider health -->
  <section class="overflow-hidden rounded-[10px] border border-border bg-surface shadow-[var(--shadow-warm)]">
    <div class="flex items-center justify-between gap-3 border-b border-border bg-surface-2 px-4 py-2.5">
      <div class="flex min-w-0 items-center gap-2">
        <span class="material-symbols-outlined text-[18px] text-text-muted" aria-hidden="true"
          >health_and_safety</span
        >
        <h2 class="font-headline text-sm font-bold text-text-main">Provider health</h2>
        <span class="font-code text-[10px] text-text-subtle">{activeProviders}/{totalProviders} active</span>
      </div>
      <button
        type="button"
        onclick={() => navigate('connections')}
        class="inline-flex shrink-0 cursor-pointer items-center gap-0.5 text-[11px] font-semibold text-brand-500 transition-colors hover:text-primary-hover"
      >
        Manage
        <span class="material-symbols-outlined text-[14px]" aria-hidden="true">arrow_forward</span>
      </button>
    </div>

    {#if healthRows.length === 0}
      <div class="flex flex-col items-center gap-2 px-4 py-8 text-center">
        <span class="material-symbols-outlined text-[32px] text-text-subtle" aria-hidden="true">dns</span>
        <p class="text-sm font-medium text-text-muted">No providers connected yet</p>
        <p class="text-xs text-text-subtle">
          Add one from the
          <button
            type="button"
            onclick={() => navigate('connections')}
            class="cursor-pointer font-semibold text-brand-500 hover:text-primary-hover"
          >
            Providers page
          </button>.
        </p>
      </div>
    {:else}
      <ul class="divide-y divide-border-subtle">
        {#each healthRows as conn (conn.id)}
          {@const hint = errorHint(conn)}
          <li class="flex items-center gap-3 px-4 py-2.5">
            <span
              class="h-2 w-2 shrink-0 rounded-full {conn.isActive === 1 ? 'bg-success' : 'bg-text-subtle'}"
              title={conn.isActive === 1 ? 'Active' : 'Inactive'}
              aria-hidden="true"
            ></span>
            <span
              class="w-20 shrink-0 truncate rounded bg-surface-2 px-1.5 py-0.5 font-code text-[10px] text-text-muted"
              title={conn.provider}
            >
              {conn.provider}
            </span>
            <span class="min-w-0 flex-1 truncate text-xs font-medium text-text-main" title={connectionLabel(conn)}>
              {connectionLabel(conn)}
            </span>
            {#if hint}
              <span
                class="inline-flex shrink-0 items-center gap-1 rounded px-1.5 py-0.5 text-[10px] font-medium {hint.kind ===
                'Quota'
                  ? 'bg-warning/10 text-warning'
                  : 'bg-danger/10 text-danger'}"
                title={hint.text}
              >
                <span class="material-symbols-outlined text-[12px]" aria-hidden="true">
                  {hint.kind === 'Quota' ? 'hourglass_empty' : 'error'}
                </span>
                {hint.kind}
              </span>
            {:else if conn.isActive === 1}
              <span class="shrink-0 text-[10px] font-medium text-text-subtle">Healthy</span>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</div>
