<script lang="ts">
  import Modal from '../../lib/ui/Modal.svelte'
  import { api, type ResellerUser } from '../../api/client'
  import { getWebsiteScope } from '../../lib/websiteScope.svelte'
  import { notifications } from '../../lib/notifications'
  import { Globe, Plus, Cpu, DollarSign, Gauge, Shield, Link2, Sparkles } from 'lucide-svelte'

  let {
    isOpen = false,
    onClose,
    onSuccess,
  }: {
    isOpen?: boolean
    onClose?: () => void
    onSuccess?: (created: ResellerUser) => void
  } = $props()

  const scope = getWebsiteScope()

  let modalName = $state('')
  let modalDomain = $state('')
  let modalSubdomain = $state('https://ai.mysite.com/v1')
  let modalAllowedModels = $state('*')
  let modalPlan = $state<'free' | 'standard' | 'pro' | 'vip' | 'enterprise'>('pro')
  let modalBudgetM = $state<number>(50)
  let modalRPMLimit = $state<number>(60)
  let modalIsBanned = $state(false)
  let isSaving = $state(false)

  const budgetPresets = [10, 25, 50, 100, 250, 500, 1000]

  async function handleSave() {
    if (!modalName.trim() || !modalDomain.trim()) {
      alert('Website Name and Main Domain URL are required')
      return
    }
    isSaving = true
    try {
      const budgetBytes = Math.round((Number(modalBudgetM) || 50) * 1_000_000)
      const res = await api.createResellerUser({
        name: modalName.trim(),
        email: modalDomain.trim(),
        subdomain: modalSubdomain.trim(),
        allowedModels: modalAllowedModels.trim(),
        planTier: modalPlan,
        tokenBudget: budgetBytes,
        rpmLimit: Number(modalRPMLimit) || 0,
        isBanned: modalIsBanned,
      })

      notifications.success(`Website "${modalName}" connected with scoped API key!`)
      await scope.loadWebsites()
      if (res && res.user && res.user.id) {
        scope.selectedWebsiteId = res.user.id
      }
      if (onSuccess && res.user) {
        onSuccess(res.user)
      }
      if (onClose) onClose()
    } catch (err) {
      alert('Save failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      isSaving = false
    }
  }
</script>

<Modal
  {isOpen}
  {onClose}
  title="Connect New Website & Scoped API"
  size="xl"
>
  <div class="space-y-4">
    <!-- Header banner -->
    <div class="flex items-center gap-3 p-3 rounded-lg border border-purple-800/60 bg-purple-950/40 text-xs text-purple-200 font-mono">
      <Sparkles class="w-5 h-5 text-purple-400 shrink-0" />
      <div>
        <span class="font-bold text-white">Multi-Website Scoping Hub:</span> Each connected website receives its own isolated API key, custom subdomain endpoint, and permitted AI model scope.
      </div>
    </div>

    <!-- Website Name -->
    <div>
      <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Website Name</label>
      <input
        type="text"
        bind:value={modalName}
        placeholder="e.g. My E-commerce Store"
        class="w-full px-3 py-2 rounded-lg bg-zinc-900 border border-zinc-800 text-xs font-mono text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-purple-600"
      />
    </div>

    <!-- Website Main Domain URL -->
    <div>
      <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Website Main Domain / URL</label>
      <input
        type="text"
        bind:value={modalDomain}
        placeholder="e.g. store.mybrand.com"
        class="w-full px-3 py-2 rounded-lg bg-zinc-900 border border-zinc-800 text-xs font-mono text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-purple-600"
      />
    </div>

    <!-- API Subdomain / Custom Endpoint Link -->
    <div>
      <label class="block text-xs font-semibold font-mono text-cyan-400 mb-1 flex items-center gap-1.5">
        <Link2 class="w-3.5 h-3.5" />
        <span>API Subdomain / Endpoint Link</span>
      </label>
      <input
        type="text"
        bind:value={modalSubdomain}
        placeholder="https://ai.store.mybrand.com/v1"
        class="w-full px-3 py-2 rounded-lg bg-cyan-950/30 border border-cyan-800/80 text-xs font-mono text-cyan-200 placeholder-cyan-700 focus:outline-none focus:border-cyan-500"
      />
      <p class="text-[11px] text-zinc-500 font-mono mt-1">
        This is the <code class="text-cyan-400 font-bold">OPENAI_BASE_URL</code> your website will call to access AI models.
      </p>
    </div>

    <!-- Allowed AI Models Scope -->
    <div>
      <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">
        Allowed AI Models (Scope)
      </label>
      <input
        type="text"
        bind:value={modalAllowedModels}
        placeholder="ag/gemini-2.5-flash, oc/space-bunny-free (or * for all)"
        class="w-full px-3 py-2 rounded-lg bg-zinc-900 border border-zinc-800 text-xs font-mono text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-purple-600"
      />
      <p class="text-[11px] text-zinc-500 font-mono mt-1">
        Comma-separated allowed patterns. Use <code class="text-purple-400 font-bold">*</code> for all models, or restrict to specific models.
      </p>
    </div>

    <!-- Plan & Speed Limit (RPM) -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
      <div>
        <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Plan Tier</label>
        <select
          bind:value={modalPlan}
          class="w-full px-3 py-2 rounded-lg bg-zinc-900 border border-zinc-800 text-xs font-mono text-zinc-100 focus:outline-none focus:border-purple-600 cursor-pointer"
        >
          <option value="free">FREE</option>
          <option value="standard">STANDARD</option>
          <option value="pro">PRO</option>
          <option value="vip">VIP</option>
          <option value="enterprise">ENTERPRISE</option>
        </select>
      </div>

      <div>
        <label class="block text-xs font-semibold font-mono text-zinc-400 mb-1">Speed Limit (RPM)</label>
        <input
          type="number"
          bind:value={modalRPMLimit}
          min="0"
          placeholder="60"
          class="w-full px-3 py-2 rounded-lg bg-zinc-900 border border-zinc-800 text-xs font-mono text-zinc-100 placeholder-zinc-500 focus:outline-none focus:border-purple-600"
        />
      </div>
    </div>

    <!-- Monthly Token Budget (Millions) -->
    <div>
      <div class="flex items-center justify-between mb-1 font-mono">
        <label class="text-xs font-semibold text-zinc-400">Monthly Token Budget (Millions)</label>
        <span class="text-xs text-purple-400 font-bold">
          {((Number(modalBudgetM) || 0) * 1_000_000).toLocaleString()} tokens
        </span>
      </div>
      <input
        type="number"
        bind:value={modalBudgetM}
        min="1"
        max="10000"
        class="w-full px-3 py-2 rounded-lg bg-zinc-900 border border-zinc-800 text-xs font-mono text-zinc-100 focus:outline-none focus:border-purple-600 mb-2"
      />
      <!-- Quick Presets -->
      <div class="flex flex-wrap gap-1.5">
        {#each budgetPresets as p}
          <button
            type="button"
            onclick={() => (modalBudgetM = p)}
            class="px-2.5 py-1 rounded text-[11px] font-mono border transition-all cursor-pointer {modalBudgetM === p
              ? 'border-purple-600 bg-purple-900/60 text-purple-200'
              : 'border-zinc-800 bg-zinc-900/60 text-zinc-400 hover:text-zinc-200 hover:bg-zinc-800'}"
          >
            {p}M
          </button>
        {/each}
      </div>
    </div>

    <div class="flex justify-end gap-2 pt-4 border-t border-zinc-800/80">
      <button
        type="button"
        onclick={onClose}
        class="px-3 py-1.5 rounded-lg text-xs font-mono border border-zinc-800 bg-zinc-900 text-zinc-300 hover:bg-zinc-800 transition-colors cursor-pointer"
      >
        Cancel
      </button>
      <button
        type="button"
        onclick={handleSave}
        disabled={isSaving}
        class="px-4 py-1.5 rounded-lg text-xs font-mono font-semibold bg-purple-600 hover:bg-purple-500 text-white transition-colors cursor-pointer disabled:opacity-50"
      >
        {isSaving ? 'Connecting...' : 'Connect Website & Generate Key'}
      </button>
    </div>
  </div>
</Modal>
