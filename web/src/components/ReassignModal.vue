<template>
  <div v-if="show" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="bg-white p-6 rounded-lg max-w-md w-full mx-4">
      <h3 class="text-xl font-semibold mb-4">Reassign Task</h3>
      <div class="mb-6">
        <p class="text-sm text-gray-600 mb-2"><strong>Task:</strong></p>
        <p class="text-gray-900">{{ task?.title }}</p>
      </div>
      
      <form @submit.prevent="handleSubmit">
        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-2">Assign to Agent</label>
          <select 
            v-model="selectedAgentId" 
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" 
            required
          >
            <option value="">Select an agent</option>
            <option v-for="agent in availableAgents" :key="agent.id" :value="agent.id">
              {{ agent.name }}
            </option>
          </select>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button 
            type="button" 
            @click="$emit('close')" 
            class="bg-gray-200 hover:bg-gray-300 text-gray-800 px-4 py-2 rounded-md font-medium transition-colors"
          >
            Cancel
          </button>
          <button 
            type="submit" 
            :disabled="!selectedAgentId || isSubmitting"
            class="bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 text-white px-4 py-2 rounded-md font-medium transition-colors"
          >
            {{ isSubmitting ? 'Reassigning...' : 'Reassign Task' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script>
import { ref, watch, computed } from 'vue'

export default {
  name: 'ReassignModal',
  props: {
    show: {
      type: Boolean,
      required: true
    },
    task: {
      type: Object,
      default: null
    },
    agents: {
      type: Array,
      required: true
    }
  },
  emits: ['close', 'reassign'],
  setup(props, { emit }) {
    const selectedAgentId = ref('')
    const isSubmitting = ref(false)

    // Filter out the currently assigned agent
    const availableAgents = computed(() => {
      return props.agents.filter(agent => {
        const assignedToId = props.task?.assigned_to
        return !assignedToId || agent.id !== assignedToId
      })
    })

    watch(() => props.show, (newVal) => {
      if (newVal) {
        selectedAgentId.value = ''
      }
    })

    const handleSubmit = async () => {
      if (!selectedAgentId.value || !props.task) {
        return
      }

      isSubmitting.value = true
      try {
        emit('reassign', {
          taskId: props.task.id,
          agentId: selectedAgentId.value
        })
        selectedAgentId.value = ''
      } finally {
        isSubmitting.value = false
      }
    }

    return {
      selectedAgentId,
      isSubmitting,
      availableAgents,
      handleSubmit
    }
  }
}
</script>
