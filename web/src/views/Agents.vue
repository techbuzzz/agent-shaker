<template>
  <div>
    <div class="flex justify-between items-center mb-8">
      <h2 class="text-3xl font-bold text-gray-900">All Agents</h2>
      <button @click="handleRefresh" :disabled="isRefreshing" class="bg-gray-100 hover:bg-gray-200 text-gray-700 px-4 py-2 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
        <svg v-if="isRefreshing" class="animate-spin inline w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <span v-else class="mr-2">🔄</span>
        {{ isRefreshing ? 'Refreshing...' : 'Refresh' }}
      </button>
    </div>

    <div v-if="loading && !isRefreshing" class="text-center py-12 text-gray-500">Loading agents...</div>
    <div v-else-if="error" class="p-4 bg-red-50 text-red-600 rounded-md mb-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center">
          <div class="w-5 h-5 bg-red-500 rounded-full flex items-center justify-center mr-3">
            <span class="text-white text-xs">⚠️</span>
          </div>
          <div>
            <p class="font-medium">Failed to load agents</p>
            <p class="text-sm text-red-600">{{ error }}</p>
          </div>
        </div>
        <button @click="handleRefresh" class="bg-gray-100 hover:bg-gray-200 text-gray-700 px-3 py-1 rounded text-sm font-medium transition-colors">
          Try Again
        </button>
      </div>
    </div>
    
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      <div v-for="agent in agents" :key="agent.id" class="bg-white p-6 rounded-lg shadow-sm">
        <div class="flex justify-between items-start mb-4">
          <h3 class="text-xl font-semibold text-gray-900">{{ agent.name }}</h3>
          <span :class="['px-3 py-1 rounded-full text-sm font-semibold', agent.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800' ]">{{ agent.status }}</span>
        </div>
        <div class="space-y-2">
          <p class="text-gray-600"><strong class="font-medium text-gray-900">Role:</strong> <span :class="['inline-block px-2 py-1 rounded text-xs font-semibold', agent.role === 'frontend' ? 'bg-blue-100 text-blue-800' : 'bg-pink-100 text-pink-800' ]">{{ agent.role }}</span></p>
          <p class="text-gray-600"><strong class="font-medium text-gray-900">Team:</strong> {{ agent.team }}</p>
          <p class="text-gray-600"><strong class="font-medium text-gray-900">Project ID:</strong> {{ agent.project_id }}</p>
          <p class="text-gray-600"><strong class="font-medium text-gray-900">Last Seen:</strong> {{ formatDate(agent.last_seen) }}</p>
          <p class="text-gray-600"><strong class="font-medium text-gray-900">Created:</strong> {{ formatDate(agent.created_at) }}</p>
        </div>
      </div>
    </div>

    <div v-if="!loading && agents.length === 0" class="text-center py-12">
      <h3 class="text-xl font-semibold text-gray-900 mb-2">No agents registered</h3>
      <p class="text-gray-600">Agents will appear here when they are registered to projects</p>
    </div>
  </div>
</template>

<script>
import { onMounted, ref } from 'vue'
import { useAgentStore } from '../stores/agentStore'

export default {
  name: 'Agents',
  setup() {
    const agentStore = useAgentStore()
    const isRefreshing = ref(false)

    onMounted(() => {
      fetchAgentsData()
    })

    const fetchAgentsData = async () => {
      try {
        await agentStore.fetchAgents()
      } catch (error) {
        console.error('Failed to load agents:', error)
        // Error is already handled in the store
        // Auto-retry once after 2 seconds if it's the first load and no data exists
        if (agentStore.agents.length === 0) {
          setTimeout(() => {
            if (agentStore.error && agentStore.agents.length === 0) {
              console.log('Auto-retrying to load agents...')
              fetchAgentsData()
            }
          }, 2000)
        }
      }
    }

    const handleRefresh = async () => {
      isRefreshing.value = true
      try {
        await agentStore.fetchAgents()
      } catch (error) {
        console.error('Failed to refresh agents:', error)
      } finally {
        isRefreshing.value = false
      }
    }

    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleString()
    }

    return {
      agents: agentStore.agents,
      loading: agentStore.loading,
      error: agentStore.error,
      handleRefresh,
      isRefreshing,
      formatDate
    }
  }
}
</script>
