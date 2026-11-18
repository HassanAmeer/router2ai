<script lang="ts">
  import { PROVIDER_CATALOG, PROVIDER_CATEGORIES, type ProviderCatalogItem } from '../lib/providers'
  import { getModelsByProviderId } from '../lib/models'
  import { getIconPath } from './connections/types'

  // Pipe Models — the cross-provider catalog: every provider the registry
  // knows, the models each of them publishes, and a link to where the model
  // is actually managed (per-provider detail pages own activation/aliasing).
  let search = $state('')
  let activeSection = $state<'all' | string>('all')

  const sections = PROVIDER_CATEGORIES.map((c) => ({ ...c }))

  const catalog = PROVIDER_CATALOG.filter((item) => !item.hidden)

  const filtered = $derived(
    catalog.filter((item) => {
      if (activeSection !== 'all' && item.category !== activeSection) return false
      if (!search.trim()) return true
      const q = search.trim().toLowerCase()
      if (item.name.toLowerCase().includes(q) || item.id.toLowerCase().includes(q)) return true
      const models = getModelsByProviderId(item.id)
      return models.some((model) => model.id.toLowerCase().includes(q))
    }),
  )

  const bySection = $derived.by(() => {
    const map = new Map<string, ProviderCatalogItem[]>()
    for (const item of filtered) {
      const list = map.get(item.category) ?? []
      list.push(item)
      map.set(item.category, list)
    }
    return map
  })

  const modelCount = $derived((id: string) => getModelsByProviderId(id).length)

  const totals = $derived({
    providers: catalog.length,
    models: catalog.reduce((sum, item) => sum + modelCount(item.id), 0),
    connected: catalog.filter((item) => item.authType !== 'none').length,
  })

  const sectionLabel = (id: string) =>
    sections.find((section) => section.id === id)?.label ?? id
</script>

<div class="space-y-6">
  <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
    <div>
      <div class="flex items-center gap-3">
        <span class="material-symbols-outlined text-[28px] text-brand-500">tune</span>
        <h1 class="text-2xl font-bold text-text-main">Pipe Models</h1>
      </div>
      <p class="mt-1 max-w-2xl text-sm text-text-muted">
        Every provider and model the gateway can route, grouped the way the
        Providers tab groups them. Activation, aliases and tests live on each
        provider's detail page.
      </p>
    </div>
    <div class="flex gap-4 text-right font-code text-sm text-text-subtle">
      <div>
        <div class="text-xl font-bold text-text-main">{totals.providers}</div>
        <div class="text-[10px] uppercase tracking-wide">providers</div>
      </div>
      <div>
        <div class="text-xl font-bold text-text-main">{totals.models}</div>
        <div class="text-[10px] uppercase tracking-wide">models</div>
      </div>
    </div>
  </div>

  <!-- Filters -->
  <div class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
    <div class="flex flex-wrap items-center gap-2">
      <button
        type="button"
        class="rounded-[10px] border px-3 py-1.5 text-xs font-semibold transition-colors {activeSection ===
        'all'
          ? 'border-brand-500 bg-brand-500/10 text-brand-500'
          : 'border-border bg-surface-2 text-text-muted hover:text-text-main'}"
        onclick={() => (activeSection = 'all')}
      >
        All ({catalog.length})
      </button>
      {#each sections as section (section.id)}
        {@const count = catalog.filter((item) => item.category === section.id).length}
        {#if count > 0}
          <button
            type="button"
            class="rounded-[10px] border px-3 py-1.5 text-xs font-semibold transition-colors {activeSection ===
            section.id
              ? 'border-brand-500 bg-brand-500/10 text-brand-500'
              : 'border-border bg-surface-2 text-text-muted hover:text-text-main'}"
            onclick={() => (activeSection = section.id)}
          >
            {section.label} ({count})
          </button>
        {/if}
      {/each}
    </div>
    <input
      type="search"
      bind:value={search}
      placeholder="Filter providers or models…"
      class="w-full rounded-[10px] border border-border bg-surface-2 px-3 py-2 text-sm text-text-main placeholder:text-text-subtle focus:outline-none focus:ring-1 focus:ring-brand-500 lg:w-80"
      aria-label="Filter providers or models"
    />
  </div>

  <!-- Catalog -->
  {#if filtered.length === 0}
    <div class="rounded-[10px] border border-dashed border-border bg-surface p-8 text-center text-sm text-text-muted">
      No provider or model matches “{search}”.
    </div>
  {:else}
    {#each [...bySection.entries()] as [category, items] (category)}
      <section class="space-y-2">
        <h2 class="text-[10px] font-semibold uppercase tracking-wider text-text-subtle">
          {sectionLabel(category)} · {items.length}
        </h2>
        <div class="overflow-hidden rounded-[10px] border border-border bg-surface">
          {#each items as item (item.id)}
            {@const models = getModelsByProviderId(item.id)}
            <details class="group border-b border-border-subtle last:border-b-0">
              <summary class="flex cursor-pointer list-none items-center gap-3 px-4 py-2.5 hover:bg-surface-2/60">
                <img
                  class="size-6 rounded object-contain"
                  src={getIconPath(item.id)}
                  alt=""
                  loading="lazy"
                  onerror={(e) => ((e.currentTarget as HTMLImageElement).style.opacity = '0')}
                />
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2">
                    <span class="truncate text-sm font-medium text-text-main">{item.name}</span>
                    {#if item.alias}
                      <code class="rounded bg-surface-2 px-1.5 py-0.5 text-[10px] text-text-muted">{item.alias}</code>
                    {/if}
                    {#if item.noAuth}
                      <span class="rounded-full border border-brand-500/40 bg-brand-500/10 px-2 py-0.5 text-[9px] font-bold uppercase tracking-wide text-brand-500">No auth</span>
                    {/if}
                  </div>
                </div>
                <span class="shrink-0 font-code text-[11px] text-text-subtle">{models.length} models</span>
                <span class="material-symbols-outlined text-[18px] text-text-subtle transition-transform group-open:rotate-180">expand_more</span>
              </summary>
              <div class="border-t border-border-subtle bg-surface-2/30 px-4 py-3">
                {#if models.length === 0}
                  <p class="text-xs text-text-subtle">Models are fetched live for this provider — manage them on the provider detail page.</p>
                {:else}
                  <div class="flex flex-wrap gap-1.5">
                    {#each models as model (model.id)}
                      <code class="rounded-md border border-border bg-surface px-2 py-0.5 text-[11px] text-text-muted">{model.id}</code>
                    {/each}
                  </div>
                {/if}
                {#if item.website}
                  <a class="mt-2 inline-flex items-center gap-1 text-[11px] text-brand-500 hover:underline" href={item.website} target="_blank" rel="noreferrer noopener">
                    {item.website}
                    <span class="material-symbols-outlined text-[12px]">open_in_new</span>
                  </a>
                {/if}
              </div>
            </details>
          {/each}
        </div>
      </section>
    {/each}
  {/if}
</div>
