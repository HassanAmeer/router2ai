<script lang="ts">
  import { onMount } from 'svelte'
  import {
    Globe,
    Search,
    Plus,
    RotateCw,
    AlertCircle,
    Check,
    Coins,
    UserCheck,
    UserX,
    Key,
    Shield,
    Trash2,
    Edit2,
    X,
    ChevronDown,
    ChevronLeft,
    ChevronRight,
    ChevronsLeft,
    ChevronsRight,
    ShieldCheck,
    ShieldAlert,
    Save,
    TrendingUp,
    Zap,
    Copy,
    ExternalLink,
    Code,
    Sparkles,
  } from 'lucide-svelte'
  import {
    api,
    type ResellerUser,
    type UsersSummaryStats,
  } from '../api/client'
  import Modal from '../lib/ui/Modal.svelte'

  let users = $state<ResellerUser[]>([])
  let stats = $state<UsersSummaryStats>({
    totalUsers: 0,
    activeUsers: 0,
    bannedUsers: 0,
    totalBudget: 0,
    totalTokensUsed: 0,
  })
  let loading = $state(true)
  let error = $state('')

  let searchQuery = $state('')
  let selectedPlan = $state<string>('all')
  let statusFilter = $state<'all' | 'active' | 'banned'>('all')

  // Pagination
  let page = $state(1)
  const pageSize = 20

  // Per-item inline edit state
  let edits = $state<
    Record<
      string,
      {
        tokenBudgetM: number
        rpmLimit: number
        planTier: string
        subdomain: string
        allowedModels: string
      }
    >
  >({})
  let savingId = $state<string | null>(null)
  let savedId = $state<string | null>(null)
  let copiedId = $state<string | null>(null)
  let copiedEnvId = $state<string | null>(null)

  // Horizontal scroll container for plan tabs
  let tabsContainer = $state<HTMLElement | null>(null)

  function scrollTabs(direction: 'left' | 'right') {
    if (!tabsContainer) return
    const offset = direction === 'left' ? -250 : 250
    tabsContainer.scrollBy({ left: offset, behavior: 'smooth' })
  }

  // Create / Edit Website Modal
  let isModalOpen = $state(false)
  let isEditing = $state(false)
  let editingId = $state('')
  let modalName = $state('')
  let modalDomain = $state('')
  let modalSubdomain = $state('')
  let modalAllowedModels = $state('*')
  let modalPlan = $state('pro')
  let modalBudgetM = $state(50) // 50M default
  let modalRPMLimit = $state(60)
  let modalIsBanned = $state(false)
  let isSaving = $state(false)

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

  function syncEdits(items: ResellerUser[]) {
    for (const u of items) {
      if (!edits[u.id]) {
        edits[u.id] = {
          tokenBudgetM: Number((u.tokenBudget / 1_000_000).toFixed(2)),
          rpmLimit: u.rpmLimit || 60,
          planTier: u.planTier || 'pro',
          subdomain: u.subdomain || 'http://localhost:20130/v1',
          allowedModels: u.allowedModels || '*',
        }
      }
    }
  }

  async function loadData() {
    loading = true
    error = ''
    try {
      const res = await api.getResellerUsers()
      users = res.users || []
      stats = res.stats || {
        totalUsers: 0,
        activeUsers: 0,
        bannedUsers: 0,
        totalBudget: 0,
        totalTokensUsed: 0,
      }
      syncEdits(users)
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load website scopes'
    } finally {
      loading = false
    }
  }

  onMount(() => {
    loadData()
  })

  // Derived available plan tiers
  let availablePlans = $derived.by(() => {
    const set = new Set<string>(['standard', 'pro', 'vip', 'enterprise', 'free'])
    for (const u of users) {
      if (u.planTier) set.add(u.planTier.toLowerCase())
    }
    return Array.from(set).sort()
  })

  // Filtered websites
  let filteredUsers = $derived.by(() => {
    let list = users
    if (selectedPlan !== 'all') {
      list = list.filter((u) => u.planTier.toLowerCase() === selectedPlan.toLowerCase())
    }
    if (statusFilter === 'active') {
      list = list.filter((u) => !u.isBanned)
    } else if (statusFilter === 'banned') {
      list = list.filter((u) => u.isBanned)
    }

    const q = searchQuery.trim().toLowerCase()
    if (q) {
      list = list.filter(
        (u) =>
          u.name.toLowerCase().includes(q) ||
          u.email.toLowerCase().includes(q) ||
          (u.subdomain && u.subdomain.toLowerCase().includes(q)) ||
          (u.apiKey && u.apiKey.toLowerCase().includes(q)) ||
          u.id.toLowerCase().includes(q)
      )
    }
    return list
  })

  // Pagination
  let totalPages = $derived(Math.max(1, Math.ceil(filteredUsers.length / pageSize)))
  let paginatedUsers = $derived.by(() => {
    const start = (page - 1) * pageSize
    return filteredUsers.slice(start, start + pageSize)
  })

  $effect(() => {
    // Reset page whenever filter changes
    selectedPlan
    statusFilter
    searchQuery
    page = 1
  })

  function openCreateModal() {
    isEditing = false
    editingId = ''
    modalName = ''
    modalDomain = ''
    modalSubdomain = 'https://ai.mysite.com/v1'
    modalAllowedModels = '*'
    modalPlan = 'pro'
    modalBudgetM = 50
    modalRPMLimit = 60
    modalIsBanned = false
    isModalOpen = true
  }

  function openEditModal(u: ResellerUser) {
    isEditing = true
    editingId = u.id
    modalName = u.name
    modalDomain = u.email
    modalSubdomain = u.subdomain || 'http://localhost:20130/v1'
    modalAllowedModels = u.allowedModels || '*'
    modalPlan = u.planTier || 'pro'
    modalBudgetM = u.tokenBudget ? Math.round(u.tokenBudget / 1_000_000) : 50
    modalRPMLimit = u.rpmLimit || 60
    modalIsBanned = u.isBanned
    isModalOpen = true
  }

  async function handleSaveUser() {
    if (!modalName.trim() || !modalDomain.trim()) {
      alert('Website Name and Domain URL are required')
      return
    }
    isSaving = true
    try {
      const budgetBytes = Math.floor(modalBudgetM * 1_000_000)
      if (isEditing) {
        await api.updateResellerUser({
          id: editingId,
          email: modalDomain.trim(),
          name: modalName.trim(),
          subdomain: modalSubdomain.trim(),
          allowedModels: modalAllowedModels.trim(),
          planTier: modalPlan,
          tokenBudget: budgetBytes,
          rpmLimit: Number(modalRPMLimit) || 0,
          isBanned: modalIsBanned,
        })
      } else {
        await api.createResellerUser({
          email: modalDomain.trim(),
          name: modalName.trim(),
          subdomain: modalSubdomain.trim(),
          allowedModels: modalAllowedModels.trim(),
          planTier: modalPlan,
          tokenBudget: budgetBytes,
          rpmLimit: Number(modalRPMLimit) || 0,
          isBanned: modalIsBanned,
        })
      }
      isModalOpen = false
      await loadData()
    } catch (err) {
      alert('Save failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      isSaving = false
    }
  }

  async function saveSingleItem(u: ResellerUser) {
    const draft = edits[u.id]
    if (!draft) return
    savingId = u.id
    try {
      const newBudget = Math.floor(draft.tokenBudgetM * 1_000_000)
      await api.updateResellerUser({
        id: u.id,
        email: u.email,
        name: u.name,
        subdomain: draft.subdomain,
        allowedModels: draft.allowedModels,
        planTier: draft.planTier,
        tokenBudget: newBudget,
        rpmLimit: Number(draft.rpmLimit) || 0,
        isBanned: u.isBanned,
      })

      u.tokenBudget = newBudget
      u.rpmLimit = Number(draft.rpmLimit) || 0
      u.planTier = draft.planTier
      u.subdomain = draft.subdomain
      u.allowedModels = draft.allowedModels

      savedId = u.id
      setTimeout(() => {
        if (savedId === u.id) savedId = null
      }, 2000)
    } catch (err) {
      alert('Save failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      savingId = null
    }
  }

  async function toggleBan(u: ResellerUser) {
    const next = !u.isBanned
    u.isBanned = next
    try {
      await api.updateResellerUser({
        id: u.id,
        email: u.email,
        name: u.name,
        subdomain: u.subdomain,
        allowedModels: u.allowedModels,
        planTier: u.planTier,
        tokenBudget: u.tokenBudget,
        rpmLimit: u.rpmLimit,
        isBanned: next,
      })
    } catch {
      u.isBanned = !next
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('Are you sure you want to delete this website scope? Its scoped API key will be permanently removed.')) return
    try {
      await api.deleteResellerUser(id)
      users = users.filter((u) => u.id !== id)
      delete edits[id]
    } catch (err) {
      alert('Delete failed: ' + (err instanceof Error ? err.message : String(err)))
    }
  }

  function writeClipboard(text: string) {
    try {
      if (navigator?.clipboard?.writeText) {
        navigator.clipboard.writeText(text).catch(() => fallbackCopy(text))
      } else {
        fallbackCopy(text)
      }
    } catch {
      fallbackCopy(text)
    }
  }

  function fallbackCopy(text: string) {
    try {
      const el = document.createElement('textarea')
      el.value = text
      el.setAttribute('readonly', '')
      el.style.position = 'absolute'
      el.style.left = '-9999px'
      document.body.appendChild(el)
      el.select()
      document.execCommand('copy')
      document.body.removeChild(el)
    } catch {}
  }

  function copyApiKey(key: string, id: string) {
    writeClipboard(key)
    copiedId = id
    setTimeout(() => {
      if (copiedId === id) copiedId = null
    }, 2000)
  }

  function copyEnvSnippet(u: ResellerUser) {
    const endpoint = u.subdomain || 'http://localhost:20130/v1'
    const snippet = `OPENAI_BASE_URL=${endpoint}\nOPENAI_API_KEY=${u.apiKey || 'sk-site-xxxx'}`
    writeClipboard(snippet)
    copiedEnvId = u.id
    setTimeout(() => {
      if (copiedEnvId === u.id) copiedEnvId = null
    }, 2500)
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
          title="Websites Hub"
        >
          <ChevronDown class="w-4 h-4" />
        </button>
        <div class="p-1.5 rounded-lg bg-purple-950/80 border border-purple-800/60 text-purple-400">
          <Globe class="w-5 h-5" />
        </div>
        <div class="flex items-center gap-2 flex-wrap">
          <h1 class="text-base font-bold text-zinc-100 font-mono tracking-tight">
            Websites & Scoped API Endpoints
          </h1>
          <span class="px-2 py-0.5 rounded-full text-xs font-mono font-semibold bg-purple-950/90 text-purple-300 border border-purple-800/70">
            {stats.activeUsers}/{stats.totalUsers} active
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
          class="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-mono font-medium border border-purple-700/80 bg-purple-900/40 text-purple-200 hover:bg-purple-800/60 hover:text-white transition-all cursor-pointer shadow-xs"
        >
          <Plus class="w-3.5 h-3.5 text-purple-300" />
          <span>Add Website</span>
        </button>
      </div>
    </div>

    <!-- Plan Tiers & Category Filter Tabs Bar (Horizontal Scrollable) -->
    <div class="mt-3 pt-3 border-t border-zinc-800/60">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-1.5">
          <span class="w-2 h-2 rounded-full bg-purple-500 animate-pulse"></span>
          <span class="text-[11px] font-bold text-zinc-300 font-mono uppercase tracking-wider">
            Website Scopes & Plans ({availablePlans.length}):
          </span>
          <span class="text-[10px] text-zinc-500 font-mono hidden sm:inline">
            — click a tab to view specific website scopes
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

      <!-- Horizontal Scrollable Tabs Container -->
      <div
        bind:this={tabsContainer}
        class="flex items-center gap-2 overflow-x-auto pb-1.5 scrollbar-thin scrollbar-thumb-zinc-800 scrollbar-track-transparent select-none"
        style="scroll-behavior: smooth;"
      >
        <!-- All Websites Tab -->
        <button
          type="button"
          onclick={() => (selectedPlan = 'all')}
          class="flex-shrink-0 inline-flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {selectedPlan === 'all'
            ? 'bg-purple-600 text-white shadow-xs ring-1 ring-purple-500 font-bold'
            : 'border border-zinc-800 bg-zinc-900/80 text-zinc-400 hover:text-zinc-200 hover:border-zinc-700 hover:bg-zinc-850'}"
        >
          <span>All Websites</span>
          <span class="px-1.5 py-0.2 rounded-full text-[10px] font-bold {selectedPlan === 'all' ? 'bg-purple-700 text-white' : 'bg-zinc-800 text-zinc-400'}">
            {users.length}
          </span>
        </button>

        <!-- Individual Plan Tier Tabs -->
        {#each availablePlans as plan}
          {@const planUsers = users.filter((u) => u.planTier.toLowerCase() === plan.toLowerCase())}
          {@const planCount = planUsers.length}
          {@const planActive = planUsers.filter((u) => !u.isBanned).length}
          <button
            type="button"
            onclick={() => (selectedPlan = plan)}
            class="flex-shrink-0 inline-flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs font-mono font-medium transition-all cursor-pointer {selectedPlan === plan
              ? 'bg-purple-600 text-white shadow-xs ring-1 ring-purple-500 font-bold'
              : 'border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:text-white hover:border-zinc-700 hover:bg-zinc-850'}"
            title="Filter by plan {plan.toUpperCase()}"
          >
            <span class="w-2 h-2 rounded-full {planActive > 0 ? 'bg-emerald-400' : 'bg-zinc-500'}"></span>
            <span class="font-semibold uppercase tracking-wide">{plan}</span>
            <span class="px-1.5 py-0.2 rounded-full text-[10px] font-bold {selectedPlan === plan ? 'bg-purple-700 text-white' : 'bg-zinc-800 text-zinc-400'}">
              {planCount}
            </span>
          </button>
        {/each}
      </div>
    </div>
  </div>

  <!-- Cyberpunk Summary Metric Cards -->
  <div class="w-full grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
    <!-- Total Websites -->
    <div class="rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm flex items-center gap-3.5">
      <div class="w-11 h-11 rounded-xl bg-blue-500/10 border border-blue-500/30 text-blue-400 flex items-center justify-center flex-shrink-0">
        <Globe class="w-5 h-5" />
      </div>
      <div>
        <div class="text-[11px] font-mono font-medium text-zinc-400 uppercase tracking-wide">Connected Websites</div>
        <div class="text-xl font-bold font-mono text-zinc-100 mt-0.5">
          {stats.totalUsers}
        </div>
      </div>
    </div>

    <!-- Active Endpoints -->
    <div class="rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm flex items-center gap-3.5">
      <div class="w-11 h-11 rounded-xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 flex items-center justify-center flex-shrink-0">
        <UserCheck class="w-5 h-5" />
      </div>
      <div>
        <div class="text-[11px] font-mono font-medium text-zinc-400 uppercase tracking-wide">Active Scopes</div>
        <div class="text-xl font-bold font-mono text-emerald-400 mt-0.5">
          {stats.activeUsers}
        </div>
      </div>
    </div>

    <!-- Total Token Budget -->
    <div class="rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm flex items-center gap-3.5">
      <div class="w-11 h-11 rounded-xl bg-purple-500/10 border border-purple-500/30 text-purple-400 flex items-center justify-center flex-shrink-0">
        <Coins class="w-5 h-5" />
      </div>
      <div>
        <div class="text-[11px] font-mono font-medium text-zinc-400 uppercase tracking-wide">Total Token Budget</div>
        <div class="text-xl font-bold font-mono text-purple-300 mt-0.5">
          {formatTokens(stats.totalBudget)}
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
        <div class="text-xl font-bold font-mono text-amber-400 mt-0.5">
          {formatTokens(stats.totalTokensUsed)}
        </div>
      </div>
    </div>
  </div>

  <!-- Search & Filter Controls -->
  <div class="w-full rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-3 shadow-sm">
    <div class="flex flex-col sm:flex-row items-center gap-3">
      <!-- Search Input -->
      <div class="relative flex-1 w-full">
        <Search class="w-4 h-4 text-zinc-500 absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none" />
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Search by website name, domain URL, subdomain link, or key..."
          class="w-full pl-9 pr-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900/90 text-zinc-200 placeholder-zinc-500 text-xs font-mono focus:outline-none focus:border-purple-500 transition-colors"
        />
      </div>

      <!-- Plan Select -->
      <div class="w-full sm:w-auto">
        <select
          bind:value={selectedPlan}
          class="w-full sm:w-auto px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900/90 text-zinc-200 text-xs font-mono focus:outline-none focus:border-purple-500"
        >
          <option value="all">All Plans ({users.length})</option>
          {#each availablePlans as plan}
            <option value={plan}>{plan.toUpperCase()}</option>
          {/each}
        </select>
      </div>

      <!-- Status Select -->
      <div class="w-full sm:w-auto">
        <select
          bind:value={statusFilter}
          class="w-full sm:w-auto px-3 py-1.5 rounded-lg border border-zinc-800 bg-zinc-900/90 text-zinc-200 text-xs font-mono focus:outline-none focus:border-purple-500"
        >
          <option value="all">All Status</option>
          <option value="active">Active Only</option>
          <option value="banned">Paused / Suspended</option>
        </select>
      </div>
    </div>
  </div>

  <!-- Subtitle Line: Showing X-Y of Total + CheapRouter Pagination -->
  <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 px-1 py-1 text-xs text-zinc-400 font-mono">
    <div class="flex items-center gap-2 flex-wrap">
      <span class="w-1.5 h-1.5 rounded-full bg-purple-500 animate-pulse"></span>
      <span>
        Showing <strong class="text-zinc-200 font-semibold">{filteredUsers.length === 0 ? 0 : (page - 1) * pageSize + 1}–{Math.min(page * pageSize, filteredUsers.length)}</strong> of <strong class="text-zinc-200 font-semibold">{filteredUsers.length}</strong> websites ({pageSize} / page)
      </span>
      {#if selectedPlan !== 'all'}
        <span class="px-2 py-0.5 rounded text-[10px] font-bold bg-purple-950/80 border border-purple-800/60 text-purple-300 uppercase">
          {selectedPlan}
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
        class="px-2.5 py-1 rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 text-xs flex items-center gap-1 disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
      >
        <ChevronLeft class="w-3.5 h-3.5" />
        <span>Prev</span>
      </button>
      <span class="px-3 py-1 rounded border border-purple-500/40 bg-purple-950/40 text-purple-300 font-bold text-xs">
        {page}
      </span>
      <button
        type="button"
        onclick={() => (page = Math.min(totalPages, page + 1))}
        disabled={page >= totalPages}
        class="px-2.5 py-1 rounded border border-zinc-800 bg-zinc-900 hover:bg-zinc-800 text-zinc-400 hover:text-zinc-200 text-xs flex items-center gap-1 disabled:opacity-30 disabled:cursor-not-allowed cursor-pointer"
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

  <!-- Websites List: CheapRouter 6-Column Card Architecture -->
  {#if loading && users.length === 0}
    <div class="p-12 text-center text-zinc-400 font-mono text-sm">
      <RotateCw class="w-6 h-6 animate-spin mx-auto mb-2 text-purple-400" />
      Loading Website Scopes...
    </div>
  {:else if error}
    <div class="p-4 rounded-xl bg-red-950/40 border border-red-900/60 text-red-300 font-mono text-xs flex items-center gap-3">
      <AlertCircle class="w-5 h-5 flex-shrink-0 text-red-400" />
      <div>{error}</div>
    </div>
  {:else if filteredUsers.length === 0}
    <div class="w-full rounded-xl border border-zinc-800/80 bg-zinc-950/60 p-12 text-center">
      <Globe class="w-12 h-12 text-zinc-600 mx-auto mb-3 opacity-60" />
      <h3 class="text-sm font-semibold font-mono text-zinc-200">No Websites Added Yet</h3>
      <p class="text-xs text-zinc-400 font-mono mt-1 mb-4">
        Connect your first website to assign an API subdomain, scoped models, and token budget.
      </p>
      <button
        type="button"
        onclick={openCreateModal}
        class="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg text-xs font-mono font-medium border border-purple-700 bg-purple-900/40 text-purple-200 hover:bg-purple-800/60 hover:text-white transition-all cursor-pointer shadow-xs"
      >
        <Plus class="w-4 h-4 text-purple-300" />
        <span>Add First Website</span>
      </button>
    </div>
  {:else}
    <div class="w-full space-y-3">
      {#each paginatedUsers as u (u.id)}
        {@const draft = edits[u.id]}
        {@const percent = u.tokenBudget > 0 ? Math.min(100, Math.round((u.tokensUsed / u.tokenBudget) * 100)) : 0}
        {@const isExhausted = u.tokenBudget > 0 && u.tokensUsed >= u.tokenBudget}
        {@const barColor = percent >= 90 ? 'bg-red-500' : percent >= 75 ? 'bg-amber-500' : 'bg-emerald-500'}

        <div
          class="w-full rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-4 shadow-sm hover:border-zinc-700/80 transition-all {!u.isBanned
            ? ''
            : 'opacity-70 bg-red-950/10 border-red-900/30'}"
        >
          <!-- Card Header: Purple Toggle Switch, Title, Subdomain Badge, Plan Tag, Actions -->
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-3 border-b border-zinc-800/60">
            <!-- Left Side: Switch + Website Name + Subdomain Badge + Plan Tag -->
            <div class="flex items-center gap-3 flex-wrap">
              <!-- Purple Switch Toggle (Active vs Paused) -->
              <button
                type="button"
                role="switch"
                aria-checked={!u.isBanned}
                onclick={() => toggleBan(u)}
                class="relative inline-flex h-5 w-9 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none {!u.isBanned
                  ? 'bg-purple-600'
                  : 'bg-zinc-700'}"
                title={!u.isBanned ? 'Website API is Active' : 'Website API is Paused'}
              >
                <span
                  class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow-sm ring-0 transition duration-200 ease-in-out {!u.isBanned
                    ? 'translate-x-4'
                    : 'translate-x-0'}"
                ></span>
              </button>

              <!-- Website Name Title -->
              <span class="text-sm font-bold text-zinc-100 font-mono tracking-tight flex items-center gap-1.5">
                <Globe class="w-4 h-4 text-purple-400" />
                {u.name || 'Untitled Website'}
              </span>

              <!-- Subdomain / API Endpoint Link Badge -->
              <span class="px-2.5 py-0.5 rounded text-[11px] font-mono font-bold text-cyan-300 border border-cyan-800/60 bg-cyan-950/40">
                {u.subdomain || 'http://localhost:20130/v1'}
              </span>

              <!-- Plan Badge -->
              <span class="px-2 py-0.5 rounded text-[11px] font-mono font-bold uppercase border {u.planTier === 'vip' || u.planTier === 'enterprise'
                ? 'border-purple-800/60 bg-purple-950/60 text-purple-300'
                : u.planTier === 'business'
                  ? 'border-amber-800/60 bg-amber-950/60 text-amber-300'
                  : u.planTier === 'pro'
                    ? 'border-blue-800/60 bg-blue-950/60 text-blue-300'
                    : 'border-zinc-700/60 bg-zinc-900/80 text-zinc-400'}">
                {u.planTier}
              </span>

              <!-- Status Badge -->
              {#if u.isBanned}
                <span class="px-2 py-0.5 rounded text-[11px] font-mono font-semibold text-red-400 border border-red-800/60 bg-red-950/60 flex items-center gap-1">
                  <UserX class="w-3 h-3" /> PAUSED
                </span>
              {:else if isExhausted}
                <span class="px-2 py-0.5 rounded text-[11px] font-mono font-semibold text-amber-400 border border-amber-800/60 bg-amber-950/60">
                  DEPLETED
                </span>
              {:else}
                <span class="px-2 py-0.5 rounded text-[11px] font-mono font-semibold text-emerald-400 border border-emerald-800/60 bg-emerald-950/60 flex items-center gap-1">
                  <Check class="w-3 h-3" /> ACTIVE
                </span>
              {/if}
            </div>

            <!-- Right Side: Copy .env Button, Edit Button, Save Button, Delete Button -->
            <div class="flex items-center gap-2 flex-shrink-0 self-end sm:self-center">
              <!-- 1-Click Copy .env Configuration Button -->
              <button
                type="button"
                onclick={() => copyEnvSnippet(u)}
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-mono font-semibold border transition-all cursor-pointer {copiedEnvId === u.id
                  ? 'border-emerald-500 bg-emerald-950/70 text-emerald-300'
                  : 'border-purple-700/80 bg-purple-950/40 text-purple-200 hover:bg-purple-900/60 hover:text-white'}"
                title="Copy ready-to-use OPENAI_BASE_URL and API Key snippet"
              >
                {#if copiedEnvId === u.id}
                  <Check class="w-3.5 h-3.5 text-emerald-400" />
                  <span>Copied .env!</span>
                {:else}
                  <Code class="w-3.5 h-3.5 text-purple-300" />
                  <span>Copy .env</span>
                {/if}
              </button>

              <!-- Edit Modal Button -->
              <button
                type="button"
                onclick={() => openEditModal(u)}
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-mono font-medium border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:bg-zinc-800 hover:text-white transition-all cursor-pointer"
              >
                <Edit2 class="w-3.5 h-3.5 text-zinc-400" />
                <span>Edit</span>
              </button>

              <!-- Green Outline Save Button -->
              <button
                type="button"
                onclick={() => saveSingleItem(u)}
                disabled={savingId === u.id}
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-mono font-semibold border transition-all cursor-pointer {savedId ===
                u.id
                  ? 'border-emerald-500 bg-emerald-950/60 text-emerald-300'
                  : 'border-emerald-600/80 bg-emerald-950/20 text-emerald-400 hover:bg-emerald-900/40 hover:text-emerald-300'}"
              >
                {#if savingId === u.id}
                  <RotateCw class="w-3.5 h-3.5 animate-spin" />
                  <span>Saving...</span>
                {:else if savedId === u.id}
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
                onclick={() => handleDelete(u.id)}
                class="p-1.5 text-zinc-500 hover:text-red-400 hover:bg-red-950/30 rounded border border-transparent hover:border-red-900/40 transition-colors cursor-pointer"
                title="Delete website"
              >
                <Trash2 class="w-4 h-4" />
              </button>
            </div>
          </div>

          <!-- Metadata Subtitle Row: DOMAIN, SUBDOMAIN LINK, KEY, MODELS -->
          <div class="flex flex-wrap items-center justify-between text-[11px] font-mono text-zinc-400 mt-2.5 pt-2 border-t border-zinc-800/50 gap-2">
            <div>
              <span class="text-zinc-500">MAIN DOMAIN:</span> <span class="text-zinc-300 font-bold">{u.email}</span>
            </div>
            <div>
              <span class="text-zinc-500">API SUBDOMAIN:</span> <span class="text-cyan-400 font-mono">{u.subdomain || 'http://localhost:20130/v1'}</span>
            </div>
            <div>
              <span class="text-zinc-500">SCOPED KEY:</span> <span class="text-purple-400 font-mono">{u.apiKey ? u.apiKey.slice(0, 16) + '...' : 'Auto-Issued'}</span>
            </div>
            <div>
              <span class="text-zinc-500">ALLOWED MODELS:</span> <span class="text-emerald-400 font-bold">{u.allowedModels || 'All Models (*)'}</span>
            </div>
          </div>

          <!-- 6-Column Grid Layout (Exact CheapRouter Architecture) -->
          <div class="w-full grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-3 mt-3 pt-3 border-t border-zinc-800/50">
            <!-- Column 1: API Subdomain Link -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <Globe class="w-3.5 h-3.5 text-cyan-400" />
                <span>API Endpoint Link</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                ACTIVE SUBDOMAIN
              </div>
              <div class="text-xs font-mono text-cyan-300 bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5 truncate" title={u.subdomain}>
                {u.subdomain || 'http://localhost:20130/v1'}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                CUSTOM ENDPOINT
              </div>
              {#if draft}
                <input
                  type="text"
                  bind:value={draft.subdomain}
                  placeholder="https://ai.mysite.com/v1"
                  class="w-full text-xs font-mono text-zinc-100 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5 focus:outline-none focus:border-purple-500"
                />
              {/if}
            </div>

            <!-- Column 2: Scoped API Key -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <Key class="w-3.5 h-3.5 text-purple-400" />
                <span>Scoped API Key</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                SECRET KEY
              </div>
              <div class="text-xs font-mono text-purple-300 bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5 flex items-center justify-between">
                <span class="truncate">{u.apiKey || 'sk-site-xxxx'}</span>
                <button
                  type="button"
                  onclick={() => copyApiKey(u.apiKey || '', u.id)}
                  class="p-0.5 text-zinc-400 hover:text-white transition-colors cursor-pointer"
                  title="Copy API key"
                >
                  {#if copiedId === u.id}
                    <Check class="w-3 h-3 text-emerald-400" />
                  {:else}
                    <Copy class="w-3 h-3" />
                  {/if}
                </button>
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                KEY STATUS
              </div>
              <div class="text-xs font-mono text-zinc-300 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5 flex items-center justify-between">
                <span>{!u.isBanned ? 'Active & Valid' : 'Suspended'}</span>
                <span class="w-2 h-2 rounded-full {!u.isBanned ? 'bg-emerald-400' : 'bg-red-400'}"></span>
              </div>
            </div>

            <!-- Column 3: Allocated Token Budget -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <Coins class="w-3.5 h-3.5 text-amber-400" />
                <span>Token Budget</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                ALLOCATED (TOTAL)
              </div>
              <div class="text-xs font-mono text-zinc-300 bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5">
                {formatTokens(u.tokenBudget)}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                CUSTOM (MILLIONS)
              </div>
              {#if draft}
                <input
                  type="number"
                  bind:value={draft.tokenBudgetM}
                  min="1"
                  step="5"
                  class="w-full text-xs font-mono text-zinc-100 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5 focus:outline-none focus:border-purple-500"
                />
              {/if}
            </div>

            <!-- Column 4: Consumed Usage -->
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
                {formatTokens(u.tokensUsed)}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                USAGE PERCENTAGE
              </div>
              <div class="text-xs font-mono font-bold {percent >= 90 ? 'text-red-400' : percent >= 75 ? 'text-amber-400' : 'text-emerald-400'} bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5">
                {percent}% of budget
              </div>
            </div>

            <!-- Column 5: RPM Rate Limits -->
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
                {u.rpmLimit > 0 ? `${u.rpmLimit} RPM` : 'Unlimited'}
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

            <!-- Column 6: Allowed AI Models Scope -->
            <div>
              <div class="flex items-center gap-1.5 text-xs font-medium text-zinc-300 mb-1.5">
                <ShieldCheck class="w-3.5 h-3.5 text-emerald-400" />
                <span>Allowed Models</span>
              </div>
              <div class="text-[10px] font-mono text-amber-400 uppercase font-semibold mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>
                ACTIVE SCOPE
              </div>
              <div class="text-xs font-mono text-emerald-400 bg-zinc-900/60 border border-zinc-800/60 rounded px-2.5 py-1.5 truncate" title={u.allowedModels || 'All Models (*)'}>
                {u.allowedModels || 'All Models (*)'}
              </div>
              <div class="text-[10px] font-mono text-emerald-400 uppercase font-semibold mt-1.5 mb-1 flex items-center gap-1">
                <span class="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
                CUSTOM ALLOWLIST
              </div>
              {#if draft}
                <input
                  type="text"
                  bind:value={draft.allowedModels}
                  placeholder="* or *gemini*,*muse*"
                  class="w-full text-xs font-mono text-zinc-100 bg-zinc-900/90 border border-zinc-800 rounded px-2.5 py-1.5 focus:outline-none focus:border-purple-500"
                />
              {/if}
            </div>
          </div>

          <!-- Visual High-Tech Consumption Progress Bar -->
          <div class="mt-3 pt-2 border-t border-zinc-800/40">
            <div class="flex items-center justify-between text-[11px] font-mono mb-1">
              <span class="text-zinc-500">
                Consumed: <strong class="text-zinc-300">{formatTokens(u.tokensUsed)}</strong> / {formatTokens(u.tokenBudget)}
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

<!-- Add / Edit Website Modal -->
<Modal
  isOpen={isModalOpen}
  onClose={() => (isModalOpen = false)}
  title={isEditing ? 'Edit Website Scope' : 'Connect New Website'}
  size="xl"
>
  <div class="space-y-4">
    <!-- Website Name -->
    <div>
      <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Website Name</label>
      <input
        type="text"
        bind:value={modalName}
        placeholder="e.g. My E-commerce Store"
        class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
      />
    </div>

    <!-- Main Domain URL -->
    <div>
      <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Website Main Domain / URL</label>
      <input
        type="text"
        bind:value={modalDomain}
        placeholder="e.g. store.mybrand.com"
        class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
      />
    </div>

    <!-- API Subdomain / Endpoint URL -->
    <div>
      <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">API Subdomain / Endpoint Link</label>
      <input
        type="text"
        bind:value={modalSubdomain}
        placeholder="e.g. https://ai.mybrand.com/v1"
        class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
      />
      <span class="text-[10px] text-zinc-500 font-mono mt-1 block">
        This is the OPENAI_BASE_URL your website will call to access AI models.
      </span>
    </div>

    <!-- Allowed Models Allowlist -->
    <div>
      <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Allowed AI Models (Scope)</label>
      <input
        type="text"
        bind:value={modalAllowedModels}
        placeholder="* or gemini-4-carbon, ag/gemini-2.5-flash, opencode/*"
        class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
      />
      <span class="text-[10px] text-zinc-500 font-mono mt-1 block">
        Comma-separated allowed patterns. Use * for all models, or restrict to specific models.
      </span>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Plan Tier</label>
        <select
          bind:value={modalPlan}
          class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500 uppercase"
        >
          <option value="standard">STANDARD</option>
          <option value="pro">PRO</option>
          <option value="vip">VIP</option>
          <option value="enterprise">ENTERPRISE</option>
          <option value="free">FREE</option>
        </select>
      </div>

      <div>
        <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Speed Limit (RPM)</label>
        <input
          type="number"
          bind:value={modalRPMLimit}
          class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
        />
      </div>
    </div>

    <div>
      <div class="flex items-center justify-between mb-1 font-mono">
        <label class="block text-xs font-semibold text-zinc-400">Monthly Token Budget (Millions)</label>
        <span class="text-xs text-purple-400 font-bold">{(modalBudgetM * 1_000_000).toLocaleString()} tokens</span>
      </div>
      <input
        type="number"
        bind:value={modalBudgetM}
        min="1"
        class="w-full px-3 py-2 text-xs font-mono rounded-lg border border-zinc-800 bg-zinc-900 text-zinc-100 focus:outline-none focus:border-purple-500"
      />
      <!-- Quick Presets -->
      <div class="flex gap-1.5 mt-2 flex-wrap">
        {#each [10, 25, 50, 100, 250, 500, 1000] as m}
          <button
            type="button"
            onclick={() => (modalBudgetM = m)}
            class="px-2 py-0.5 text-xs font-mono rounded border border-zinc-800 bg-zinc-900/80 text-zinc-300 hover:border-purple-500 hover:text-white transition-colors cursor-pointer"
          >
            {m}M
          </button>
        {/each}
      </div>
    </div>

    {#if isEditing}
      <div class="pt-2 border-t border-zinc-800/80">
        <label class="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            bind:checked={modalIsBanned}
            class="rounded border-zinc-800 bg-zinc-900 text-purple-600 focus:ring-purple-500"
          />
          <span class="text-xs font-mono text-zinc-300">Pause / Suspend this website's API access</span>
        </label>
      </div>
    {/if}

    <div class="flex justify-end gap-2 pt-4 border-t border-zinc-800/80">
      <button
        type="button"
        onclick={() => (isModalOpen = false)}
        class="px-3 py-1.5 rounded-lg text-xs font-mono border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 transition-colors cursor-pointer"
      >
        Cancel
      </button>
      <button
        type="button"
        onclick={handleSaveUser}
        disabled={isSaving}
        class="px-4 py-1.5 rounded-lg text-xs font-mono font-medium border border-purple-700 bg-purple-600 text-white hover:bg-purple-500 transition-colors disabled:opacity-50 cursor-pointer"
      >
        {isSaving ? 'Saving...' : isEditing ? 'Update Scope' : 'Connect Website & Generate Key'}
      </button>
    </div>
  </div>
</Modal>
