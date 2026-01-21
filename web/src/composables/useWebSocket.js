import { ref } from 'vue'

const ws = ref(null)
const isConnected = ref(false)
const listeners = new Map()

export function useWebSocket(projectId) {
  const connect = () => {
    if (!projectId) {
      console.error('Project ID is required for WebSocket connection')
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.hostname}:${window.location.port}/ws?project_id=${encodeURIComponent(projectId)}`
    
    ws.value = new WebSocket(wsUrl)

    ws.value.onopen = () => {
      console.log('WebSocket connected for project:', projectId)
      isConnected.value = true
    }

    ws.value.onclose = () => {
      console.log('WebSocket disconnected')
      isConnected.value = false
      // Reconnect after 3 seconds
      setTimeout(() => {
        if (!isConnected.value) {
          connect()
        }
      }, 3000)
    }

    ws.value.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    ws.value.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        notifyListeners(data)
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }
  }

  const disconnect = () => {
    if (ws.value) {
      ws.value.close()
      ws.value = null
      isConnected.value = false
    }
  }

  const send = (data) => {
    if (ws.value && isConnected.value) {
      ws.value.send(JSON.stringify(data))
    }
  }

  const on = (eventType, callback) => {
    if (!listeners.has(eventType)) {
      listeners.set(eventType, [])
    }
    listeners.get(eventType).push(callback)
  }

  const off = (eventType, callback) => {
    if (listeners.has(eventType)) {
      const callbacks = listeners.get(eventType)
      const index = callbacks.indexOf(callback)
      if (index > -1) {
        callbacks.splice(index, 1)
      }
    }
  }

  const notifyListeners = (data) => {
    const eventType = data.type || 'message'
    if (listeners.has(eventType)) {
      listeners.get(eventType).forEach(callback => callback(data))
    }
  }

  return {
    connect,
    disconnect,
    send,
    on,
    off,
    isConnected
  }
}
