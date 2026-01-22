<template>
  <div v-if="show" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="bg-white p-6 rounded-lg max-w-md w-full mx-4">
      <h3 class="text-xl font-semibold mb-6">{{ isEdit ? 'Edit Task' : 'Create New Task' }}</h3>
      <form @submit.prevent="handleSubmit">
        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-2">Task Title</label>
          <input 
            v-model="formData.title" 
            type="text" 
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" 
            required 
          />
        </div>
        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-2">Description</label>
          <textarea 
            v-model="formData.description" 
            rows="4" 
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          ></textarea>
        </div>
        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-2">Assign to Agent</label>
          <select 
            v-model="formData.agent_id" 
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500" 
            required
          >
            <option value="">Select an agent</option>
            <option v-for="agent in agents" :key="agent.id" :value="agent.id">
              {{ agent.name }}
            </option>
          </select>
        </div>
        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-2">Priority</label>
          <select 
            v-model="formData.priority" 
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="low">Low</option>
            <option value="medium">Medium</option>
            <option value="high">High</option>
          </select>
        </div>
        <div v-if="isEdit" class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-2">Status</label>
          <select 
            v-model="formData.status" 
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="pending">Pending</option>
            <option value="in_progress">In Progress</option>
            <option value="done">Done</option>
            <option value="blocked">Blocked</option>
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
            class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors"
          >
            {{ isEdit ? 'Update' : 'Create' }} Task
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script>
import { ref, watch } from 'vue'

export default {
  name: 'TaskModal',
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
  emits: ['close', 'save'],
  setup(props, { emit }) {
    const isEdit = ref(false)
    const formData = ref({
      title: '',
      description: '',
      agent_id: '',
      priority: 'medium',
      status: 'pending'
    })

    watch(() => props.task, (newTask) => {
      if (newTask) {
        isEdit.value = true
        formData.value = {
          title: newTask.title,
          description: newTask.description || '',
          agent_id: newTask.assigned_to || newTask.agent_id || '',
          priority: newTask.priority,
          status: newTask.status
        }
      } else {
        isEdit.value = false
        formData.value = {
          title: '',
          description: '',
          agent_id: '',
          priority: 'medium',
          status: 'pending'
        }
      }
    }, { immediate: true })

    const handleSubmit = () => {
      if (!formData.value.title.trim() || !formData.value.agent_id) {
        alert('Please fill in all required fields')
        return
      }
      
      // Map agent_id to assigned_to for API
      const taskData = {
        title: formData.value.title,
        description: formData.value.description,
        assigned_to: formData.value.agent_id,
        priority: formData.value.priority,
        status: formData.value.status
      }
      
      emit('save', taskData)
    }

    return {
      isEdit,
      formData,
      handleSubmit
    }
  }
}
</script>
