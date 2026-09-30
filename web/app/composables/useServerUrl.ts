/**
 * Server URL is the single most cross-cutting piece of state.
 * Stored in a cookie so SSR and CSR see the same value (no hydration mismatch).
 *
 * Replaces the old `web/src/stores/settingsStore.js` localStorage pattern.
 */

const COOKIE = 'mcp-server-url'
const DEFAULT_URL = 'http://localhost:8080'

export interface UseServerUrl {
  url: ReturnType<typeof computed<string>>
  apiBase: ReturnType<typeof computed<string>>
  wsBase: ReturnType<typeof computed<string>>
  set: (next: string) => Promise<void>
  reset: () => Promise<void>
  isDefault: ReturnType<typeof computed<boolean>>
}

function deriveApiBase(raw: string): string {
  try {
    const u = new URL(raw)
    return `${u.protocol}//${u.host}/api`
  } catch {
    return `${raw.replace(/\/$/, '')}/api`
  }
}

function deriveWsBase(raw: string): string {
  try {
    const u = new URL(raw)
    const scheme = u.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${scheme}//${u.host}/ws`
  } catch {
    return raw.replace(/^http/, 'ws').replace(/\/$/, '') + '/ws'
  }
}

export function useServerUrl(): UseServerUrl {
  const cookie = useCookie<string>(COOKIE, {
    default: () => DEFAULT_URL,
    sameSite: 'lax',
    maxAge: 60 * 60 * 24 * 30 // 30 days
  })

  const url = computed(() => cookie.value || DEFAULT_URL)
  const apiBase = computed(() => deriveApiBase(url.value))
  const wsBase = computed(() => deriveWsBase(url.value))
  const isDefault = computed(() => url.value === DEFAULT_URL)

  async function set(next: string): Promise<void> {
    const trimmed = (next || '').trim().replace(/\/$/, '')
    if (!trimmed) return
    cookie.value = trimmed
    // Reset realtime so the layout picks up the new URL on next nav.
    const realtime = useRealtime()
    realtime.disconnect()
  }

  async function reset(): Promise<void> {
    await set(DEFAULT_URL)
  }

  return { url, apiBase, wsBase, set, reset, isDefault }
}
