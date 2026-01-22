<template>
  <Teleport to="body">
    <div 
      v-if="isOpen" 
      class="fixed inset-0 z-50 flex items-center justify-center p-4"
      @click.self="handleClose"
    >
      <!-- Backdrop -->
      <div class="absolute inset-0 bg-black/50 backdrop-blur-sm"></div>
      
      <!-- Modal -->
      <div class="relative bg-white rounded-xl shadow-2xl w-full max-w-md transform transition-all">
        <!-- Header -->
        <div class="flex items-center justify-between p-4 border-b border-gray-200">
          <div class="flex items-center gap-3">
            <span class="text-2xl">🔗</span>
            <h2 class="text-lg font-semibold text-gray-900">Enter Server URL</h2>
          </div>
          <button 
            @click="handleClose"
            class="text-gray-400 hover:text-gray-600 transition-colors"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Body -->
        <div class="p-4">
          <label class="block text-sm text-gray-600 mb-2">
            URL of the MCP server (e.g., http://localhost:8080)
          </label>
          <input
            ref="urlInput"
            v-model="inputUrl"
            type="url"
            placeholder="http://localhost:8080"
            class="w-full px-4 py-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent text-gray-800 placeholder-gray-400"
            @keydown.enter="handleConnect"
            @keydown.escape="handleClose"
          />
          
          <!-- Connection status -->
          <div v-if="connectionStatus" class="mt-3">
            <div 
              :class="[
                'flex items-center gap-2 px-3 py-2 rounded-lg text-sm',
                connectionStatus === 'success' ? 'bg-green-50 text-green-700' : '',
                connectionStatus === 'error' ? 'bg-red-50 text-red-700' : '',
                connectionStatus === 'connecting' ? 'bg-blue-50 text-blue-700' : ''
              ]"
            >
              <span v-if="connectionStatus === 'connecting'" class="animate-spin">⏳</span>
              <span v-else-if="connectionStatus === 'success'">✅</span>
              <span v-else-if="connectionStatus === 'error'">❌</span>
              <span>{{ statusMessage }}</span>
            </div>
          </div>

          <p class="text-xs text-gray-500 mt-3">
            Press 'Enter' to confirm your input or 'Escape' to cancel
          </p>
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-between p-4 border-t border-gray-200 bg-gray-50 rounded-b-xl">
          <button
            @click="resetToDefault"
            class="text-sm text-gray-500 hover:text-gray-700 transition-colors"
          >
            Reset to default
          </button>
          <div class="flex gap-2">
            <button
              @click="handleClose"
              class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button
              @click="handleConnect"
              :disabled="isConnecting || !inputUrl"
              :class="[
                'px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors flex items-center gap-2',
                isConnecting || !inputUrl 
                  ? 'bg-blue-400 cursor-not-allowed' 
                  : 'bg-blue-600 hover:bg-blue-700'
              ]"
            >
              <span v-if="isConnecting" class="animate-spin">⏳</span>
              {{ isConnecting ? 'Connecting...' : 'Connect' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { useSettingsStore } from '@/stores/settingsStore'

const props = defineProps({
  isOpen: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['close', 'connected'])

const settingsStore = useSettingsStore()

const inputUrl = ref(settingsStore.serverUrl)
const urlInput = ref(null)
const connectionStatus = ref(null)
const statusMessage = ref('')
const isConnecting = ref(false)

watch(() => props.isOpen, (newValue) => {
  if (newValue) {
    inputUrl.value = settingsStore.serverUrl
    connectionStatus.value = null
    statusMessage.value = ''
    nextTick(() => {
      urlInput.value?.focus()
      urlInput.value?.select()
    })
  }
})

const handleConnect = async () => {
  if (!inputUrl.value || isConnecting.value) return
  
  isConnecting.value = true
  connectionStatus.value = 'connecting'
  statusMessage.value = 'Testing connection...'
  
  settingsStore.setServerUrl(inputUrl.value)
  const result = await settingsStore.testConnection()
  
  isConnecting.value = false
  
  if (result.success) {
    connectionStatus.value = 'success'
    statusMessage.value = 'Connected successfully!'
    setTimeout(() => {
      emit('connected', inputUrl.value)
      emit('close')
    }, 1000)
  } else {
    connectionStatus.value = 'error'
    statusMessage.value = result.error || 'Failed to connect to server'
  }
}

const handleClose = () => {
  emit('close')
}

const resetToDefault = () => {
  inputUrl.value = 'http://localhost:8080'
  connectionStatus.value = null
  statusMessage.value = ''
}
</script>
