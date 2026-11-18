<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Search,
    Check,
    Copy,
    Play,
    Save,
    RotateCw,
    Sliders,
    Layers,
    GitFork,
    Cpu,
    ArrowDown,
    ArrowUp,
    ChevronDown,
    ChevronLeft,
    ChevronRight,
    ChevronsLeft,
    ChevronsRight,
    X,
    AlertCircle,
  } from 'lucide-svelte'
  import { api, type CurationModelItem } from '../api/client'

  let models = $state<CurationModelItem[]>([])
  let providers = $state<string[]>([])
  let loading = $state(true)
  let error = $state('')
  let searchQuery = $state('')
  let selectedProvider = $state<string>('all')
  let statusFilter = $state<'all' | 'active' | 'inactive'>('all')

  // Pagination matching CheapRouter
  let page = $state(1)
  const pageSize = 50

  // Live test state per model key (`${providerId}::${originalModelId}`)
  let testingKey = $state<string | null>(null)
  let testResults = $state<Record<string, { ok: boolean; response?: string; latencyMs?: number; error?: string }>>({})
  let copiedId = $state<string | null>(null)
  let savingKey = $state<string | null>(null)
  let saveSuccessKey = $state<string | null>(null)

  // Horizontal scrollable tabs container
  let tabsContainer = $state<HTMLElement | null>(null)

  function scrollTabs(direction: 'left' | 'right') {
    if (!tabsContainer) return
    const offset = direction === 'left' ? -250 : 250
    tabsContainer.scrollBy({ left: offset, behavior: 'smooth' })
  }

  async function loadData() {
    loading = true
    error = ''
    try {
      const res = await api.getModelCuration()
      const rawModels = res.models || []
      models = rawModels.map((m) => ({
        ...m,
        origInputPrice: m.origInputPrice ?? (m.isFree ? 0 : 1.6),
        origCachePrice: m.origCachePrice ?? (m.isFree ? 0 : 0.25),
        origOutputPrice: m.origOutputPrice ?? (m.isFree ? 0 : 3.0),
        inputPrice: m.inputPrice ?? (m.isFree ? 0 : 0),
        cachePrice: m.cachePrice ?? (m.isFree ? 0 : 0),
        outputPrice: m.outputPrice ?? (m.isFree ? 0 : 0),
      }))
      providers = res.providers || []
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load model curation'
    } finally {
      loading = false
    }
  }

  onMount(() => {
    loadData()
  })

  let filteredModels = $derived.by(() => {
    let list = models
    if (selectedProvider !== 'all') {
      list = list.filter((m) => m.providerId === selectedProvider)
    }
    if (statusFilter === 'active') {
      list = list.filter((m) => m.isActive)
    } else if (statusFilter === 'inactive') {
      list = list.filter((m) => !m.isActive)
    }
    const q = searchQuery.trim().toLowerCase()
    if (q) {
      list = list.filter(
        (m) =>
          m.originalModelId.toLowerCase().includes(q) ||
          (m.customName && m.customName.toLowerCase().includes(q)) ||
          (m.customAlias && m.customAlias.toLowerCase().includes(q)) ||
          m.providerName.toLowerCase().includes(q)
      )
    }
    return list
  })

  let totalPages = $derived(Math.max(1, Math.ceil(filteredModels.length / pageSize)))
  let paginatedModels = $derived.by(() => {
    const start = (page - 1) * pageSize
    return filteredModels.slice(start, start + pageSize)
  })

  $effect(() => {
    // Reset to page 1 whenever selectedProvider or searchQuery changes
    selectedProvider
    searchQuery
    page = 1
  })

  let activeCount = $derived(models.filter((m) => m.isActive).length)

  async function toggleActive(item: CurationModelItem) {
    const nextState = !item.isActive
    item.isActive = nextState
    try {
      await api.postModelCuration({
        action: 'setActive',
        providerId: item.providerId,
        originalModelId: item.originalModelId,
        isActive: nextState,
      })
    } catch (e) {
      item.isActive = !nextState
    }
  }

  async function setAll(active: boolean) {
    try {
      await api.postModelCuration({
        action: 'setAllActive',
        isActive: active,
      })
      for (const m of models) {
        m.isActive = active
      }
    } catch (e) {
      // rollback
    }
  }

  async function saveModel(item: CurationModelItem) {
    const key = `${item.providerId}::${item.originalModelId}`
    savingKey = key
    saveSuccessKey = null
    try {
      await api.postModelCuration({
        action: 'save',
        providerId: item.providerId,
        originalModelId: item.originalModelId,
        customName: item.customName || '',
        customAlias: item.customAlias || '',
        contextWindow: Number(item.contextWindow) || 128000,
        inputPrice: Number(item.inputPrice) || 0,
        cachePrice: Number(item.cachePrice) || 0,
        outputPrice: Number(item.outputPrice) || 0,
        isActive: item.isActive,
      })
      saveSuccessKey = key
      setTimeout(() => {
        if (saveSuccessKey === key) saveSuccessKey = null
      }, 2500)
    } catch (err) {
      alert('Save failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      savingKey = null
    }
  }

  async function runLiveTest(item: CurationModelItem) {
    const key = `${item.providerId}::${item.originalModelId}`
    testingKey = key
    try {
      const res = await api.postModelCuration({
        action: 'liveTest',
        providerId: item.providerId,
        originalModelId: item.originalModelId,
        prompt: 'Hi, respond in 1 short sentence confirming you are working.',
      })
      testResults[key] = {
        ok: !!res.ok,
        response: res.response,
        latencyMs: res.latencyMs,
      }
    } catch (err) {
      testResults[key] = {
        ok: false,
        error: err instanceof Error ? err.message : 'Test failed',
      }
    } finally {
      testingKey = null
    }
  }

  function copyText(text: string, id: string) {
    navigator.clipboard.writeText(text)
    copiedId = id
    setTimeout(() => {
      if (copiedId === id) copiedId = null
    }, 2000)
  }
</script>

<div class="w-full space-y-4">
  <!-- CheapRouter Style Top Header -->
  <div class="w-full rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-3">
      <!-- Title + Active Badge -->
      <div class="flex items-center gap-2.5">
        <ChevronDown class="w-4 h-4 text-zinc-400" />
        <div class="p-1 rounded-lg bg-purple-500/10 text-purple-400 border border-purple-500/20">
          <Layers class="w-5 h-5" />
        </div>
        <h1 class="text-base font-bold text-zinc-100 tracking-tight">Routing Combos & Pipe Models</h1>
        <span class="px-2.5 py-0.5 rounded-full text-xs font-mono font-medium bg-purple-950/60 text-purple-400 border border-purple-800/50">
          {activeCount}/{models.length} active
        </span>
      </div>

      <!-- Quick Actions -->
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          onclick={loadData}
          disabled={loading}
          class="h-8 inline-flex items-center gap-1.5 px-3 text-xs font-medium rounded-lg border border-zinc-800 bg-zinc-900/90 hover:bg-zinc-800 text-zinc-300 transition-colors cursor-pointer disabled:opacity-50"
        >
          <RotateCw class="w-3.5 h-3.5 {loading ? 'animate-spin' : ''}" />
          <span>Refresh</span>
        </button>
        <button
          type="button"
          onclick={() => setAll(true)}
          class="h-8 inline-flex items-center gap-1.5 px-3 text-xs font-medium rounded-lg border border-zinc-800 bg-zinc-900/90 hover:bg-zinc-800 text-emerald-400 transition-colors cursor-pointer"
        >
          <Check class="w-3.5 h-3.5 text-emerald-400" />
          <span>Enable All</span>
        </button>
        <button
          type="button"
          onclick={() => setAll(false)}
          class="h-8 inline-flex items-center gap-1.5 px-3 text-xs font-medium rounded-lg border border-zinc-800 bg-zinc-900/90 hover:bg-zinc-800 text-zinc-400 transition-colors cursor-pointer"
        >
          <X class="w-3.5 h-3.5 text-zinc-400" />
          <span>Disable All</span>
        </button>
      </div>
    </div>

    <!-- Connected Providers Horizontal Scrollable Tabs Bar -->
    <div class="mt-3 pt-3 border-t border-zinc-800/60">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-1.5">
          <span class="w-2 h-2 rounded-full bg-purple-500 animate-pulse"></span>
          <span class="text-[11px] font-bold text-zinc-300 font-mono uppercase tracking-wider">
            Connected Providers ({providers.length})
          </span>
          <span class="text-[10px] text-zinc-500 font-mono hidden sm:inline">
            — click a tab to view specific AI models
          </span>
        </div>

        <!-- Left / Right Scroll Buttons -->
        <div class="flex items-center gap-1">
          <button
            type="button"
            onclick={() => scrollTabs('left')}
            class="p-1 rounded-md border border-zinc-800 bg-zinc-900/90 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition-colors cursor-pointer"
            title="Scroll tabs left"
          >
            <ChevronLeft class="w-3.5 h-3.5" />
          </button>
          <button
            type="button"
            onclick={() => scrollTabs('right')}
            class="p-1 rounded-md border border-zinc-800 bg-zinc-900/90 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 transition-colors cursor-pointer"
            title="Scroll tabs right"
          >
            <ChevronRight class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <!-- Horizontal Scrollable Container -->
      <div
        bind:this={tabsContainer}
        class="flex items-center gap-2 overflow-x-auto pb-1.5 scrollbar-thin scrollbar-thumb-zinc-800 scrollbar-track-transparent select-none"
        style="scroll-behavior: smooth;"
      >
        <!-- All Providers Tab -->
        <button
          type="button"
          onclick={() => {
            selectedProvider = 'all'
            page = 1
          }}
          class="flex-shrink-0 inline-flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {selectedProvider === 'all'
            ? 'bg-purple-600 text-white shadow-xs ring-1 ring-purple-500 font-bold'
            : 'border border-zinc-800 bg-zinc-900/80 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700 hover:bg-zinc-850'}"
        >
          <span>All Providers</span>
          <span class="px-1.5 py-0.2 rounded-full text-[10px] font-bold {selectedProvider === 'all' ? 'bg-purple-700 text-white' : 'bg-zinc-800 text-zinc-400'}">
            {models.length}
          </span>
        </button>

        <!-- Connected Provider Tabs -->
        {#each providers as p}
          {@const provModels = models.filter((m) => m.providerId === p)}
          {@const provCount = provModels.length}
          {@const provActive = provModels.filter((m) => m.isActive).length}
          {@const provName = provModels[0]?.providerName || p}
          <button
            type="button"
            onclick={() => {
              selectedProvider = p
              page = 1
            }}
            class="flex-shrink-0 inline-flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {selectedProvider === p
              ? 'bg-purple-600 text-white shadow-xs ring-1 ring-purple-500 font-bold'
              : 'border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:text-white hover:border-zinc-700 hover:bg-zinc-850'}"
            title="Filter by {provName} ({p})"
          >
            <span class="w-2 h-2 rounded-full {provActive > 0 ? 'bg-emerald-400' : 'bg-zinc-500'}"></span>
            <span class="font-semibold uppercase tracking-wide">{provName}</span>
            <span class="px-1.5 py-0.2 rounded-full text-[10px] font-bold {selectedProvider === p ? 'bg-purple-700 text-white' : 'bg-zinc-800 text-zinc-400'}">
              {provCount}
            </span>
          </button>
        {/each}
      </div>
    </div>
  </div>

  <!-- Search & Provider Filter Bar -->
  <div class="rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-3">
    <div class="flex flex-col sm:flex-row items-center gap-3">
      <div class="relative flex-1 w-full">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500 pointer-events-none" />
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Search by combo name, original model, or wire alias..."
          class="w-full h-8.5 pl-9 pr-3 rounded-lg border border-zinc-800 bg-zinc-900/90 text-xs font-mono text-zinc-200 placeholder:text-zinc-600 focus:border-purple-500/80 focus:outline-none focus:ring-1 focus:ring-purple-500/40 transition-all"
        />
      </div>

      <div class="flex items-center gap-2 w-full sm:w-auto shrink-0">
        <!-- Provider filter -->
        <select
          bind:value={selectedProvider}
          class="h-8.5 px-3 rounded-lg border border-zinc-800 bg-zinc-900/90 text-xs font-mono text-zinc-300 focus:border-purple-500/80 focus:outline-none cursor-pointer"
        >
          <option value="all">All Providers ({models.length})</option>
          {#each providers as p}
            <option value={p}>{p}</option>
          {/each}
        </select>

        <!-- Status filter -->
        <select
          bind:value={statusFilter}
          class="h-8.5 px-3 rounded-lg border border-zinc-800 bg-zinc-900/90 text-xs font-mono text-zinc-300 focus:border-purple-500/80 focus:outline-none cursor-pointer"
        >
          <option value="all">All Status</option>
          <option value="active">Active ({activeCount})</option>
          <option value="inactive">Inactive ({models.length - activeCount})</option>
        </select>
      </div>
    </div>
  </div>

  <!-- Subtitle Line: Showing X-Y of Total + CheapRouter Pagination (Directly below search, right above models list) -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-1 py-1 text-xs text-zinc-400 font-mono">
    <div class="flex items-center gap-2 flex-wrap">
      <span class="w-1.5 h-1.5 rounded-full bg-purple-500 animate-pulse"></span>
      <span>
        Showing <strong class="text-zinc-200 font-semibold">{filteredModels.length === 0 ? 0 : (page - 1) * pageSize + 1}–{Math.min(page * pageSize, filteredModels.length)}</strong> of <strong class="text-zinc-200 font-semibold">{filteredModels.length}</strong> combos ({pageSize} / page)
      </span>
      {#if selectedProvider !== 'all'}
        <span class="px-2 py-0.5 rounded text-[10px] font-bold bg-purple-950/80 border border-purple-800/60 text-purple-300 uppercase">
          {selectedProvider}
        </span>
      {/if}
    </div>

    <!-- Pagination Buttons -->
    <div class="flex items-center gap-1">
      <button
        type="button"
        onclick={() => (page = 1)}
        disabled={page <= 1}
        class="p-1.5 rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
        title="First page"
      >
        <ChevronsLeft class="w-3.5 h-3.5" />
      </button>
      <button
        type="button"
        onclick={() => (page = Math.max(1, page - 1))}
        disabled={page <= 1}
        class="px-2 py-1 rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 text-xs flex items-center gap-1 disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
      >
        <ChevronLeft class="w-3.5 h-3.5" />
        <span>Prev</span>
      </button>
      <span class="px-2.5 py-1 rounded border border-purple-500/40 bg-purple-950/40 text-purple-400 font-bold text-xs">
        {page}
      </span>
      <button
        type="button"
        onclick={() => (page = Math.min(totalPages, page + 1))}
        disabled={page >= totalPages}
        class="px-2 py-1 rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 text-xs flex items-center gap-1 disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
      >
        <span>Next</span>
        <ChevronRight class="w-3.5 h-3.5" />
      </button>
      <button
        type="button"
        onclick={() => (page = totalPages)}
        disabled={page >= totalPages}
        class="p-1.5 rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
        title="Last page"
      >
        <ChevronsRight class="w-3.5 h-3.5" />
      </button>
    </div>
  </div>

  <!-- Models List in CheapRouter Card Format -->
  {#if loading && models.length === 0}
    <div class="p-12 text-center text-zinc-500 font-mono text-xs">
      <RotateCw class="w-6 h-6 animate-spin mx-auto mb-2 text-purple-400" />
      Loading CheapRouter Pipe Models...
    </div>
  {:else if error}
    <div class="p-4 bg-red-500/10 border border-red-500/20 rounded-xl text-red-400 text-xs flex items-center gap-3">
      <AlertCircle class="w-5 h-5 shrink-0" />
      <div>{error}</div>
    </div>
  {:else if filteredModels.length === 0}
    <div class="p-12 text-center rounded-xl border border-zinc-800 bg-zinc-950/60">
      <Layers class="w-10 h-10 text-zinc-600 mx-auto mb-2 opacity-50" />
      <h3 class="text-sm font-semibold text-zinc-200">No Combos or Models Found</h3>
      <p class="text-xs text-zinc-500 mt-1">Connect providers or clear search filters to view models.</p>
    </div>
  {:else}
    <div class="space-y-3">
      {#each paginatedModels as item (item.providerId + '::' + item.originalModelId)}
        {@const key = item.providerId + '::' + item.originalModelId}
        {@const testRes = testResults[key]}
        <div
          class="rounded-xl border border-zinc-800/80 bg-zinc-950/70 p-4 transition-all duration-150 hover:border-zinc-700/80 shadow-sm {item.isActive ? '' : 'opacity-70 bg-zinc-950/40'}"
        >
          <!-- Top Row: Purple Toggle, Title, Strategy Pill, FREE badge, Save Button -->
          <div class="flex items-center justify-between gap-3">
            <div class="flex items-center gap-2.5 min-w-0 flex-wrap">
              <!-- Purple Toggle Switch -->
              <button
                type="button"
                role="switch"
                aria-checked={item.isActive}
                onclick={() => toggleActive(item)}
                class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none {item.isActive ? 'bg-purple-600' : 'bg-zinc-700'}"
                title={item.isActive ? 'Active — click to disable' : 'Disabled — click to enable'}
              >
                <span
                  class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {item.isActive ? 'translate-x-4' : 'translate-x-0'}"
                />
              </button>

              <!-- Bold Title -->
              <span class="font-bold text-sm text-zinc-100 font-mono truncate" title={item.customName || item.originalModelId}>
                {item.customName || item.originalModelId}
              </span>

              <!-- Provider Tag Pill -->
              <span class="px-2 py-0.5 rounded text-[11px] font-mono font-bold text-purple-300 border border-purple-800/60 bg-purple-950/60 uppercase">
                {item.providerId}
              </span>

              <!-- Strategy / Provider Pill -->
              <span class="px-2 py-0.5 rounded text-[11px] font-mono text-zinc-400 border border-zinc-700/60 bg-zinc-900/80">
                {item.providerName || 'priority'}
              </span>

              <!-- FREE Badge -->
              {#if item.isFree || (item.inputPrice === 0 && item.outputPrice === 0)}
                <span class="inline-flex items-center gap-1 text-[11px] font-semibold px-2 py-0.5 rounded border border-emerald-500/40 bg-emerald-500/10 text-emerald-400 font-mono">
                  <Check class="w-3 h-3 text-emerald-400" />
                  FREE
                </span>
              {:else}
                <span class="inline-flex items-center gap-1 text-[11px] font-semibold px-2 py-0.5 rounded border border-zinc-700/60 bg-zinc-800 text-zinc-400 font-mono">
                  PAID
                </span>
              {/if}
            </div>

            <!-- Right: Actions (Test, Save) -->
            <div class="flex items-center gap-2 shrink-0">
              <!-- Live Test Button -->
              <button
                type="button"
                onclick={() => runLiveTest(item)}
                disabled={testingKey === key}
                class="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium rounded-lg border border-zinc-700/70 bg-zinc-900 hover:bg-zinc-800 text-zinc-300 transition-colors cursor-pointer disabled:opacity-50"
                title="Send a 1-token live ping test"
              >
                <Play class="w-3 h-3 {testingKey === key ? 'animate-pulse text-amber-400' : 'text-emerald-400'}" />
                <span>{testingKey === key ? 'Testing...' : 'Test'}</span>
              </button>

              <!-- Save Button in Green Outline (CheapRouter Style) -->
              <button
                type="button"
                onclick={() => saveModel(item)}
                disabled={savingKey === key}
                class="inline-flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-lg border border-emerald-500/50 bg-emerald-950/40 hover:bg-emerald-900/60 text-emerald-400 transition-all cursor-pointer disabled:opacity-50 shadow-xs"
              >
                {#if saveSuccessKey === key}
                  <Check class="w-3.5 h-3.5 text-emerald-400" />
                  <span>Saved</span>
                {:else}
                  <Save class="w-3.5 h-3.5 text-emerald-400 {savingKey === key ? 'animate-spin' : ''}" />
                  <span>{savingKey === key ? 'Saving...' : 'Save'}</span>
                {/if}
              </button>
            </div>
          </div>

          <!-- Metadata Subtitle Row: ORIG NAME, ORIG ID, STRATEGY, PROVIDER -->
          <div class="flex flex-wrap items-center justify-between text-[11px] font-mono text-zinc-400 mt-2.5 pt-2 border-t border-zinc-800/50 gap-2">
            <div>
              <span class="text-zinc-500">ORIG NAME:</span> <span class="text-zinc-300 font-bold">{item.originalModelId}</span>
            </div>
            <div>
              <span class="text-zinc-500">ORIG ID:</span> <span class="text-zinc-300 font-mono">{item.providerId}::{item.originalModelId}</span>
            </div>
            <div>
              <span class="text-zinc-500">PROVIDER:</span> <span class="text-purple-400 font-bold uppercase">{item.providerName || item.providerId}</span>
            </div>
            <div>
              <span class="text-zinc-500">STRATEGY:</span> <span class="text-zinc-300 font-mono">priority</span>
            </div>
          </div>

          <!-- 6-Column Grid Layout (Exact CheapRouter Architecture) -->
          <div class="w-full grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3 mt-3 pt-3 border-t border-zinc-800/50">
            <!-- Column 1: Combo Name -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <GitFork class="w-3.5 h-3.5 text-zinc-400" />
                <span>Combo Name</span>
              </div>
              <div class="space-y-1.5">
                <div>
                  <div class="flex items-center gap-1 text-[10px] font-mono font-bold text-amber-500 tracking-wider mb-0.5">
                    <span>•</span><span>ORIGINAL</span>
                  </div>
                  <div class="text-xs font-mono text-zinc-400 truncate py-1" title={item.originalModelId}>
                    {item.originalModelId}
                  </div>
                </div>
                <div>
                  <div class="flex items-center gap-1 text-[10px] font-mono font-bold text-emerald-500 tracking-wider mb-0.5">
                    <span>•</span><span>CUSTOM</span>
                  </div>
                  <input
                    type="text"
                    bind:value={item.customName}
                    placeholder={item.originalModelId}
                    class="w-full bg-zinc-900/90 border border-zinc-800 focus:border-purple-500/70 focus:ring-1 focus:ring-purple-500/40 rounded px-2.5 py-1 text-xs font-mono text-zinc-200 placeholder:text-zinc-600"
                  />
                </div>
              </div>
            </div>

            <!-- Column 2: Combo ID / Alias -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <Layers class="w-3.5 h-3.5 text-zinc-400" />
                <span>Combo ID / Alias</span>
              </div>
              <div class="space-y-1.5">
                <div>
                  <div class="flex items-center gap-1 text-[10px] font-mono font-bold text-amber-500 tracking-wider mb-0.5">
                    <span>•</span><span>ORIGINAL</span>
                  </div>
                  <div class="text-xs font-mono text-zinc-400 truncate py-1" title={item.providerId + '/' + item.originalModelId}>
                    {item.providerId}/{item.originalModelId}
                  </div>
                </div>
                <div>
                  <div class="flex items-center gap-1 text-[10px] font-mono font-bold text-emerald-500 tracking-wider mb-0.5">
                    <span>•</span><span>CUSTOM</span>
                  </div>
                  <input
                    type="text"
                    bind:value={item.customAlias}
                    placeholder={item.originalModelId}
                    class="w-full bg-zinc-900/90 border border-zinc-800 focus:border-purple-500/70 focus:ring-1 focus:ring-purple-500/40 rounded px-2.5 py-1 text-xs font-mono text-zinc-200 placeholder:text-zinc-600"
                  />
                </div>
              </div>
            </div>

            <!-- Column 3: Context Window -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <Cpu class="w-3.5 h-3.5 text-zinc-400" />
                <span>Context Window</span>
              </div>
              <div class="space-y-1.5">
                <div>
                  <div class="flex items-center gap-1 text-[10px] font-mono font-bold text-amber-500 tracking-wider mb-0.5">
                    <span>•</span><span>ORIGINAL</span>
                  </div>
                  <div class="text-xs font-mono text-zinc-400 truncate py-1">
                    Inherited
                  </div>
                </div>
                <div>
                  <div class="flex items-center gap-1 text-[10px] font-mono font-bold text-emerald-500 tracking-wider mb-0.5">
                    <span>•</span><span>CUSTOM</span>
                  </div>
                  <input
                    type="text"
                    bind:value={item.contextWindow}
                    placeholder="e.g. 500k"
                    class="w-full bg-zinc-900/90 border border-zinc-800 focus:border-purple-500/70 focus:ring-1 focus:ring-purple-500/40 rounded px-2.5 py-1 text-xs font-mono text-zinc-200 placeholder:text-zinc-600"
                  />
                </div>
              </div>
            </div>

            <!-- Column 4: Input ($/1M) -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <ArrowDown class="w-3.5 h-3.5 text-zinc-400" />
                <span>Input ($/1M)</span>
              </div>
              <div class="space-y-1.5">
                <div>
                  <div class="flex items-center gap-1 text-[10px] font-mono font-bold text-amber-500 tracking-wider mb-0.5">
                    <span>•</span><span>ORIG (CUT-BAL)</span>
                  </div>
                  <input
                    type="number"
                    step="0.01"
                    bind:value={item.origInputPrice}
                    placeholder="e.g. 1.60"
                    class="w-full bg-zinc-900/50 border border-zinc-800/80 rounded px-2.5 py-1 text-xs font-mono text-zinc-400 placeholder:text-zinc-600"
                  />
                </div>
                <div>
                  <div class="flex items-center justify-between text-[10px] font-mono font-bold mb-0.5">
                    <div class="flex items-center gap-1 text-emerald-500 tracking-wider">
                      <span>•</span><span>CUSTOM (RATE)</span>
                    </div>
                    <span class="text-emerald-400 font-semibold">$0 FREE</span>
                  </div>
                  <input
                    type="number"
                    step="0.01"
                    bind:value={item.inputPrice}
                    placeholder="0"
                    class="w-full bg-zinc-900/90 border border-zinc-800 focus:border-purple-500/70 focus:ring-1 focus:ring-purple-500/40 rounded px-2.5 py-1 text-xs font-mono text-zinc-200 placeholder:text-zinc-600"
                  />
                </div>
              </div>
            </div>

            <!-- Column 5: Cache ($/1M) -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <RotateCw class="w-3.5 h-3.5 text-zinc-400" />
                <span>Cache ($/1M)</span>
              </div>
              <div class="space-y-1.5">
                <div>
                  <div class="flex items-center gap-1 text-[10px] font-mono font-bold text-amber-500 tracking-wider mb-0.5">
                    <span>•</span><span>ORIG (CUT-BAL)</span>
                  </div>
                  <input
                    type="number"
                    step="0.01"
                    bind:value={item.origCachePrice}
                    placeholder="e.g. 0.25"
                    class="w-full bg-zinc-900/50 border border-zinc-800/80 rounded px-2.5 py-1 text-xs font-mono text-zinc-400 placeholder:text-zinc-600"
                  />
                </div>
                <div>
                  <div class="flex items-center justify-between text-[10px] font-mono font-bold mb-0.5">
                    <div class="flex items-center gap-1 text-emerald-500 tracking-wider">
                      <span>•</span><span>CUSTOM (RATE)</span>
                    </div>
                    <span class="text-emerald-400 font-semibold">$0 FREE</span>
                  </div>
                  <input
                    type="number"
                    step="0.01"
                    bind:value={item.cachePrice}
                    placeholder="0"
                    class="w-full bg-zinc-900/90 border border-zinc-800 focus:border-purple-500/70 focus:ring-1 focus:ring-purple-500/40 rounded px-2.5 py-1 text-xs font-mono text-zinc-200 placeholder:text-zinc-600"
                  />
                </div>
              </div>
            </div>

            <!-- Column 6: Output ($/1M) -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <ArrowUp class="w-3.5 h-3.5 text-zinc-400" />
                <span>Output ($/1M)</span>
              </div>
              <div class="space-y-1.5">
                <div>
                  <div class="flex items-center gap-1 text-[10px] font-mono font-bold text-amber-500 tracking-wider mb-0.5">
                    <span>•</span><span>ORIG (CUT-BAL)</span>
                  </div>
                  <input
                    type="number"
                    step="0.01"
                    bind:value={item.origOutputPrice}
                    placeholder="e.g. 3.00"
                    class="w-full bg-zinc-900/50 border border-zinc-800/80 rounded px-2.5 py-1 text-xs font-mono text-zinc-400 placeholder:text-zinc-600"
                  />
                </div>
                <div>
                  <div class="flex items-center justify-between text-[10px] font-mono font-bold mb-0.5">
                    <div class="flex items-center gap-1 text-emerald-500 tracking-wider">
                      <span>•</span><span>CUSTOM (RATE)</span>
                    </div>
                    <span class="text-emerald-400 font-semibold">$0 FREE</span>
                  </div>
                  <input
                    type="number"
                    step="0.01"
                    bind:value={item.outputPrice}
                    placeholder="0"
                    class="w-full bg-zinc-900/90 border border-zinc-800 focus:border-purple-500/70 focus:ring-1 focus:ring-purple-500/40 rounded px-2.5 py-1 text-xs font-mono text-zinc-200 placeholder:text-zinc-600"
                  />
                </div>
              </div>
            </div>
          </div>

          <!-- Live Test Result Banner -->
          {#if testRes}
            <div
              class="mt-3 pt-2.5 border-t border-zinc-800/50 flex items-center justify-between text-xs font-mono {testRes.ok
                ? 'text-emerald-400'
                : 'text-red-400'}"
            >
              <div class="flex items-center gap-2 truncate">
                <span class="material-symbols-outlined text-sm">{testRes.ok ? 'check_circle' : 'error'}</span>
                <span class="truncate">{testRes.ok ? testRes.response : testRes.error}</span>
              </div>
              {#if testRes.latencyMs}
                <span class="text-zinc-500 shrink-0 ml-2">{testRes.latencyMs}ms</span>
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>
