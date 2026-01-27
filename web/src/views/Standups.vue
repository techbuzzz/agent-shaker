<template>
  <div>
    <div class="flex justify-between items-center mb-8">
      <div>
        <h2 class="text-3xl font-bold text-gray-900">Daily Standups</h2>
        <p class="text-gray-600 mt-2">Track team progress and sync on daily activities</p>
      </div>
      <div class="flex gap-3">
        <button 
          @click="handleRefresh" 
          :disabled="isRefreshing" 
          class="bg-gray-100 hover:bg-gray-200 text-gray-700 px-4 py-2 rounded-md font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <svg v-if="isRefreshing" class="animate-spin inline w-4 h-4 mr-2" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <span v-else class="mr-2">🔄</span>
          {{ isRefreshing ? 'Refreshing...' : 'Refresh' }}
        </button>
        <button 
          @click="openModal" 
          class="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-md font-medium transition-colors"
        >
          <span class="mr-2">➕</span> Submit Standup
        </button>
      </div>
    </div>

    <!-- Filters -->
    <div class="bg-white rounded-lg shadow-sm p-4 mb-6">
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-2">Filter by Project</label>
          <select 
            v-model="filters.project_id" 
            @change="applyFilters"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="">All Projects</option>
            <option v-for="project in projects" :key="project.id" :value="project.id">
              {{ project.name }}
            </option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-2">Filter by Agent</label>
          <select 
            v-model="filters.agent_id" 
            @change="applyFilters"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          >
            <option value="">All Agents</option>
            <option v-for="agent in agents" :key="agent.id" :value="agent.id">
              {{ agent.name }} ({{ agent.role }})
            </option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-2">Filter by Date</label>
          <input 
            type="date" 
            v-model="filters.date" 
            @change="applyFilters"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
        </div>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading && !isRefreshing" class="text-center py-12 text-gray-500">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
      Loading standups...
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="p-4 bg-red-50 text-red-600 rounded-md mb-4">
      <div class="flex items-center justify-between">
        <div class="flex items-center">
          <div class="w-5 h-5 bg-red-500 rounded-full flex items-center justify-center mr-3">
            <span class="text-white text-xs">⚠️</span>
          </div>
          <div>
            <p class="font-medium">Failed to load standups</p>
            <p class="text-sm text-red-600">{{ error }}</p>
          </div>
        </div>
        <button @click="handleRefresh" class="bg-gray-100 hover:bg-gray-200 text-gray-700 px-3 py-1 rounded text-sm font-medium transition-colors">
          Try Again
        </button>
      </div>
    </div>

    <!-- Standups List -->
    <div v-else class="space-y-6">
      <div v-for="standup in standups" :key="standup.id" class="bg-white rounded-lg shadow-sm overflow-hidden hover:shadow-md transition-shadow">
        <div class="bg-gradient-to-r from-blue-50 to-purple-50 px-6 py-4 border-b border-gray-200">
          <div class="flex justify-between items-start">
            <div>
              <h3 class="text-xl font-semibold text-gray-900 flex items-center gap-2">
                <span>{{ standup.agent_name }}</span>
                <span class="text-xs bg-blue-100 text-blue-800 px-2 py-1 rounded-full font-medium">
                  {{ standup.agent_role }}
                </span>
              </h3>
              <p class="text-sm text-gray-600 mt-1">
                Team: {{ standup.agent_team }} • {{ formatDate(standup.standup_date) }}
              </p>
            </div>
            <div class="flex gap-2">
              <button 
                @click="editStandup(standup)" 
                class="text-blue-600 hover:text-blue-800 px-3 py-1 rounded text-sm font-medium"
                title="Edit"
              >
                ✏️
              </button>
              <button 
                @click="confirmDelete(standup.id)" 
                class="text-red-600 hover:text-red-800 px-3 py-1 rounded text-sm font-medium"
                title="Delete"
              >
                🗑️
              </button>
            </div>
          </div>
        </div>

        <div class="p-6 space-y-4">
          <!-- Did -->
          <div>
            <h4 class="text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
              <span>📝</span> What I completed yesterday
            </h4>
            <div class="prose prose-sm max-w-none text-gray-600 bg-gray-50 p-3 rounded" v-html="renderMarkdown(standup.did)"></div>
          </div>

          <!-- Doing -->
          <div>
            <h4 class="text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
              <span>🚀</span> What I'm working on today
            </h4>
            <div class="prose prose-sm max-w-none text-gray-600 bg-gray-50 p-3 rounded" v-html="renderMarkdown(standup.doing)"></div>
          </div>

          <!-- Done -->
          <div>
            <h4 class="text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
              <span>✅</span> What I plan to complete
            </h4>
            <div class="prose prose-sm max-w-none text-gray-600 bg-gray-50 p-3 rounded" v-html="renderMarkdown(standup.done)"></div>
          </div>

          <!-- Blockers -->
          <div v-if="standup.blockers">
            <h4 class="text-sm font-semibold text-red-700 mb-2 flex items-center gap-2">
              <span>🚧</span> Blockers
            </h4>
            <div class="prose prose-sm max-w-none text-red-600 bg-red-50 p-3 rounded border border-red-200" v-html="renderMarkdown(standup.blockers)"></div>
          </div>

          <!-- Challenges -->
          <div v-if="standup.challenges">
            <h4 class="text-sm font-semibold text-yellow-700 mb-2 flex items-center gap-2">
              <span>💪</span> Challenges
            </h4>
            <div class="prose prose-sm max-w-none text-yellow-800 bg-yellow-50 p-3 rounded border border-yellow-200" v-html="renderMarkdown(standup.challenges)"></div>
          </div>

          <!-- References -->
          <div v-if="standup.references">
            <h4 class="text-sm font-semibold text-gray-700 mb-2 flex items-center gap-2">
              <span>🔗</span> References
            </h4>
            <div class="prose prose-sm max-w-none text-gray-600 bg-blue-50 p-3 rounded" v-html="renderMarkdown(standup.references)"></div>
          </div>

          <div class="text-xs text-gray-500 pt-2 border-t">
            Created: {{ formatDateTime(standup.created_at) }}
            <span v-if="standup.updated_at !== standup.created_at" class="ml-3">
              Updated: {{ formatDateTime(standup.updated_at) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-if="!loading && standups.length === 0" class="text-center py-12">
      <div class="text-6xl mb-4">📋</div>
      <h3 class="text-xl font-semibold text-gray-900 mb-2">No standups yet</h3>
      <p class="text-gray-600 mb-4">Get started by submitting your first daily standup!</p>
      <button 
        @click="openModal" 
        class="bg-blue-600 hover:bg-blue-700 text-white px-6 py-2 rounded-md font-medium transition-colors inline-flex items-center"
      >
        <span class="mr-2">➕</span> Submit Standup
      </button>
    </div>

    <!-- Modal -->
    <StandupModal 
      :is-open="showModal" 
      :standup="selectedStandup"
      @close="closeModal" 
      @saved="handleStandupSaved" 
    />
  </div>
</template>

<script>
import { onMounted, ref } from 'vue'
import { useStandupStore } from '../stores/standupStore'
import { useAgentStore } from '../stores/agentStore'
import { useProjectStore } from '../stores/projectStore'
import StandupModal from '../components/StandupModal.vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

export default {
  name: 'Standups',
  components: {
    StandupModal
  },
  setup() {
    const standupStore = useStandupStore()
    const agentStore = useAgentStore()
    const projectStore = useProjectStore()

    const showModal = ref(false)
    const selectedStandup = ref(null)
    const isRefreshing = ref(false)
    const agents = ref([])
    const projects = ref([])

    const filters = ref({
      project_id: '',
      agent_id: '',
      date: ''
    })

    const standups = ref([])
    const loading = ref(false)
    const error = ref(null)

    onMounted(async () => {
      await loadData()
    })

    const loadData = async () => {
      loading.value = true
      error.value = null
      try {
        await Promise.all([
          fetchStandups(),
          agentStore.fetchAgents().then(data => { agents.value = data }),
          projectStore.fetchProjects().then(data => { projects.value = data })
        ])
      } catch (err) {
        error.value = err.message || 'Failed to load data'
        console.error('Failed to load data:', err)
      } finally {
        loading.value = false
      }
    }

    const fetchStandups = async () => {
      try {
        const filterParams = {}
        if (filters.value.project_id) filterParams.project_id = filters.value.project_id
        if (filters.value.agent_id) filterParams.agent_id = filters.value.agent_id
        if (filters.value.date) filterParams.date = filters.value.date
        
        standups.value = await standupStore.fetchStandups(filterParams)
      } catch (err) {
        throw err
      }
    }

    const handleRefresh = async () => {
      isRefreshing.value = true
      try {
        await fetchStandups()
      } catch (err) {
        error.value = err.message || 'Failed to refresh'
      } finally {
        isRefreshing.value = false
      }
    }

    const applyFilters = async () => {
      await handleRefresh()
    }

    const openModal = () => {
      selectedStandup.value = null
      showModal.value = true
    }

    const editStandup = (standup) => {
      selectedStandup.value = standup
      showModal.value = true
    }

    const closeModal = () => {
      showModal.value = false
      selectedStandup.value = null
    }

    const handleStandupSaved = async () => {
      await handleRefresh()
    }

    const confirmDelete = async (id) => {
      if (confirm('Are you sure you want to delete this standup?')) {
        try {
          await standupStore.deleteStandup(id)
          standups.value = standups.value.filter(s => s.id !== id)
        } catch (err) {
          alert('Failed to delete standup')
        }
      }
    }

    const renderMarkdown = (text) => {
      if (!text) return ''
      const html = marked.parse(text)
      return DOMPurify.sanitize(html)
    }

    const formatDate = (dateString) => {
      const date = new Date(dateString)
      return date.toLocaleDateString('en-US', { 
        weekday: 'long', 
        year: 'numeric', 
        month: 'long', 
        day: 'numeric' 
      })
    }

    const formatDateTime = (dateString) => {
      const date = new Date(dateString)
      return date.toLocaleString('en-US', { 
        year: 'numeric', 
        month: 'short', 
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      })
    }

    return {
      standups,
      loading,
      error,
      isRefreshing,
      showModal,
      selectedStandup,
      filters,
      agents,
      projects,
      handleRefresh,
      applyFilters,
      openModal,
      editStandup,
      closeModal,
      handleStandupSaved,
      confirmDelete,
      renderMarkdown,
      formatDate,
      formatDateTime
    }
  }
}
</script>

<style scoped>
.prose {
  color: inherit;
}
.prose a {
  color: #2563eb;
  text-decoration: underline;
}
.prose code {
  background-color: #f3f4f6;
  padding: 0.125rem 0.25rem;
  border-radius: 0.25rem;
  font-size: 0.875em;
}
.prose pre {
  background-color: #1f2937;
  color: #f3f4f6;
  padding: 1rem;
  border-radius: 0.5rem;
  overflow-x: auto;
}
.prose ul, .prose ol {
  padding-left: 1.5rem;
}
.prose li {
  margin-top: 0.25rem;
}
</style>
