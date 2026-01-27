<template>
  <div v-if="isOpen" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
    <div class="bg-white rounded-lg shadow-xl max-w-3xl w-full max-h-[90vh] overflow-y-auto">
      <div class="sticky top-0 bg-white border-b px-6 py-4 flex justify-between items-center">
        <h2 class="text-2xl font-bold text-gray-900">{{ isEditing ? 'Update Daily Standup' : 'Submit Daily Standup' }}</h2>
        <button @click="handleClose" class="text-gray-400 hover:text-gray-600 text-2xl">&times;</button>
      </div>

      <form @submit.prevent="handleSubmit" class="p-6 space-y-6">
        <div class="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-4">
          <p class="text-sm text-blue-800">
            <strong>💡 Tip:</strong> Use Markdown formatting in your responses! You can add links, lists, code snippets, and more.
          </p>
        </div>

        <!-- Agent Selection -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2">
            Agent <span class="text-red-500">*</span>
          </label>
          <select 
            v-model="formData.agent_id" 
            required 
            class="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            :disabled="isEditing"
          >
            <option value="">Select an agent</option>
            <option v-for="agent in agents" :key="agent.id" :value="agent.id">
              {{ agent.name }} - {{ agent.role }} ({{ agent.team }})
            </option>
          </select>
        </div>

        <!-- Project Selection -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2">
            Project <span class="text-red-500">*</span>
          </label>
          <select 
            v-model="formData.project_id" 
            required 
            class="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            :disabled="isEditing"
          >
            <option value="">Select a project</option>
            <option v-for="project in projects" :key="project.id" :value="project.id">
              {{ project.name }}
            </option>
          </select>
        </div>

        <!-- Standup Date -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2">
            Date <span class="text-red-500">*</span>
          </label>
          <input 
            type="date" 
            v-model="formData.standup_date" 
            required 
            class="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
            :disabled="isEditing"
          />
        </div>

        <!-- Did (Yesterday) -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2">
            📝 What did I complete yesterday? <span class="text-red-500">*</span>
          </label>
          <textarea 
            v-model="formData.did" 
            required 
            rows="4" 
            placeholder="List completed tasks, features, or accomplishments..."
            class="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          ></textarea>
        </div>

        <!-- Doing (Today) -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2">
            🚀 What am I working on today? <span class="text-red-500">*</span>
          </label>
          <textarea 
            v-model="formData.doing" 
            required 
            rows="4" 
            placeholder="Current focus and planned tasks..."
            class="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          ></textarea>
        </div>

        <!-- Done (Plan to Complete) -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2">
            ✅ What do I plan to complete? <span class="text-red-500">*</span>
          </label>
          <textarea 
            v-model="formData.done" 
            required 
            rows="4" 
            placeholder="Expected deliverables by end of day..."
            class="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          ></textarea>
        </div>

        <!-- Blockers -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2">
            🚧 Blockers
          </label>
          <textarea 
            v-model="formData.blockers" 
            rows="3" 
            placeholder="Any obstacles preventing progress..."
            class="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          ></textarea>
        </div>

        <!-- Challenges -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2">
            💪 Challenges
          </label>
          <textarea 
            v-model="formData.challenges" 
            rows="3" 
            placeholder="Current technical or organizational challenges..."
            class="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          ></textarea>
        </div>

        <!-- References -->
        <div>
          <label class="block text-sm font-semibold text-gray-700 mb-2">
            🔗 References
          </label>
          <textarea 
            v-model="formData.references" 
            rows="3" 
            placeholder="Links to PRs, docs, tickets, or relevant resources..."
            class="w-full px-4 py-2 border border-gray-300 rounded-md focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          ></textarea>
        </div>

        <div class="flex justify-end gap-3 pt-4 border-t">
          <button 
            type="button" 
            @click="handleClose" 
            class="px-6 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50 font-medium"
          >
            Cancel
          </button>
          <button 
            type="submit" 
            :disabled="loading"
            class="px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 font-medium disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ loading ? 'Saving...' : (isEditing ? 'Update' : 'Submit') }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script>
import { ref, watch } from 'vue'
import { useAgentStore } from '../stores/agentStore'
import { useProjectStore } from '../stores/projectStore'
import { useStandupStore } from '../stores/standupStore'

export default {
  name: 'StandupModal',
  props: {
    isOpen: {
      type: Boolean,
      default: false
    },
    standup: {
      type: Object,
      default: null
    }
  },
  emits: ['close', 'saved'],
  setup(props, { emit }) {
    const agentStore = useAgentStore()
    const projectStore = useProjectStore()
    const standupStore = useStandupStore()

    const loading = ref(false)
    const isEditing = ref(false)

    const formData = ref({
      agent_id: '',
      project_id: '',
      standup_date: new Date().toISOString().split('T')[0],
      did: '',
      doing: '',
      done: '',
      blockers: '',
      challenges: '',
      references: ''
    })

    const agents = ref([])
    const projects = ref([])

    // Load agents and projects
    const loadData = async () => {
      try {
        agents.value = await agentStore.fetchAgents()
        projects.value = await projectStore.fetchProjects()
      } catch (error) {
        console.error('Failed to load data:', error)
      }
    }

    // Initialize form when modal opens
    watch(() => props.isOpen, (newVal) => {
      if (newVal) {
        loadData()
        if (props.standup) {
          isEditing.value = true
          formData.value = {
            agent_id: props.standup.agent_id,
            project_id: props.standup.project_id,
            standup_date: props.standup.standup_date.split('T')[0],
            did: props.standup.did,
            doing: props.standup.doing,
            done: props.standup.done,
            blockers: props.standup.blockers || '',
            challenges: props.standup.challenges || '',
            references: props.standup.references || ''
          }
        } else {
          isEditing.value = false
          formData.value = {
            agent_id: '',
            project_id: '',
            standup_date: new Date().toISOString().split('T')[0],
            did: '',
            doing: '',
            done: '',
            blockers: '',
            challenges: '',
            references: ''
          }
        }
      }
    })

    const handleSubmit = async () => {
      loading.value = true
      try {
        if (isEditing.value) {
          await standupStore.updateStandup(props.standup.id, formData.value)
        } else {
          await standupStore.createStandup(formData.value)
        }
        emit('saved')
        handleClose()
      } catch (error) {
        console.error('Failed to save standup:', error)
        alert('Failed to save standup. Please try again.')
      } finally {
        loading.value = false
      }
    }

    const handleClose = () => {
      emit('close')
    }

    return {
      formData,
      loading,
      isEditing,
      agents,
      projects,
      handleSubmit,
      handleClose
    }
  }
}
</script>
