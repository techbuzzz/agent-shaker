/**
 * WebSocket client with auto-reconnect (exponential backoff) and a tiny in-memory
 * event bus. Subscribers can `on(type, handler)` and `off(...)`.
 *
 * The connection is project-scoped — calling `connect(projectId)` closes any
 * existing socket and opens a new one. `disconnect()` tears it down cleanly.
 *
 * Replaces web/src/composables/useWebSocket.js.
 */
import type { WsEvent, WsEventType, WsEventHandler } from '~/types/ws'

type AnyHandler = WsEventHandler<WsEvent>

interface State {
  socket: WebSocket | null
  projectId: string | null
  reconnectAttempt: number
  reconnectTimer: ReturnType<typeof setTimeout> | null
  status: 'idle' | 'connecting' | 'open' | 'closed'
  heartbeatTimer: ReturnType<typeof setInterval> | null
  lastError: string | null
}

const BACKOFF_MS = [1000, 2000, 5000, 10000, 30000]
const HEARTBEAT_MS = 25000

export interface UseRealtime {
  state: () => Readonly<State>
  isConnected: Ref<boolean>
  status: Ref<State['status']>
  projectId: Ref<string | null>
  connect: (projectId: string) => void
  disconnect: () => void
  send: (data: unknown) => void
  on: <E extends WsEvent = WsEvent>(type: WsEventType | '*', handler: WsEventHandler<E>) => () => void
  off: (type: WsEventType | '*', handler: AnyHandler) => void
}

function backoffDelay(attempt: number): number {
  const idx = Math.min(attempt, BACKOFF_MS.length - 1)
  return BACKOFF_MS[idx]!
}

export function useRealtime(): UseRealtime {
  const s: State = {
    socket: null,
    projectId: null,
    reconnectAttempt: 0,
    reconnectTimer: null,
    status: 'idle',
    heartbeatTimer: null,
    lastError: null
  }
  const listeners = new Map<WsEventType | '*', Set<AnyHandler>>()
  const isClient = () => typeof window !== 'undefined'

  const isConnected = ref(false)
  const status = ref<State['status']>('idle')
  const projectId = ref<string | null>(null)

  function notify(event: WsEvent) {
    const set = listeners.get(event.type)
    if (set) for (const h of set) try { (h as WsEventHandler<typeof event>)(event) } catch (e) { console.error(e) }
    const wild = listeners.get('*')
    if (wild) for (const h of wild) try { (h as WsEventHandler<typeof event>)(event) } catch (e) { console.error(e) }
  }

  function clearTimers() {
    if (s.reconnectTimer) { clearTimeout(s.reconnectTimer); s.reconnectTimer = null }
    if (s.heartbeatTimer) { clearInterval(s.heartbeatTimer); s.heartbeatTimer = null }
  }

  function openSocket() {
    if (!isClient() || !s.projectId) return
    const { wsBase, url } = useServerUrl()
    let base = wsBase.value
    try {
      const u = new URL(url.value)
      // Same-origin fallback (when proxy is wired): use relative ws://host/ws
      const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      base = `${proto}//${window.location.host}/ws`
      // but allow absolute override when serverUrl host differs
      if (u.host && u.host !== window.location.host) base = wsBase.value
    } catch { /* ignore */ }
    const wsUrl = `${base}?project_id=${encodeURIComponent(s.projectId)}`

    s.status = 'connecting'
    status.value = 'connecting'
    let socket: WebSocket
    try {
      socket = new WebSocket(wsUrl)
    } catch (e) {
      s.lastError = (e as Error).message
      scheduleReconnect()
      return
    }
    s.socket = socket

    socket.onopen = () => {
      s.status = 'open'
      status.value = 'open'
      isConnected.value = true
      s.reconnectAttempt = 0
      // Heartbeat ping keeps the proxy happy and surfaces dead connections fast.
      s.heartbeatTimer = setInterval(() => {
        try { socket.readyState === WebSocket.OPEN && socket.send(JSON.stringify({ type: 'ping' })) } catch { /* ignore */ }
      }, HEARTBEAT_MS)
      // Surface reconnection
      if (s.reconnectAttempt > 0) {
        const bus = useToastBus()
        bus.info('Reconnected to live updates.')
      }
    }
    socket.onerror = (ev) => {
      s.lastError = (ev as Event).type || 'WebSocket error'
    }
    socket.onclose = () => {
      isConnected.value = false
      s.status = 'closed'
      status.value = 'closed'
      clearTimers()
      s.socket = null
      scheduleReconnect()
    }
    socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as WsEvent
        if (data && data.type) notify(data)
      } catch (e) { console.error('Failed to parse WS frame', e) }
    }
  }

  function scheduleReconnect() {
    if (!s.projectId) return
    const delay = backoffDelay(s.reconnectAttempt)
    s.reconnectAttempt++
    s.reconnectTimer = setTimeout(openSocket, delay)
  }

  function connect(projectId: string) {
    if (s.projectId === projectId && s.socket) return
    disconnect()
    if (!isClient()) return
    s.projectId = projectId
    s.reconnectAttempt = 0
    projectId.value = projectId
    openSocket()
  }

  function disconnect() {
    clearTimers()
    if (s.socket) {
      try { s.socket.close() } catch { /* ignore */ }
      s.socket = null
    }
    s.projectId = null
    s.reconnectAttempt = 0
    s.status = 'idle'
    status.value = 'idle'
    isConnected.value = false
    projectId.value = null
  }

  function send(data: unknown) {
    if (s.socket && s.socket.readyState === WebSocket.OPEN) {
      try { s.socket.send(JSON.stringify(data)) } catch (e) { console.error('WS send failed', e) }
    }
  }

  function on<E extends WsEvent = WsEvent>(type: WsEventType | '*', handler: WsEventHandler<E>) {
    const key = type
    if (!listeners.has(key)) listeners.set(key, new Set())
    listeners.get(key)!.add(handler as AnyHandler)
    return () => off(key, handler as AnyHandler)
  }

  function off(type: WsEventType | '*', handler: AnyHandler) {
    listeners.get(type)?.delete(handler)
  }

  return {
    state: () => s,
    isConnected,
    status,
    projectId,
    connect,
    disconnect,
    send,
    on,
    off
  }
}
