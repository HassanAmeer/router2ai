<script lang="ts">
  import { onMount } from 'svelte'
  import { getWebsiteScope } from '../../lib/websiteScope.svelte'
  import AddWebsiteModal from './AddWebsiteModal.svelte'
  import { Globe, ChevronDown, Check, Plus, ExternalLink, Settings, Sparkles, X, ShieldAlert } from 'lucide-svelte'

  let {
    iconRail = false,
    onNavigateStudio,
  }: {
    iconRail?: boolean
    onNavigateStudio?: () => void
  } = $props()

  const scope = getWebsiteScope()

  let isOpen = $state(false)
  let filterText = $state('')
  let dropdownEl = $state<HTMLDivElement | null>(null)

  onMount(() => {
    scope.loadWebsites()
  })

  // Close dropdown on click outside
  $effect(() => {
    if (!isOpen) return
    function handleClickOutside(e: MouseEvent) {
      if (dropdownEl && !dropdownEl.contains(e.target as Node)) {
        isOpen = false
      }
    }
    window.addEventListener('click', handleClickOutside)
    return () => window.removeEventListener('click', handleClickOutside)
  })

  let filteredWebsites = $derived(
    scope.websites.filter(
      (w) =>
        w.name.toLowerCase().includes(filterText.toLowerCase()) ||
        w.email.toLowerCase().includes(filterText.toLowerCase()) ||
        (w.subdomain && w.subdomain.toLowerCase().includes(filterText.toLowerCase()))
    )
  )

  function selectScope(id: string) {
    scope.selectedWebsiteId = id
    isOpen = false
  }
</script>

<!-- Add Website Modal -->
<AddWebsiteModal
  isOpen={scope.isAddModalOpen}
  onClose={() => (scope.isAddModalOpen = false)}
  onSuccess={() => {
    isOpen = false
  }}
/>

{#if iconRail}
  <!-- Compact Icon Rail Mode -->
  <div class="relative flex justify-center py-1.5" bind:this={dropdownEl}>
    <button
      type="button"
      onclick={(e) => {
        e.stopPropagation()
        isOpen = !isOpen
      }}
      title={scope.activeWebsite ? `Scope: ${scope.activeWebsite.name}` : 'All Websites (Global Scope)'}
      class="relative size-9 rounded-lg flex items-center justify-center transition-all cursor-pointer {scope.activeWebsite
        ? 'border border-cyan-500/60 bg-cyan-950/60 text-cyan-300 shadow-sm shadow-cyan-500/20'
        : 'border border-purple-800/60 bg-purple-950/40 text-purple-300 hover:bg-purple-900/60 hover:text-white'}"
    >
      <Globe class="size-4.5" />
      {#if scope.activeWebsite}
        <span class="absolute top-1 right-1 size-2 rounded-full bg-cyan-400 animate-pulse"></span>
      {/if}
    </button>
  </div>
{:else}
  <!-- Full Sidebar Mode: Directly below Providers tab -->
  <div class="px-3 pt-1.5 pb-2 relative" bind:this={dropdownEl}>
    <!-- Container Box -->
    <div class="rounded-xl border border-purple-800/50 bg-purple-950/30 p-2 shadow-xs transition-all hover:border-purple-700/70">
      <!-- Section Label + Quick Add Button -->
      <div class="flex items-center justify-between px-1 mb-1.5 font-mono">
        <span class="text-[10px] uppercase font-bold tracking-wider text-purple-300/80 flex items-center gap-1">
          <Globe class="w-3 h-3 text-purple-400" />
          <span>Active Scope</span>
        </span>
        <div class="flex items-center gap-1.5">
          <span class="text-[10px] text-zinc-500 font-mono">
            {scope.websites.length} sites
          </span>
          <button
            type="button"
            onclick={() => (scope.isAddModalOpen = true)}
            title="Add New Website"
            class="p-0.5 rounded text-purple-400 hover:text-white hover:bg-purple-800/60 transition-colors cursor-pointer"
          >
            <Plus class="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      <!-- Dropdown Trigger Button -->
      <button
        type="button"
        onclick={(e) => {
          e.stopPropagation()
          isOpen = !isOpen
        }}
        class="w-full flex items-center justify-between gap-2 px-2.5 py-1.5 rounded-lg border transition-all cursor-pointer text-left {scope.activeWebsite
          ? 'border-cyan-700/80 bg-cyan-950/50 text-cyan-200'
          : 'border-purple-800/60 bg-purple-900/40 text-zinc-200 hover:bg-purple-900/60 hover:text-white'}"
      >
        <div class="flex items-center gap-2 min-w-0 flex-1">
          <div class="size-6 rounded-md flex items-center justify-center shrink-0 {scope.activeWebsite ? 'bg-cyan-900/60 text-cyan-300' : 'bg-purple-950 text-purple-300'}">
            <Globe class="w-3.5 h-3.5" />
          </div>
          <div class="flex flex-col min-w-0">
            <span class="text-xs font-mono font-bold truncate leading-tight">
              {scope.activeWebsite ? scope.activeWebsite.name : 'All Websites (Global)'}
            </span>
            <span class="text-[10px] font-mono text-zinc-400 truncate leading-tight">
              {scope.activeWebsite
                ? scope.activeWebsite.subdomain || scope.activeWebsite.email
                : 'Universal Routing Scope'}
            </span>
          </div>
        </div>

        <ChevronDown class="w-4 h-4 text-purple-400 shrink-0 transition-transform duration-200 {isOpen ? 'rotate-180' : ''}" />
      </button>
    </div>

    <!-- Dropdown Menu Popover -->
    {#if isOpen}
      <div
        class="absolute left-3 right-3 top-full mt-1.5 z-50 rounded-xl border border-purple-700/80 bg-zinc-950/95 backdrop-blur-xl shadow-2xl p-2 animate-in fade-in zoom-in-95 duration-150 font-mono"
      >
        <!-- Filter Search Bar -->
        <div class="mb-2">
          <input
            type="text"
            bind:value={filterText}
            placeholder="Search website or subdomain..."
            class="w-full px-2.5 py-1.5 rounded-lg bg-zinc-900 border border-zinc-800 text-[11px] text-zinc-200 placeholder-zinc-500 focus:outline-none focus:border-purple-600"
          />
        </div>

        <div class="max-h-56 overflow-y-auto custom-scrollbar space-y-1">
          <!-- Option: Global (All Websites) -->
          <button
            type="button"
            onclick={() => selectScope('all')}
            class="w-full flex items-center justify-between gap-2 px-2.5 py-1.5 rounded-lg text-left transition-all cursor-pointer {scope.selectedWebsiteId === 'all'
              ? 'bg-purple-900/60 text-purple-100 font-bold border border-purple-700/60'
              : 'text-zinc-300 hover:bg-zinc-900 hover:text-white'}"
          >
            <div class="flex items-center gap-2 min-w-0">
              <Globe class="w-4 h-4 text-purple-400 shrink-0" />
              <div class="flex flex-col min-w-0">
                <span class="text-xs truncate">All Websites (Global Scope)</span>
                <span class="text-[10px] text-zinc-500">Unrestricted full gateway view</span>
              </div>
            </div>
            {#if scope.selectedWebsiteId === 'all'}
              <Check class="w-3.5 h-3.5 text-purple-300 shrink-0" />
            {/if}
          </button>

          <!-- Divider -->
          <div class="my-1 border-t border-zinc-800/80"></div>
          <div class="px-2 py-0.5 text-[9px] uppercase tracking-wider text-zinc-500 font-semibold">
            Connected Websites ({scope.websites.length})
          </div>

          <!-- List of Connected Websites -->
          {#each filteredWebsites as w (w.id)}
            {@const isSelected = scope.selectedWebsiteId === w.id}
            <button
              type="button"
              onclick={() => selectScope(w.id)}
              class="w-full flex items-center justify-between gap-2 px-2.5 py-1.5 rounded-lg text-left transition-all cursor-pointer {isSelected
                ? 'bg-cyan-950/70 text-cyan-200 font-bold border border-cyan-700/80 shadow-xs'
                : 'text-zinc-300 hover:bg-zinc-900 hover:text-white'}"
            >
              <div class="flex items-center gap-2 min-w-0 flex-1">
                <div class="size-2 rounded-full {w.isBanned ? 'bg-red-500' : isSelected ? 'bg-cyan-400' : 'bg-emerald-500'} shrink-0"></div>
                <div class="flex flex-col min-w-0 flex-1">
                  <div class="flex items-center gap-1.5 truncate">
                    <span class="text-xs truncate text-zinc-100">{w.name}</span>
                    <span class="px-1.5 py-0.2 rounded text-[9px] font-bold uppercase border {w.planTier === 'vip' || w.planTier === 'enterprise'
                      ? 'border-purple-800 bg-purple-950 text-purple-300'
                      : 'border-zinc-800 bg-zinc-900 text-zinc-400'}">
                      {w.planTier}
                    </span>
                  </div>
                  <span class="text-[10px] text-cyan-400/80 truncate">
                    {w.subdomain || w.email}
                  </span>
                </div>
              </div>

              {#if isSelected}
                <Check class="w-3.5 h-3.5 text-cyan-400 shrink-0" />
              {/if}
            </button>
          {/each}

          {#if filteredWebsites.length === 0}
            <div class="p-3 text-center text-xs text-zinc-500">
              No matching websites found
            </div>
          {/if}
        </div>

        <!-- Footer Action Buttons -->
        <div class="pt-2 mt-2 border-t border-zinc-800/80 flex items-center justify-between gap-2">
          <button
            type="button"
            onclick={() => {
              scope.isAddModalOpen = true
              isOpen = false
            }}
            class="flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-semibold border border-purple-700/80 bg-purple-900/50 text-purple-200 hover:bg-purple-800/70 hover:text-white transition-all cursor-pointer"
          >
            <Plus class="w-3.5 h-3.5" />
            <span>Add Website</span>
          </button>

          {#if onNavigateStudio}
            <button
              type="button"
              onclick={() => {
                onNavigateStudio()
                isOpen = false
              }}
              class="flex items-center gap-1 px-2 py-1 rounded-md text-[11px] text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900 transition-colors cursor-pointer"
            >
              <Settings class="w-3 h-3" />
              <span>Manage Scopes</span>
            </button>
          {/if}
        </div>
      </div>
    {/if}
  </div>
{/if}
