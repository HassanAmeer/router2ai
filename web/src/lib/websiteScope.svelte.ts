import { api, type ResellerUser } from '../api/client'

// Global reactive state for website scoping across the entire dashboard
let selectedWebsiteId = $state<string>('all')
let websites = $state<ResellerUser[]>([])
let isLoading = $state(false)
let isAddModalOpen = $state(false)

export function getWebsiteScope() {
  return {
    get selectedWebsiteId(): string {
      return selectedWebsiteId
    },
    set selectedWebsiteId(id: string) {
      selectedWebsiteId = id
      if (typeof window !== 'undefined') {
        localStorage.setItem('9router_selected_website', id)
      }
    },
    get websites(): ResellerUser[] {
      return websites
    },
    get activeWebsite(): ResellerUser | null {
      if (selectedWebsiteId === 'all') return null
      return websites.find((w) => w.id === selectedWebsiteId) || null
    },
    get isLoading(): boolean {
      return isLoading
    },
    get isAddModalOpen(): boolean {
      return isAddModalOpen
    },
    set isAddModalOpen(val: boolean) {
      isAddModalOpen = val
    },
    async loadWebsites() {
      isLoading = true
      try {
        const res = await api.getResellerUsers()
        websites = res.users || []
        // Restore from localStorage if valid
        if (typeof window !== 'undefined') {
          const stored = localStorage.getItem('9router_selected_website')
          if (stored && (stored === 'all' || websites.some((w) => w.id === stored))) {
            selectedWebsiteId = stored
          }
        }
      } catch (err) {
        console.error('Failed to load websites:', err)
      } finally {
        isLoading = false
      }
    },
    isModelAllowed(modelId: string): boolean {
      if (selectedWebsiteId === 'all') return true
      const active = websites.find((w) => w.id === selectedWebsiteId)
      if (!active || !active.allowedModels || active.allowedModels.trim() === '*' || active.allowedModels.trim() === '') {
        return true
      }
      const patterns = active.allowedModels.split(',').map((p) => p.trim().toLowerCase())
      const m = modelId.toLowerCase()
      return patterns.some((p) => {
        if (p === '*' || p === m) return true
        if (p.endsWith('*')) return m.startsWith(p.slice(0, -1))
        return m.includes(p)
      })
    }
  }
}
