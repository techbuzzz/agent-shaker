/**
 * Settings store: cross-page UI preferences (table density, last-used project filter).
 * Server URL lives in useServerUrl (cookie-backed) — NOT in this store, to avoid
 * hydration mismatches between SSR and client.
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useSettingsStore = defineStore('settings', () => {
  /**
   * Cookies are the durable source of truth; the store exposes plain refs.
   *
   * The cookie refs below are captured in this closure and deliberately NOT
   * returned from the store. Pinia unwraps Vue refs into its own state, but it
   * does not unwrap a Nuxt `CookieRef`: returning one would leave a Ref
   * *object* in the store's state, and SSR payload serialisation then calls
   * `hasOwnProperty` on it and throws
   * "obj.hasOwnProperty is not a function", 500-ing every page.
   *
   * Reading `.value` here still happens during SSR, so the first paint already
   * has the persisted value — no hydration mismatch.
   */
  const tableDensityCookie = useCookie<'comfortable' | 'compact'>('ui:table-density', {
    default: () => 'comfortable',
    sameSite: 'lax',
  })

  // The most recent agent this browser session has interacted with — used
  // for PM-role gating and as the default agent_id on the global-context
  // publish modal. Set by the new-agent / connect flows.
  const recentAgentCookie = useCookie<string | null>('ui:recent-agent', {
    default: () => null,
    sameSite: 'lax',
  })

  const lastProjectCookie = useCookie<string | null>('ui:last-project', {
    default: () => null,
    sameSite: 'lax',
  })

  const tableDensity = ref<'comfortable' | 'compact'>(tableDensityCookie.value ?? 'comfortable')
  const recentAgentId = ref<string | null>(recentAgentCookie.value ?? null)
  const lastProjectId = ref<string | null>(lastProjectCookie.value ?? null)

  function setDensity(v: 'comfortable' | 'compact') {
    tableDensity.value = v
    tableDensityCookie.value = v
  }

  function rememberProject(id: string | null) {
    lastProjectId.value = id
    lastProjectCookie.value = id
  }

  function rememberAgent(id: string | null) {
    recentAgentId.value = id
    recentAgentCookie.value = id
  }

  return {
    tableDensity,
    lastProjectId,
    recentAgentId,
    setDensity,
    rememberProject,
    rememberAgent,
  }
})
