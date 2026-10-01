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

export default defineWebSocketHandler({
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
      upstream = new WebSocket(target)
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
