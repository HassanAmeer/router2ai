<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Wallet,
    Search,
    Plus,
    RotateCw,
    AlertCircle,
    Check,
    Coins,
    Sliders,
    Layers,
    Trash2,
    X,
    Activity,
    Clock,
    ShieldAlert,
    TrendingUp,
    ChevronDown,
    Zap,
    Save,
    Play,
    ShieldCheck,
  } from 'lucide-svelte'
  import {
    api,
    type ModelLimitPolicy,
    type LimitsTargetOption,
  } from '../api/client'
  import Card from '../lib/ui/Card.svelte'
  import Button from '../lib/ui/Button.svelte'
  import Badge from '../lib/ui/Badge.svelte'
  import Modal from '../lib/ui/Modal.svelte'

  let limits = $state<ModelLimitPolicy[]>([])
  let targets = $state<LimitsTargetOption[]>([])
  let loading = $state(true)
  let error = $state('')

  let searchQuery = $state('')
  let targetTypeFilter = $state<'all' | 'model' | 'combo'>('all')
  let statusFilter = $state<'all' | 'active' | 'paused' | 'exhausted'>('all')
  let selectedProvider = $state('all')

  // Pagination
  let currentPage = $state(1)
  const pageSize = 20

  // Per-item inline edit state
  let edits = $state<
    Record<
      string,
      {
        totalTokenPoolM: number
        rpmLimit: number
        tpmLimit: number
        resetWindow: 'lifetime' | 'monthly' | 'weekly' | 'daily'
        onExhaustion: 'error' | 'fallback'
        fallbackTarget: string
        customErrorMessage: string
      }
    >
  >({})
  let savingId = $state<string | null>(null)
  let savedId = $state<string | null>(null)

  // Create / Edit Modal
  let isModalOpen = $state(false)
  let modalTarget = $state('')
  let modalTokenPoolM = $state(10) // default 10M
  let modalRPMLimit = $state(0)
  let modalTPMLimit = $state(0)
  let modalResetWindow = $state<'lifetime' | 'monthly' | 'weekly' | 'daily'>('lifetime')
  let modalOnExhaustion = $state<'error' | 'fallback'>('error')
  let modalFallbackTarget = $state('')
  let modalCustomError = $state('')
  let isSaving = $state(false)

  // Refill Modal
  let isRefillOpen = $state(false)
  let refillTargetLimit = $state<ModelLimitPolicy | null>(null)
  let refillAmountM = $state(10) // 10M default
  let isRefilling = $state(false)

  function formatTokens(n: number): string {
    if (!n || n <= 0) return '0M'
    const m = n / 1_000_000
    if (m >= 1000) {
      const b = n / 1_000_000_000
      return `${m.toLocaleString()}M (${b.toFixed(2)}B)`
    }
    if (m >= 1) return `${m.toFixed(m % 1 === 0 ? 0 : 2)}M`
    if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
    return `${n}`
  }

  function syncEdits(items: ModelLimitPolicy[]) {
    for (const lim of items) {
      if (!edits[lim.id]) {
        edits[lim.id] = {
          totalTokenPoolM: Number((lim.totalTokenPool / 1_000_000).toFixed(2)),
          rpmLimit: lim.rpmLimit || 0,
          tpmLimit: lim.tpmLimit || 0,
          resetWindow: lim.resetWindow || 'lifetime',
          onExhaustion: lim.onExhaustion || 'error',
          fallbackTarget: lim.fallbackTarget || '',
          customErrorMessage: lim.customErrorMessage || '',
        }
      }
    }
  }

  async function loadData() {
    loading = true
    error = ''
    try {
      const res = await api.getLimits()
      limits = res.limits || []
      targets = res.targets || []
      syncEdits(limits)
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load limit policies'
    } finally {
      loading = false
    }
  }

  onMount(() => {
    loadData()
  })

  // Providers derived
  let providers = $derived.by(() => {
    const set = new Set<string>()
    for (const l of limits) {
      if (l.providerId) set.add(l.providerId)
    }
    return Array.from(set).sort()
  })

  // Summary Metrics
  let totalAllocated = $derived(limits.reduce((acc, l) => acc + (l.totalTokenPool || 0), 0))
  let totalConsumed = $derived(limits.reduce((acc, l) => acc + (l.consumedTokens || 0), 0))
  let activePoliciesCount = $derived(limits.filter((l) => l.isActive).length)

  // Filtered limits
  let filteredLimits = $derived.by(() => {
    let list = limits
    if (targetTypeFilter !== 'all') {
      list = list.filter((l) => l.targetType === targetTypeFilter)
    }
    if (selectedProvider !== 'all') {
      list = list.filter((l) => l.providerId === selectedProvider)
    }
    if (statusFilter === 'active') {
      list = list.filter((l) => l.isActive && (l.totalTokenPool === 0 || l.consumedTokens < l.totalTokenPool))
    } else if (statusFilter === 'paused') {
      list = list.filter((l) => !l.isActive)
    } else if (statusFilter === 'exhausted') {
      list = list.filter((l) => l.consumedTokens >= l.totalTokenPool && l.totalTokenPool > 0)
    }

    const q = searchQuery.trim().toLowerCase()
    if (q) {
      list = list.filter(
        (l) =>
          l.targetName.toLowerCase().includes(q) ||
          l.targetId.toLowerCase().includes(q) ||
          (l.providerId && l.providerId.toLowerCase().includes(q))
      )
    }
    return list
  })

  // Pagination calculation
  let totalPages = $derived(Math.max(1, Math.ceil(filteredLimits.length / pageSize)))
  let paginatedLimits = $derived.by(() => {
    const start = (currentPage - 1) * pageSize
    return filteredLimits.slice(start, start + pageSize)
  })

  function openCreateModal() {
    modalTarget = targets.length > 0 ? `${targets[0].type}::${targets[0].id}` : ''
    modalTokenPoolM = 10
    modalRPMLimit = 0
    modalTPMLimit = 0
    modalResetWindow = 'lifetime'
    modalOnExhaustion = 'error'
    modalFallbackTarget = ''
    modalCustomError = 'Quota exhausted for this model. Please refill or upgrade.'
    isModalOpen = true
  }

  async function handleSaveLimit() {
    if (!modalTarget) return
    isSaving = true
    try {
      const parts = modalTarget.split('::')
      const targetType = parts[0] as 'model' | 'combo'
      const targetId = parts[1]
      const chosen = targets.find((t) => t.type === targetType && t.id === targetId)

      await api.saveLimit({
        targetType,
        targetId,
        targetName: chosen ? chosen.name : targetId,
        providerId: chosen?.providerId || '',
        totalTokenPool: Math.floor(modalTokenPoolM * 1_000_000),
        rpmLimit: Number(modalRPMLimit) || 0,
        tpmLimit: Number(modalTPMLimit) || 0,
        resetWindow: modalResetWindow,
        onExhaustion: modalOnExhaustion,
        fallbackTarget: modalFallbackTarget,
        customErrorMessage: modalCustomError,
        isActive: true,
      })

      isModalOpen = false
      await loadData()
    } catch (err) {
      alert('Failed to save limit: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      isSaving = false
    }
  }

  async function saveSingleItem(lim: ModelLimitPolicy) {
    const draft = edits[lim.id]
    if (!draft) return
    savingId = lim.id
    try {
      const newPool = Math.floor(draft.totalTokenPoolM * 1_000_000)
      await api.saveLimit({
        id: lim.id,
        targetType: lim.targetType,
        targetId: lim.targetId,
        targetName: lim.targetName,
        providerId: lim.providerId,
        totalTokenPool: newPool,
        rpmLimit: Number(draft.rpmLimit) || 0,
        tpmLimit: Number(draft.tpmLimit) || 0,
        resetWindow: draft.resetWindow,
        onExhaustion: draft.onExhaustion,
        fallbackTarget: draft.fallbackTarget,
        customErrorMessage: draft.customErrorMessage,
        isActive: lim.isActive,
      })

      lim.totalTokenPool = newPool
      lim.rpmLimit = Number(draft.rpmLimit) || 0
      lim.tpmLimit = Number(draft.tpmLimit) || 0
      lim.resetWindow = draft.resetWindow
      lim.onExhaustion = draft.onExhaustion
      lim.fallbackTarget = draft.fallbackTarget

      savedId = lim.id
      setTimeout(() => {
        if (savedId === lim.id) savedId = null
      }, 2000)
    } catch (err) {
      alert('Save failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      savingId = null
    }
  }

  async function handleToggle(lim: ModelLimitPolicy) {
    const next = !lim.isActive
    lim.isActive = next
    try {
      await api.toggleLimit(lim.id, next)
    } catch {
      lim.isActive = !next
    }
  }

  async function handleEnableAll() {
    for (const l of filteredLimits) {
      if (!l.isActive) {
        l.isActive = true
        api.toggleLimit(l.id, true).catch(() => {})
      }
    }
  }

  async function handleDisableAll() {
    for (const l of filteredLimits) {
      if (l.isActive) {
        l.isActive = false
        api.toggleLimit(l.id, false).catch(() => {})
      }
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Are you sure you want to delete this limit policy?')) return
    try {
      await api.deleteLimit(id)
      limits = limits.filter((l) => l.id !== id)
      delete edits[id]
    } catch (err) {
      alert('Delete failed: ' + (err instanceof Error ? err.message : String(err)))
    }
  }

  function openRefill(lim: ModelLimitPolicy) {
    refillTargetLimit = lim
    refillAmountM = 10
    isRefillOpen = true
  }

  async function submitRefill() {
    if (!refillTargetLimit) return
    isRefilling = true
    try {
      const addTokens = Math.floor(refillAmountM * 1_000_000)
      await api.refillLimit({
        id: refillTargetLimit.id,
        tokensToAdd: addTokens,
      })
      refillTargetLimit.totalTokenPool += addTokens
      if (edits[refillTargetLimit.id]) {
        edits[refillTargetLimit.id].totalTokenPoolM = Number((refillTargetLimit.totalTokenPool / 1_000_000).toFixed(2))
      }
      isRefillOpen = false
    } catch (err) {
      alert('Refill failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      isRefilling = false
    }
  }
</script>

<div class="w-full space-y-4">
  <!-- CheapRouter Style Top Header -->
  <div class="w-full rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm">
    <div class="flex flex-col md:flex-row md:items-center justify-between gap-3">
      <!-- Title + Active Badge -->
      <div class="flex items-center gap-2.5">
        <button
          type="button"
          class="p-1 rounded text-zinc-400 hover:text-zinc-200 transition-colors"
          title="Limits Bank"
        >
          <ChevronDown class="w-4 h-4" />
        </button>
        <div class="p-1.5 rounded-lg bg-purple-950/80 border border-purple-800/60 text-purple-400">
          <Wallet class="w-5 h-5" />
        </div>
        <div class="flex items-center gap-2 flex-wrap">
          <h1 class="text-base font-bold text-zinc-100 font-mono tracking-tight">
            Limits Bank & Token Budget
          </h1>
          <span class="px-2 py-0.5 rounded-full text-xs font-mono font-semibold bg-purple-950/90 text-purple-300 border border-purple-800/70">
            {activePoliciesCount}/{limits.length} active
          </span>
        </div>
      </div>

      <!-- Quick Action Buttons -->
      <div class="flex items-center gap-2 flex-wrap">
        <button
          type="button"
          onclick={loadData}
          disabled={loading}
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-mono font-medium border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:bg-zinc-800 hover:text-white transition-all cursor-pointer"
        >
          <RotateCw class="w-3.5 h-3.5 {loading ? 'animate-spin' : ''}" />
          <span>Refresh</span>
        </button>

        <button
          type="button"
          onclick={openCreateModal}
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-mono font-medium border border-purple-700/80 bg-purple-900/40 text-purple-200 hover:bg-purple-800/60 hover:text-white transition-all cursor-pointer shadow-xs"
        >
          <Plus class="w-3.5 h-3.5 text-purple-300" />
          <span>New Policy</span>
        </button>

        <button
          type="button"
          onclick={handleEnableAll}
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-mono font-medium border border-emerald-900/60 bg-emerald-950/40 text-emerald-300 hover:bg-emerald-900/60 transition-all cursor-pointer"
        >
          <Check class="w-3.5 h-3.5" />
          <span>Enable All</span>
        </button>

        <button
          type="button"
          onclick={handleDisableAll}
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-mono font-medium border border-zinc-800 bg-zinc-900/80 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-200 transition-all cursor-pointer"
        >
          <X class="w-3.5 h-3.5" />
          <span>Disable All</span>
        </button>
      </div>
    </div>

    <!-- Provider & Category Filter Chips Row -->
    <div class="flex flex-wrap items-center gap-2 mt-3 pt-3 border-t border-zinc-800/60">
      <span class="text-[11px] font-bold text-zinc-400 font-mono uppercase tracking-wider flex items-center gap-1.5 mr-1">
        <span class="w-2 h-2 rounded-full bg-purple-500 animate-pulse"></span>
        Target Types & Providers:
      </span>

      <!-- All Pill -->
      <button
        type="button"
        onclick={() => {
          targetTypeFilter = 'all'
          selectedProvider = 'all'
        }}
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {targetTypeFilter === 'all' && selectedProvider === 'all'
          ? 'bg-purple-600 text-white shadow-xs'
          : 'border border-zinc-800 bg-zinc-900/80 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700'}"
      >
        <span>All</span>
        <span class="px-1.5 py-0.2 rounded-full text-[10px] {targetTypeFilter === 'all' && selectedProvider === 'all' ? 'bg-purple-700 text-white' : 'bg-zinc-800 text-zinc-400'}">
          {limits.length}
        </span>
      </button>

      <!-- Models Category Pill -->
      <button
        type="button"
        onclick={() => {
          targetTypeFilter = targetTypeFilter === 'model' ? 'all' : 'model'
          selectedProvider = 'all'
        }}
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {targetTypeFilter === 'model'
          ? 'bg-purple-600 text-white shadow-xs'
          : 'border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:text-white hover:border-zinc-700'}"
      >
        <span class="w-2 h-2 rounded-full bg-blue-400"></span>
        <span>MODELS</span>
        <span class="px-1.5 py-0.2 rounded-full text-[10px] {targetTypeFilter === 'model' ? 'bg-purple-700 text-white' : 'bg-zinc-800 text-zinc-400'}">
          {limits.filter((l) => l.targetType === 'model').length}
        </span>
      </button>

      <!-- Combos Category Pill -->
      <button
        type="button"
        onclick={() => {
          targetTypeFilter = targetTypeFilter === 'combo' ? 'all' : 'combo'
          selectedProvider = 'all'
        }}
        class="inline-flex items-center gap-1.5 px-3 py-1 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {targetTypeFilter === 'combo'
          ? 'bg-purple-600 text-white shadow-xs'
          : 'border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:text-white hover:border-zinc-700'}"
      >
        <span class="w-2 h-2 rounded-full bg-purple-400"></span>
        <span>COMBOS</span>
        <span class="px-1.5 py-0.2 rounded-full text-[10px] {targetTypeFilter === 'combo' ? 'bg-purple-700 text-white' : 'bg-zinc-800 text-zinc-400'}">
          {limits.filter((l) => l.targetType === 'combo').length}
        </span>
      </button>

      <!-- Provider Specific Pills -->
      {#each providers as p}
        {@const provCount = limits.filter((m) => m.providerId === p).length}
        {@const provActive = limits.filter((m) => m.providerId === p && m.isActive).length}
        <button
          type="button"
          onclick={() => {
            selectedProvider = selectedProvider === p ? 'all' : p
            targetTypeFilter = 'all'
          }}
          class="inline-flex items-center gap-2 px-3 py-1 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {selectedProvider === p
            ? 'bg-purple-600 text-white shadow-xs'
            : 'border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:text-white hover:border-zinc-700'}"
          title="Filter by {p}"
        >
          <span class="w-2 h-2 rounded-full {provActive > 0 ? 'bg-emerald-400' : 'bg-zinc-500'}"></span>
          <span class="font-semibold uppercase tracking-wide">{p}</span>
          <span class="px-1.5 py-0.2 rounded-full text-[10px] {selectedProvider === p ? 'bg-purple-700 text-white' : 'bg-zinc-800 text-zinc-400'}">
            {provCount}
          </span>
        </button>
      {/each}
    </div>

    <!-- Subtitle Line: Showing X-Y of Total + CheapRouter Pagination -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mt-3 pt-3 border-t border-zinc-800/60 text-xs text-zinc-400 font-mono">
      <div>
        Showing <span class="text-zinc-200 font-semibold">{filteredLimits.length === 0 ? 0 : (currentPage - 1) * pageSize + 1}–{Math.min(currentPage * pageSize, filteredLimits.length)}</span> of <span class="text-zinc-200 font-semibold">{filteredLimits.length}</span> limit policies
      </div>

      <!-- Pagination Buttons -->
      <div class="flex items-center gap-1">
        <button
          type="button"
          disabled={currentPage <= 1}
          onclick={() => (currentPage = 1)}
          class="px-2 py-1 rounded border border-zinc-800 bg-zinc-900/80 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed text-zinc-300 transition-colors"
          title="First page"
        >
          &vert;&lt;
        </button>
        <button
          type="button"
          disabled={currentPage <= 1}
          onclick={() => (currentPage = Math.max(1, currentPage - 1))}
          class="px-2 py-1 rounded border border-zinc-800 bg-zinc-900/80 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed text-zinc-300 transition-colors"
          title="Previous page"
        >
          &lt; Prev
        </button>
        <span class="px-3 py-1 rounded border border-purple-800/80 bg-purple-950/60 text-purple-200 font-bold">
          {currentPage}
        </span>
        <button
          type="button"
          disabled={currentPage >= totalPages}
          onclick={() => (currentPage = Math.min(totalPages, currentPage + 1))}
          class="px-2 py-1 rounded border border-zinc-800 bg-zinc-900/80 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed text-zinc-300 transition-colors"
          title="Next page"
        >
          Next &gt;
        </button>
        <button
          type="button"
          disabled={currentPage >= totalPages}
          onclick={() => (currentPage = totalPages)}
          class="px-2 py-1 rounded border border-zinc-800 bg-zinc-900/80 hover:bg-zinc-800 disabled:opacity-30 disabled:cursor-not-allowed text-zinc-300 transition-colors"
          title="Last page"
        >
          &gt;&vert;
        </button>
      </div>
    </div>
  </div>

  <!-- Cyberpunk Summary Metric Cards -->
  <div class="w-full grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
    <!-- Total Allocated Pool -->
    <div class="rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm flex items-center gap-3.5">
      <div class="w-11 h-11 rounded-xl bg-blue-500/10 border border-blue-500/30 text-blue-400 flex items-center justify-center flex-shrink-0">
        <Wallet class="w-5 h-5" />
      </div>
      <div>
        <div class="text-[11px] font-mono font-medium text-zinc-400 uppercase tracking-wide">Total Allocated Pool</div>
        <div class="text-xl font-bold font-mono text-zinc-100 mt-0.5">
          {formatTokens(totalAllocated)}
        </div>
      </div>
    </div>

    <!-- Consumed Tokens -->
    <div class="rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm flex items-center gap-3.5">
      <div class="w-11 h-11 rounded-xl bg-amber-500/10 border border-amber-500/30 text-amber-400 flex items-center justify-center flex-shrink-0">
        <TrendingUp class="w-5 h-5" />
      </div>
      <div>
        <div class="text-[11px] font-mono font-medium text-zinc-400 uppercase tracking-wide">Consumed Tokens</div>
        <div class="text-xl font-bold font-mono text-zinc-100 mt-0.5">
          {formatTokens(totalConsumed)}
        </div>
      </div>
    </div>

    <!-- Remaining Balance -->
    <div class="rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm flex items-center gap-3.5">
      <div class="w-11 h-11 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 flex items-center justify-center flex-shrink-0">
        <Coins class="w-5 h-5" />
      </div>
      <div>
        <div class="text-[11px] font-mono font-medium text-zinc-400 uppercase tracking-wide">Remaining Balance</div>
        <div class="text-xl font-bold font-mono text-emerald-400 mt-0.5">
          {formatTokens(Math.max(0, totalAllocated - totalConsumed))}
        </div>
      </div>
    </div>

    <!-- Active Policies -->
    <div class="rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm flex items-center gap-3.5">
      <div class="w-11 h-11 rounded-xl bg-purple-500/10 border border-purple-500/30 text-purple-400 flex items-center justify-center flex-shrink-0">
        <ShieldCheck class="w-5 h-5" />
      </div>
      <div>
        <div class="text-[11px] font-mono font-medium text-zinc-400 uppercase tracking-wide">Active Policies</div>
        <div class="text-xl font-bold font-mono text-purple-300 mt-0.5">
          {activePoliciesCount} / {limits.length}
        </div>
      </div>
    </div>
  </div>

  <!-- Search & Filter Controls -->
  <div class="w-full rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-3 shadow-sm">
    <div class="flex flex-col sm:flex-row items-center gap-3">
      <!-- Search Input -->
      <div class="relative flex-1 w-full">
        <Search class="w-4 h-4 text-zinc-400 absolute left-3 top-1/2 -translate-y-1/2" />
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Search by target name, original model, or provider..."
          class="w-full pl-9 pr-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900/90 text-zinc-200 placeholder-zinc-500 text-xs font-mono focus:outline-none focus:border-purple-500 transition-colors"
        />
      </div>

      <!-- Target Category Filter -->
      <div class="w-full sm:w-auto">
        <select
          bind:value={targetTypeFilter}
          class="w-full sm:w-auto px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900/90 text-zinc-200 text-xs font-mono focus:outline-none focus:border-purple-500"
        >
          <option value="all">All Targets ({limits.length})</option>
          <option value="model">Models Only</option>
          <option value="combo">Combos Only</option>
        </select>
      </div>

      <!-- Status Filter -->
      <div class="w-full sm:w-auto">
        <select
          bind:value={statusFilter}
          class="w-full sm:w-auto px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900/90 text-zinc-200 text-xs font-mono focus:outline-none focus:border-purple-500"
        >
          <option value="all">All Status</option>
          <option value="active">Active & Healthy</option>
          <option value="paused">Paused</option>
          <option value="exhausted">Exhausted</option>
        </select>
      </div>
    </div>
  </div>

  <!-- Limit Policies List: CheapRouter 6-Column Card Architecture -->
  {#if loading && limits.length === 0}
    <div class="p-12 text-center text-zinc-400 font-mono text-sm">
      <RotateCw class="w-6 h-6 animate-spin mx-auto mb-2 text-purple-400" />
      Loading Limits Bank Policies...
    </div>
  {:else if error}
    <div class="p-4 rounded-xl bg-red-950/40 border border-red-900/60 text-red-300 font-mono text-xs flex items-center gap-3">
      <AlertCircle class="w-5 h-5 flex-shrink-0 text-red-400" />
      <div>{error}</div>
    </div>
  {:else if filteredLimits.length === 0}
    <div class="w-full rounded-xl border border-zinc-800/80 bg-zinc-950/60 p-12 text-center">
      <Wallet class="w-12 h-12 text-zinc-600 mx-auto mb-3 opacity-60" />
      <h3 class="text-sm font-semibold font-mono text-zinc-200">No Limit Policies Configured</h3>
      <p class="text-xs text-zinc-400 font-mono mt-1 mb-4">
        Add your first model or combo token limit policy to prevent runaway quota usage.
      </p>
      <button
        type="button"
        onclick={openCreateModal}
        class="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-mono font-medium border border-purple-700 bg-purple-900/40 text-purple-200 hover:bg-purple-800/60 hover:text-white transition-all cursor-pointer shadow-xs"
      >
        <Plus class="w-4 h-4 text-purple-300" />
        <span>Configure Limit</span>
      </button>
    </div>
  {:else}
    <div class="w-full space-y-3">
      {#each paginatedLimits as lim (lim.id)}
        {@const draft = edits[lim.id]}
        {@const percent = lim.totalTokenPool > 0 ? Math.min(100, Math.round((lim.consumedTokens / lim.totalTokenPool) * 100)) : 0}
        {@const isExhausted = lim.totalTokenPool > 0 && lim.consumedTokens >= lim.totalTokenPool}
        {@const barColor = percent >= 90 ? 'bg-red-500' : percent >= 75 ? 'bg-amber-500' : 'bg-emerald-500'}

        <div
          class="w-full rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm hover:border-zinc-700/80 transition-all {lim.isActive
            ? ''
            : 'opacity-70'}"
        >
          <!-- Card Header: Purple Toggle Switch, Title, Provider Tag, Status Badge, Actions -->
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-3 border-b border-zinc-800/60">
            <!-- Left Side: Switch + Title + Badges -->
            <div class="flex items-center gap-3 flex-wrap">
              <!-- Purple Switch Toggle -->
              <button
                type="button"
                role="switch"
                aria-checked={lim.isActive}
                onclick={() => handleToggle(lim)}
                class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none {lim.isActive
                  ? 'bg-purple-600'
                  : 'bg-zinc-700'}"
                title={lim.isActive ? 'Policy is Active' : 'Policy is Paused'}
              >
                <span
                  class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow-sm ring-0 transition duration-200 ease-in-out {lim.isActive
                    ? 'translate-x-4'
                    : 'translate-x-0'}"
                ></span>
              </button>

              <!-- Target Name Title -->
              <span class="text-sm font-bold text-zinc-100 font-mono tracking-tight">
                {lim.targetName || lim.targetId}
              </span>

              <!-- Provider Tag Pill -->
              {#if lim.providerId}
                <span class="px-2 py-0.5 rounded text-[11px] font-mono font-bold text-purple-300 border border-purple-800/60 bg-purple-950/60 uppercase">
                  {lim.providerId}
                </span>
              {/if}

              <!-- Target Type Pill -->
              <span class="px-2 py-0.5 rounded text-[11px] font-mono text-zinc-300 border border-zinc-700/60 bg-zinc-900/80 uppercase">
                {lim.targetType}
              </span>

              <!-- Status Badge -->
              {#if isExhausted}
                <span class="px-2 py-0.5 rounded text-[11px] font-mono font-semibold text-red-400 border border-red-800/60 bg-red-950/60">
                  EXHAUSTED
                </span>
              {:else if !lim.isActive}
                <span class="px-2 py-0.5 rounded text-[11px] font-mono text-zinc-400 border border-zinc-700/60 bg-zinc-900/80">
                  PAUSED
                </span>
              {:else}
                <span class="px-2 py-0.5 rounded text-[11px] font-mono font-semibold text-emerald-400 border border-emerald-800/60 bg-emerald-950/60 flex items-center gap-1">
                  <Check class="w-3 h-3" /> ACTIVE
                </span>
              {/if}
            </div>

            <!-- Right Side: Refill Button, Save Button, Delete Button -->
            <div class="flex items-center gap-2 flex-shrink-0 self-end sm:self-center">
              <!-- Quick Refill Button (Amber) -->
              <button
                type="button"
                onclick={() => openRefill(lim)}
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-mono font-semibold border border-amber-800/70 bg-amber-950/40 text-amber-300 hover:bg-amber-900/60 hover:text-white transition-all cursor-pointer shadow-xs"
              >
                <Coins class="w-3.5 h-3.5 text-amber-400" />
                <span>Refill</span>
              </button>

              <!-- Green Outline Save Button -->
              <button
                type="button"
                onclick={() => saveSingleItem(lim)}
                disabled={savingId === lim.id}
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-mono font-semibold border transition-all cursor-pointer {savedId ===
                lim.id
                  ? 'border-emerald-500 bg-emerald-950/60 text-emerald-300'
                  : 'border-emerald-600/80 bg-emerald-950/20 text-emerald-400 hover:bg-emerald-900/40 hover:text-emerald-300'}"
              >
                {#if savingId === lim.id}
                  <RotateCw class="w-3.5 h-3.5 animate-spin" />
                  <span>Saving...</span>
                {:else if savedId === lim.id}
                  <Check class="w-3.5 h-3.5 text-emerald-400" />
                  <span>Saved!</span>
                {:else}
                  <Save class="w-3.5 h-3.5" />
                  <span>Save</span>
                {/if}
              </button>

              <!-- Delete Button -->
              <button
                type="button"
                onclick={() => handleDelete(lim.id)}
                class="p-1.5 text-zinc-500 hover:text-red-400 hover:bg-red-950/30 rounded border border-transparent hover:border-red-900/40 transition-colors"
                title="Delete policy"
              >
                <Trash2 class="w-4 h-4" />
              </button>
            </div>
          </div>

          <!-- Metadata Subtitle Row: TARGET ID, PROVIDER, RESET, EXHAUSTION -->
          <div class="flex flex-wrap items-center justify-between text-[11px] font-mono text-zinc-400 mt-2.5 pt-2 border-t border-zinc-800/50 gap-2">
            <div>
              <span class="text-zinc-500">TARGET ID:</span> <span class="text-zinc-300 font-bold">{lim.targetId}</span>
            </div>
            <div>
              <span class="text-zinc-500">PROVIDER:</span> <span class="text-purple-400 font-bold uppercase">{lim.providerId || 'CUSTOM'}</span>
            </div>
            <div>
              <span class="text-zinc-500">RESET CYCLE:</span> <span class="text-zinc-300 capitalize">{lim.resetWindow}</span>
            </div>
            <div>
              <span class="text-zinc-500">ACTION ON EXHAUST:</span> <span class="text-zinc-300 uppercase">{lim.onExhaustion}</span>
              {#if lim.onExhaustion === 'fallback' && lim.fallbackTarget}
                <span class="text-primary-400 font-mono ml-1">&rarr; {lim.fallbackTarget}</span>
              {/if}
            </div>
          </div>

          <!-- 6-Column Grid Layout (Exact CheapRouter Architecture) -->
          <div class="w-full grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3 mt-3 pt-3 border-t border-zinc-800/50">
            <!-- Column 1: Allocated Token Pool -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <Wallet class="w-3.5 h-3.5 text-purple-400" />
                <span>Token Pool</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                ALLOCATED (TOTAL)
              </div>
              <div class="text-xs font-mono text-zinc-300 bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5">
                {formatTokens(lim.totalTokenPool)}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                CUSTOM (MILLIONS)
              </div>
              {#if draft}
                <input
                  type="number"
                  bind:value={draft.totalTokenPoolM}
                  min="0.1"
                  step="1"
                  class="w-full text-xs font-mono text-zinc-100 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5 focus:outline-none focus:border-purple-500"
                />
              {/if}
            </div>

            <!-- Column 2: Consumed Tokens -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <TrendingUp class="w-3.5 h-3.5 text-amber-400" />
                <span>Consumption</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                CONSUMED (TOKENS)
              </div>
              <div class="text-xs font-mono text-zinc-300 bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5">
                {formatTokens(lim.consumedTokens)}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                USAGE PERCENTAGE
              </div>
              <div class="text-xs font-mono font-bold {percent >= 90 ? 'text-red-400' : percent >= 75 ? 'text-amber-400' : 'text-emerald-400'} bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5">
                {percent}% of pool
              </div>
            </div>

            <!-- Column 3: Remaining Balance -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <Coins class="w-3.5 h-3.5 text-emerald-400" />
                <span>Remaining</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                AVAILABLE (BAL)
              </div>
              <div class="text-xs font-mono text-emerald-400 font-bold bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5">
                {formatTokens(Math.max(0, lim.totalTokenPool - lim.consumedTokens))}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                HEALTH STATUS
              </div>
              <div class="text-xs font-mono text-zinc-300 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5">
                {#if isExhausted}
                  <span class="text-red-400 font-bold">Depleted</span>
                {:else if percent >= 90}
                  <span class="text-amber-400 font-bold">Critical</span>
                {:else}
                  <span class="text-emerald-400 font-bold">Healthy</span>
                {/if}
              </div>
            </div>

            <!-- Column 4: RPM Speed Limit -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <Zap class="w-3.5 h-3.5 text-cyan-400" />
                <span>Speed Limits</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                RPM (CONFIGURED)
              </div>
              <div class="text-xs font-mono text-zinc-300 bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5">
                {lim.rpmLimit > 0 ? `${lim.rpmLimit} RPM` : 'Unlimited'}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                CUSTOM RPM
              </div>
              {#if draft}
                <input
                  type="number"
                  bind:value={draft.rpmLimit}
                  min="0"
                  placeholder="0 = Unlimited"
                  class="w-full text-xs font-mono text-zinc-100 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5 focus:outline-none focus:border-purple-500"
                />
              {/if}
            </div>

            <!-- Column 5: Reset Window -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <Clock class="w-3.5 h-3.5 text-blue-400" />
                <span>Reset Cycle</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                ORIGINAL CYCLE
              </div>
              <div class="text-xs font-mono text-zinc-300 capitalize bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5">
                {lim.resetWindow}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                SCHEDULE INTERVAL
              </div>
              {#if draft}
                <select
                  bind:value={draft.resetWindow}
                  class="w-full text-xs font-mono text-zinc-100 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5 focus:outline-none focus:border-purple-500"
                >
                  <option value="lifetime">Lifetime</option>
                  <option value="monthly">Monthly</option>
                  <option value="weekly">Weekly</option>
                  <option value="daily">Daily</option>
                </select>
              {/if}
            </div>

            <!-- Column 6: Action on Exhaustion -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <ShieldAlert class="w-3.5 h-3.5 text-red-400" />
                <span>Exhaustion Action</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                ORIGINAL ACTION
              </div>
              <div class="text-xs font-mono text-zinc-300 capitalize bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5">
                {lim.onExhaustion}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                CONFIGURED ACTION
              </div>
              {#if draft}
                <select
                  bind:value={draft.onExhaustion}
                  class="w-full text-xs font-mono text-zinc-100 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5 focus:outline-none focus:border-purple-500"
                >
                  <option value="error">Return Error (429)</option>
                  <option value="fallback">Silent Fallback</option>
                </select>
              {/if}
            </div>
          </div>

          <!-- If Fallback selected, show Fallback Target input -->
          {#if draft && draft.onExhaustion === 'fallback'}
            <div class="mt-3 pt-2.5 border-t border-zinc-800/50 flex flex-col sm:flex-row items-start sm:items-center gap-2 text-xs font-mono">
              <span class="text-purple-400 font-semibold whitespace-nowrap">Fallback Target:</span>
              <input
                type="text"
                bind:value={draft.fallbackTarget}
                placeholder="e.g. gpt-4o-mini or combo ID"
                class="flex-1 w-full text-xs font-mono text-zinc-100 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1 focus:outline-none focus:border-purple-500"
              />
            </div>
          {/if}

          <!-- Visual High-Tech Consumption Progress Bar -->
          <div class="mt-3 pt-2 border-t border-zinc-800/40">
            <div class="flex items-center justify-between text-[11px] font-mono mb-1">
              <span class="text-zinc-500">
                Consumed: <strong class="text-zinc-300">{formatTokens(lim.consumedTokens)}</strong> / {formatTokens(lim.totalTokenPool)}
              </span>
              <span class="font-bold {percent >= 90 ? 'text-red-400' : percent >= 75 ? 'text-amber-400' : 'text-emerald-400'}">
                {percent}% used
              </span>
            </div>
            <div class="w-full bg-zinc-900 h-1.5 rounded-full overflow-hidden border border-zinc-800/60">
              <div class="{barColor} h-full transition-all duration-300" style="width: {percent}%"></div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Create / Edit Policy Modal -->
<Modal bind:open={isModalOpen} title="New Token Limit Policy">
  <div class="space-y-4">
    <div>
      <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Target Model or Combo</label>
      <select
        bind:value={modalTarget}
        class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
      >
        {#each targets as t}
          <option value="{t.type}::{t.id}">[{t.type.toUpperCase()}] {t.name}</option>
        {/each}
      </select>
    </div>

    <div>
      <div class="flex items-center justify-between mb-1 font-mono">
        <label class="block text-xs font-semibold text-zinc-400">Total Token Pool (Millions)</label>
        <span class="text-xs text-purple-400 font-bold">{(modalTokenPoolM * 1_000_000).toLocaleString()} tokens</span>
      </div>
      <input
        type="number"
        bind:value={modalTokenPoolM}
        min="1"
        class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
      />
      <!-- Quick Presets -->
      <div class="flex gap-1.5 mt-2 flex-wrap">
        {#each [5, 10, 25, 50, 100, 250, 500, 1000] as m}
          <button
            type="button"
            onclick={() => (modalTokenPoolM = m)}
            class="px-2 py-0.5 text-xs font-mono rounded border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:border-purple-500 hover:text-white transition-colors"
          >
            {m}M
          </button>
        {/each}
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Reset Cycle</label>
        <select
          bind:value={modalResetWindow}
          class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
        >
          <option value="lifetime">Lifetime (No auto-reset)</option>
          <option value="monthly">Monthly</option>
          <option value="weekly">Weekly</option>
          <option value="daily">Daily</option>
        </select>
      </div>

      <div>
        <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">RPM Limit (Optional)</label>
        <input
          type="number"
          bind:value={modalRPMLimit}
          placeholder="0 = Unlimited"
          class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
        />
      </div>
    </div>

    <div>
      <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">On Exhaustion Behavior</label>
      <select
        bind:value={modalOnExhaustion}
        class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
      >
        <option value="error">Return Custom 429 Error Message</option>
        <option value="fallback">Silent Fallback to Another Model/Combo</option>
      </select>
    </div>

    {#if modalOnExhaustion === 'fallback'}
      <div>
        <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Fallback Target</label>
        <input
          type="text"
          bind:value={modalFallbackTarget}
          placeholder="e.g. gpt-4o-mini or combo id"
          class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
        />
      </div>
    {:else}
      <div>
        <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Custom Error Message</label>
        <input
          type="text"
          bind:value={modalCustomError}
          placeholder="Model token pool exhausted."
          class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
        />
      </div>
    {/if}

    <div class="flex justify-end gap-2 pt-4 border-t border-zinc-800/80">
      <button
        type="button"
        onclick={() => (isModalOpen = false)}
        class="px-3 py-1.5 rounded-lg text-xs font-mono border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 transition-colors"
      >
        Cancel
      </button>
      <button
        type="button"
        onclick={handleSaveLimit}
        disabled={isSaving}
        class="px-4 py-1.5 rounded-lg text-xs font-mono font-medium border border-purple-700 bg-purple-600 text-white hover:bg-purple-500 transition-colors disabled:opacity-50"
      >
        {isSaving ? 'Saving...' : 'Save Limit Policy'}
      </button>
    </div>
  </div>
</Modal>

<!-- Quick Refill Modal -->
<Modal bind:open={isRefillOpen} title="Refill Token Pool">
  {#if refillTargetLimit}
    <div class="space-y-4">
      <div class="text-xs font-mono text-zinc-300">
        Adding tokens to <strong class="text-purple-300">{refillTargetLimit.targetName}</strong>
      </div>

      <div>
        <div class="flex items-center justify-between mb-1 font-mono">
          <label class="block text-xs font-semibold text-zinc-400">Tokens to Add (Millions)</label>
          <span class="text-xs text-amber-400 font-bold">+{(refillAmountM * 1_000_000).toLocaleString()} tokens</span>
        </div>
        <input
          type="number"
          bind:value={refillAmountM}
          min="1"
          class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-amber-500"
        />

        <div class="flex gap-1.5 mt-2 flex-wrap">
          {#each [5, 10, 25, 50, 100, 250, 500] as m}
            <button
              type="button"
              onclick={() => (refillAmountM = m)}
              class="px-2 py-0.5 text-xs font-mono rounded border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:border-amber-500 hover:text-white transition-colors"
            >
              +{m}M
            </button>
          {/each}
        </div>
      </div>

      <div class="flex justify-end gap-2 pt-4 border-t border-zinc-800/80">
        <button
          type="button"
          onclick={() => (isRefillOpen = false)}
          class="px-3 py-1.5 rounded-lg text-xs font-mono border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 transition-colors"
        >
          Cancel
        </button>
        <button
          type="button"
          onclick={submitRefill}
          disabled={isRefilling}
          class="px-4 py-1.5 rounded-lg text-xs font-mono font-medium border border-amber-700 bg-amber-600 text-white hover:bg-amber-500 transition-colors disabled:opacity-50"
        >
          {isRefilling ? 'Refilling...' : 'Confirm Refill'}
        </button>
      </div>
    </div>
  {/if}
</Modal>
