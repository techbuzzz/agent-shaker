<template>
  <div class="bg-white p-6 rounded-lg shadow-sm hover:shadow-md transition-shadow">
    <div class="flex justify-between items-start mb-3">
      <h4 class="text-lg font-semibold text-gray-900">{{ task.title }}</h4>
      <div class="flex gap-2">
        <span :class="[
          'px-2 py-1 rounded text-xs font-semibold',
          priorityClass
        ]">
          {{ task.priority }}
        </span>
        <span :class="[
          'px-2 py-1 rounded text-xs font-semibold',
          statusClass
        ]">
          {{ formatStatus(task.status) }}
        </span>
      </div>
    </div>
    
    <p class="text-gray-600 mb-4">{{ truncatedDescription }}</p>
    
    <div class="flex justify-between items-center text-sm text-gray-500 pt-4 border-t border-gray-200">
      <span>Agent: {{ agentName }}</span>
      <span>{{ formattedDate }}</span>
    </div>
    <div class="flex justify-between items-center text-sm text-gray-500 pt-4">
      <small>Id: {{ task.id }}</small>
    </div>
    
    <div class="flex justify-end gap-2 mt-4 pt-4 border-t border-gray-100">
      <button
        @click="$emit('reassign', task)"
        class="flex items-center gap-1 px-3 py-1.5 text-sm bg-purple-50 text-purple-600 hover:bg-purple-100 rounded-md transition-colors"
        title="Reassign Task"
      >
        <span>↔️</span>
        <span class="hidden sm:inline">Reassign</span>
      </button>
      <button
        @click="$emit('edit', task)"
        class="flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-50 text-blue-600 hover:bg-blue-100 rounded-md transition-colors"
        title="Edit Task"
      >
        <span>✏️</span>
        <span class="hidden sm:inline">Edit</span>
      </button>
      <button
        @click="$emit('delete', task)"
        class="flex items-center gap-1 px-3 py-1.5 text-sm bg-red-50 text-red-600 hover:bg-red-100 rounded-md transition-colors"
        title="Delete Task"
      >
        <span>🗑️</span>
        <span class="hidden sm:inline">Delete</span>
      </button>
    </div>
  </div>
</template>

<script>
import { computed } from 'vue'

export default {
  name: 'TaskCard',
  props: {
    task: {
      type: Object,
      required: true
    },
    agentName: {
      type: String,
      default: 'Unknown'
    }
  },
  emits: ['edit', 'delete', 'reassign'],
  setup(props) {
    /**
     * Get CSS class for priority badge
     */
    const priorityClass = computed(() => {
      const classes = {
        high: 'bg-red-100 text-red-800',
        medium: 'bg-yellow-100 text-yellow-800',
        low: 'bg-blue-100 text-blue-800'
      }
      return classes[props.task.priority] || 'bg-gray-100 text-gray-800'
    })

    /**
     * Get CSS class for status badge
     */
    const statusClass = computed(() => {
      const classes = {
        done: 'bg-green-100 text-green-800',
        in_progress: 'bg-blue-100 text-blue-800',
        pending: 'bg-gray-100 text-gray-800',
        blocked: 'bg-red-100 text-red-800'
      }
      return classes[props.task.status] || 'bg-gray-100 text-gray-800'
    })

    /**
     * Format status for display
     * @param {string} status - Task status
     * @returns {string} Formatted status
     */
    const formatStatus = (status) => {
      return status.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())
    }

    /**
     * Format date for display
     */
    const formattedDate = computed(() => {
      return new Date(props.task.created_at).toLocaleString()
    })

    /**
     * Truncate description to 200 characters
     */
    const truncatedDescription = computed(() => {
      const desc = props.task.description || ''
      return desc.length > 200 ? desc.substring(0, 200) + '...' : desc
    })

    return {
      priorityClass,
      statusClass,
      formatStatus,
      formattedDate,
      truncatedDescription
    }
  }
}
</script>
