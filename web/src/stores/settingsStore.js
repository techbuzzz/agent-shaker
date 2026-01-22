import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { updateApiBaseUrl } from '../services/api'

const STORAGE_KEY = 'mcp-server-url'
const DEFAULT_URL = 'http://localhost:8080'

export const useSettingsStore = defineStore('settings', () => {
  const serverUrl = ref(localStorage.getItem(STORAGE_KEY) || DEFAULT_URL)
  const isConnected = ref(false)
  const connectionError = ref(null)
  const isConnecting = ref(false)

  const apiBaseUrl = computed(() => {
    try {
      const url = new URL(serverUrl.value)
      return `${url.protocol}//${url.host}/api`
    } catch {
      return `${serverUrl.value}/api`
    }
  })

  const wsBaseUrl = computed(() => {
    try {
      const url = new URL(serverUrl.value)
      const wsProtocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
      return `${wsProtocol}//${url.host}/ws`
    } catch {
      return serverUrl.value.replace(/^http/, 'ws') + '/ws'
    }
  })

  const setServerUrl = (url) => {
    serverUrl.value = url
    localStorage.setItem(STORAGE_KEY, url)
    updateApiBaseUrl(url)
    isConnected.value = false
    connectionError.value = null
  }

  const testConnection = async () => {
    isConnecting.value = true
    connectionError.value = null
    
    try {
      const response = await fetch(`${serverUrl.value}/health`, {
        method: 'GET',
        headers: {
          'Accept': 'text/plain,application/json'
        }
      })
      
      if (response.ok) {
        isConnected.value = true
        connectionError.value = null
        return { success: true }
      } else {
        throw new Error(`Server returned ${response.status}`)
      }
    } catch (error) {
      isConnected.value = false
      connectionError.value = error.message || 'Failed to connect to server'
      return { success: false, error: connectionError.value }
    } finally {
      isConnecting.value = false
    }
  }

  const resetToDefault = () => {
    setServerUrl(DEFAULT_URL)
  }

  return {
    serverUrl,
    apiBaseUrl,
    wsBaseUrl,
    isConnected,
    connectionError,
    isConnecting,
    setServerUrl,
    testConnection,
    resetToDefault
  }
})
