/**
 * Settings store: cross-page UI preferences (table density, last-used project filter).
 * Server URL lives in useServerUrl (cookie-backed) — NOT in this store, to avoid
 * hydration mismatches between SSR and client.
 */
import { defineStore } from 'pinia'

export const useSettingsStore = defineStore('settings', () => {
  const tableDensity = useCookie<'comfortable' | 'compact'>('ui:table-density', {
    default: () => 'comfortable',
    sameSite: 'lax'
  })

  const lastProjectId = useCookie<string | null>('ui:last-project', {
    default: () => null,
    sameSite: 'lax'
  })

  // The most recent agent this browser session has interacted with — used
  // for PM-role gating and as the default agent_id on the global-context
  // publish modal. Set by the new-agent / connect flows.
  const recentAgentId = useCookie<string | null>('ui:recent-agent', {
    default: () => null,
    sameSite: 'lax'
  })

  function setDensity(v: 'comfortable' | 'compact') { tableDensity.value = v }
  function rememberProject(id: string | null) { lastProjectId.value = id }
  function rememberAgent(id: string | null) { recentAgentId.value = id }

  return {
    tableDensity,
    lastProjectId,
    recentAgentId,
    setDensity,
    rememberProject,
    rememberAgent
  }
})
