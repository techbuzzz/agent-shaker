/**
 * Server URL is the single most cross-cutting piece of state.
 * Stored in a cookie so SSR and CSR see the same value (no hydration mismatch).
 *
 * Replaces the old `web/src/stores/settingsStore.js` localStorage pattern.
 */

const COOKIE = 'mcp-server-url'

/**
 * Empty by default. An empty value means "same origin as this page", so
 * `apiBase` resolves to `/api` and `wsBase` to `/ws` — served by the Nitro dev
 * proxy locally and by the Nitro server proxy in production.
 *
 * This is what makes single-origin deployment work: no CORS negotiation, no
 * second host, no cookie sent cross-origin. Only a user who explicitly points
 * the app at a *different* backend (the Settings page) sets the cookie.
 */
const DEFAULT_URL = ''

/**
 * True when the value denotes this page's own origin, or nothing at all.
 *
 * Runs during SSR too, where `window` is undefined; the `startsWith('/')` and
 * empty-string branches keep that safe without a `import.meta.client` guard.
 */
function isSameOrigin(raw: string): boolean {
  if (!raw) return true
  if (raw.startsWith('/')) return true
  try {
    return new URL(raw).host === window.location.host
  } catch {
    // Relative or unparseable: treat as same-origin, the proxy will handle it.
    return true
  }
}

export interface UseServerUrl {
  /**
   * Declared as `ComputedRef` rather than `ReturnType<typeof computed<T>>`:
   * the latter resolves to `WritableComputedRef`, which advertises a setter
   * this composable does not provide — callers could then assign to `url.value`
   * and silently bypass the cookie, which is the actual source of truth.
   */
  url: ComputedRef<string>
  apiBase: ComputedRef<string>
  wsBase: ComputedRef<string>
  set: (next: string) => Promise<void>
  reset: () => Promise<void>
  isDefault: ComputedRef<boolean>
}

function deriveApiBase(raw: string): string {
  if (isSameOrigin(raw)) return '/api'
  try {
    const u = new URL(raw)
    return `${u.protocol}//${u.host}/api`
  } catch {
    return `${raw.replace(/\/$/, '')}/api`
  }
}

function deriveWsBase(raw: string): string {
  if (isSameOrigin(raw)) return '/ws'
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
