<script lang="ts">
  import { api, type APIKey } from '../api/client'
  import ApiKeysTable from './keys/ApiKeysTable.svelte'
  import { notifications } from '../lib/notifications'

  interface Props {
    apiKeys?: APIKey[]
    onRefresh?: () => void | Promise<void>
    onNavigateEndpoint?: () => void
  }

  let { apiKeys = [], onRefresh = () => {}, onNavigateEndpoint = () => {} }: Props = $props()

  let showCreate = $state(false)
  let name = $state('')
  let tokenQuota = $state('')
  let contextLimit = $state('')
  let creating = $state(false)
  let createdKey = $state<string | null>(null)
  let copied = $state(false)

  const activeCount = $derived(apiKeys.filter((key) => key.isActive === 1).length)

  async function createKey(event: SubmitEvent) {
    event.preventDefault()
    if (!name.trim() || creating) return

    creating = true
    try {
      const res = await api.createApiKey({
        name: name.trim(),
        ...(tokenQuota.trim() ? { tokenQuota: Number(tokenQuota.trim()) } : {}),
        ...(contextLimit.trim() ? { contextLimit: Number(contextLimit.trim()) } : {}),
      })
      if (res.key) {
        createdKey = res.key
        name = ''
        tokenQuota = ''
        contextLimit = ''
        showCreate = false
        await onRefresh()
      }
    } catch (err) {
      notifications.error(`Failed to create key: ${err instanceof Error ? err.message : String(err)}`)
    } finally {
      creating = false
    }
  }

  async function copyCreatedKey() {
    if (!createdKey) return
    try {
      await navigator.clipboard.writeText(createdKey)
      copied = true
      setTimeout(() => (copied = false), 1500)
    } catch {
      notifications.error('Could not copy the key to the clipboard')
    }
  }
</script>

<div class="space-y-6">
  <!-- Header -->
  <div class="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
    <div>
      <div class="flex items-center gap-3">
        <span class="material-symbols-outlined text-[28px] text-brand-500">vpn_key</span>
        <h1 class="text-2xl font-bold text-text-main">API Keys</h1>
      </div>
      <p class="mt-1 max-w-2xl text-sm text-text-muted">
        Client keys for the gateway <code class="rounded bg-surface-2 px-1 py-0.5 text-[12px]">/v1</code> endpoints.
        {activeCount} active of {apiKeys.length} issued. Quotas and access rules are edited per key below.
      </p>
    </div>
  </div>

  <!-- One-time secret reveal -->
  {#if createdKey}
    <div class="rounded-[10px] border border-brand-500/40 bg-brand-500/5 p-4">
      <div class="flex items-center justify-between gap-3">
        <div class="min-w-0">
          <p class="text-xs font-semibold uppercase tracking-wide text-brand-500">Key created — store it now</p>
          <p class="mt-1 break-all font-code text-sm text-text-main">{createdKey}</p>
        </div>
        <button
          type="button"
          class="shrink-0 rounded-[10px] border border-border bg-surface-2 p-2 text-text-muted hover:text-text-main"
          title="Copy key"
          onclick={copyCreatedKey}
        >
          <span class="material-symbols-outlined text-[20px]">{copied ? 'check' : 'content_copy'}</span>
        </button>
      </div>
      <p class="mt-2 text-[11px] text-text-subtle">This secret is shown once. It is stored masked in the database; a lost key cannot be recovered.</p>
      <button
        type="button"
        class="mt-3 inline-flex items-center gap-1.5 rounded-[10px] bg-primary px-4 py-2 text-xs font-semibold text-white hover:bg-primary-hover"
        onclick={() => (createdKey = null)}
      >
        Done
      </button>
    </div>
  {/if}

  <!-- Keys table — the table's own “Create Key” button toggles the inline form -->
  <ApiKeysTable {apiKeys} onRefresh={onRefresh} onCreate={() => (showCreate = !showCreate)} />

  {#if showCreate}
    <form onsubmit={createKey} class="rounded-[10px] border border-border bg-surface p-4 space-y-3">
      <h2 class="text-sm font-semibold text-text-main">New client key</h2>
      <label for="key-name-input" class="text-[10px] font-semibold uppercase tracking-wide text-text-subtle">
        Key Name
      </label>
      <input
        id="key-name-input"
        type="text"
        bind:value={name}
        placeholder="Name (e.g. ci-runner, laptop)"
        class="w-full rounded-[10px] border border-border bg-surface-2 px-3 py-2 text-sm text-text-main placeholder:text-text-subtle focus:outline-none focus:ring-1 focus:ring-brand-500"
      />
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <input
          type="text"
          inputmode="numeric"
          bind:value={tokenQuota}
          placeholder="Token budget (0 = unlimited)"
          class="w-full rounded-[10px] border border-border bg-surface-2 px-3 py-2 text-sm text-text-main placeholder:text-text-subtle focus:outline-none focus:ring-1 focus:ring-brand-500"
        />
        <input
          type="text"
          inputmode="numeric"
          bind:value={contextLimit}
          placeholder="Context limit (0 = model default)"
          class="w-full rounded-[10px] border border-border bg-surface-2 px-3 py-2 text-sm text-text-main placeholder:text-text-subtle focus:outline-none focus:ring-1 focus:ring-brand-500"
        />
      </div>
      <div class="flex items-center gap-3">
        <button
          type="submit"
          disabled={!name.trim() || creating}
          class="rounded-[10px] bg-primary px-4 py-2 text-sm font-semibold text-white hover:bg-primary-hover disabled:opacity-50"
        >
          {creating ? 'Creating…' : 'Create key'}
        </button>
        <button
          type="button"
          class="text-xs text-text-muted hover:text-text-main underline underline-offset-2"
          onclick={onNavigateEndpoint}
        >
          Bulk create &amp; machine IDs live on the API Endpoints page →
        </button>
      </div>
    </form>
  {/if}
</div>
