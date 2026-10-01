/**
 * Production WebSocket proxy: /ws -> Go MCP backend.
 *
 * The browser holds a single origin (the Nuxt service), so the raw WebSocket
 * cannot reach the Go server directly. Nitro terminates the upgrade here and
 * re-originates it onto the internal network, which also means the internal
 * hostname never has to be exposed to the browser.
 *
 * crossws drives this through *hooks* (`open` / `message` / `close`), not an
 * EventEmitter-style `.on()`: `Peer` exposes `send` and `close` only. Using
 * `.on()` here would throw at runtime the moment the first client frame
 * arrived, so the wiring below is hook-based on purpose.
 */
import { defineWebSocketHandler } from 'h3'

/**
 * Read from runtimeConfig rather than process.env directly: Nuxt only
 * propagates `NUXT_*` variables that are declared in `runtimeConfig`, and going
 * through the config object keeps that contract in one place.
 */
function upstreamWs(): string {
  const configured = useRuntimeConfig().apiUpstream || 'http://127.0.0.1:8080'
  return configured.replace(/^http/, 'ws').replace(/\/+$/, '')
}

/**
 * Credentials for the upstream handshake.
 *
 * A caller's own key is forwarded as-is; the SPA has no key, so the server-only
 * NUXT_API_KEY is injected instead. Because the handshake is dialled
 * server-to-server, the key never reaches the browser and never appears in the
 * upstream request line.
 *
 * The incoming credential is read from the upgrade request carried on the peer
 * (crossws exposes it there; there is no h3 event inside a WebSocket hook).
 *
 * The parameter is typed structurally rather than as crossws's `Peer` so this
 * file does not depend on a transitive package's exported type surface. Only
 * `request.headers` is used.
 */
function upstreamAuthHeaders(peer: { request?: { headers?: Headers } }): Record<string, string> {
  const headers: Record<string, string> = {}
  const incoming = peer.request?.headers

  const authz = incoming?.get?.('authorization')
  const apiKey = incoming?.get?.('x-api-key')
  const presented = authz ?? apiKey

  if (presented) {
    headers['x-api-key'] = presented
  } else {
    const configured = useRuntimeConfig().apiKey
    if (configured) headers['x-api-key'] = configured
  }
  return headers
}

/**
 * Reject handshakes from origins we do not serve.
 *
 * The Go service runs its own Origin check, but that check is bypassed for
 * browser traffic: this proxy dials upstream server-to-server with no Origin
 * header, and the Go handler treats an absent Origin as same-origin. Combined
 * with the credential being injected here for any caller, that would leave the
 * endpoint open to cross-site WebSocket hijacking — a page on any site could
 * open ws://<this-host>/ws?project_id=… and receive another user's project
 * events.
 *
 * So the browser-facing hop is validated here instead. Configure with
 * NUXT_WS_ALLOWED_ORIGINS; an empty value falls back to the request's own Host,
 * i.e. same-origin only, which is the correct default for this topology.
 */
function originAllowed(origin: string, host: string): boolean {
  const allowed = useRuntimeConfig().wsAllowedOrigins
  const list = (allowed || '')
    .split(',')
    .map((o) => o.trim())
    .filter(Boolean)

  if (list.length === 0) {
    // Same-origin fallback: no explicit list means only this host may connect.
    if (!host) return false
    try {
      return new URL(origin).host === host
    } catch {
      return false
    }
  }

  if (list.includes('*')) return true
  const lowered = origin.toLowerCase()
  return list.some((allowed) => {
    const a = allowed.toLowerCase()
    // Accept a sub-port form too: allow "https://app.example.com" to match
    // "https://app.example.com:8443", mirroring the Go-side allow-list.
    return lowered === a || lowered.startsWith(`${a}:`)
  })
}

export default defineWebSocketHandler({
  // `upgrade` runs before the socket exists.
  //
  // crossws turns a rejected upgrade into a real HTTP response only when this
  // hook *returns* a Response: its wrapper reads `res.ok === false` and passes
  // it to `sendResponse`. Throwing does not work here — the catch block only
  // converts a throw that is `instanceof Response` (or has a `.response` that
  // is), and an h3 `createError` is an H3Error, so it would be re-thrown and
  // leave the socket hanging instead of refusing the handshake.
  upgrade(request) {
    const origin = request.headers.get('origin')
    const host = request.headers.get('host') ?? ''

    // A missing Origin means a non-browser client (CLI, test harness). Those
    // must present a credential, which the Go service enforces, and there is
    // no ambient cookie to ride on, so there is no hijacking surface here.
    if (!origin) return

    if (!originAllowed(origin, host)) {
      // No statusMessage: the DOM ResponseInit type omits it even though
      // undici honours it, and the 403 status line is what the client sees.
      return new Response('Forbidden', {
        status: 403,
        headers: { 'content-type': 'text/plain; charset=utf-8' },
      })
    }
  },

  open(peer) {
    const requestUrl = peer.request?.url ?? '/ws'
    const url = new URL(requestUrl, 'http://localhost')
    const projectId = url.searchParams.get('project_id')

    if (!projectId) {
      peer.send(JSON.stringify({ type: 'error', message: 'project_id is required' }))
      peer.close(1008, 'project_id is required')
      return
    }

    const target = `${upstreamWs()}/ws?project_id=${encodeURIComponent(projectId)}`

    let upstream: WebSocket
    try {
      // Node's global WebSocket (undici) accepts a WHATWG options dictionary as
      // its second argument, so `headers` is honoured at runtime. The bundled
      // DOM lib still types that parameter as the subprotocol list, so the
      // cast is confined to this single call rather than weakening the file.
      //
      // Injecting the credential on the handshake — rather than as a query
      // parameter — keeps the key out of upstream access logs. A caller-supplied
      // key still wins, so a headless client can use its own credential.
      upstream = new WebSocket(
        target,
        { headers: upstreamAuthHeaders(peer) } as unknown as string[],
      )
    } catch (err) {
      peer.send(JSON.stringify({ type: 'error', message: `upstream dial failed: ${(err as Error).message}` }))
      peer.close(1011, 'upstream dial failed')
      return
    }

    // Stash the upstream socket on the peer's context so the `message` and
    // `close` hooks below can reach it without a module-level Map keyed by peer
    // id (which would leak on an abrupt disconnect).
    peer.context.upstream = upstream

    upstream.addEventListener('open', () => {
      try {
        peer.send(JSON.stringify({ type: 'ws:connected', project_id: projectId }))
      } catch {
        // The client vanished between upgrade and dial; nothing to do.
      }
    })

    upstream.addEventListener('message', (ev) => {
      try {
        peer.send((ev as MessageEvent).data as string)
      } catch {
        // Downstream backpressure or a closed socket; the close hook cleans up.
      }
    })

    upstream.addEventListener('error', () => {
      try {
        peer.send(JSON.stringify({ type: 'error', message: 'upstream websocket error' }))
        peer.close(1011, 'upstream websocket error')
      } catch {
        // Already closed.
      }
    })
  },

  message(peer, message) {
    const upstream = peer.context.upstream as WebSocket | undefined
    if (!upstream || upstream.readyState !== WebSocket.OPEN) return
    try {
      upstream.send(message.text())
    } catch {
      // Upstream closed between the state check and the send; the close hook
      // will tear down the peer.
    }
  },

  close(peer) {
    const upstream = peer.context.upstream as WebSocket | undefined
    if (!upstream) return
    try {
      if (upstream.readyState === WebSocket.OPEN || upstream.readyState === WebSocket.CONNECTING) {
        upstream.close()
      }
    } catch {
      // Already closing.
    }
  },
})
