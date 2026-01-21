import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../services/api'

export const useTaskStore = defineStore('tasks', () => {
  const tasks = ref([])
  const currentTask = ref(null)
  const loading = ref(false)
  const error = ref(null)

  const fetchTasks = async (filters = {}) => {
    loading.value = true
    error.value = null
    try {
      tasks.value = await api.getTasks(filters)
      return tasks.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch tasks'
      console.error('Error fetching tasks:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchTask = async (id) => {
    loading.value = true
    error.value = null
    try {
      currentTask.value = await api.getTask(id)
      return currentTask.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch task'
      console.error('Error fetching task:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchProjectTasks = async (projectId) => {
    loading.value = true
    error.value = null
    try {
      tasks.value = await api.getProjectTasks(projectId)
      return tasks.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch project tasks'
      console.error('Error fetching project tasks:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchAgentTasks = async (agentId) => {
    loading.value = true
    error.value = null
    try {
      tasks.value = await api.getAgentTasks(agentId)
      return tasks.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch agent tasks'
      console.error('Error fetching agent tasks:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const createTask = async (data) => {
    loading.value = true
    error.value = null
    try {
      const task = await api.createTask(data)
      tasks.value.unshift(task)
      return task
    } catch (err) {
      error.value = err.message || 'Failed to create task'
      console.error('Error creating task:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateTask = async (id, data) => {
    loading.value = true
    error.value = null
    try {
      const updatedTask = await api.updateTask(id, data)
      const index = tasks.value.findIndex(t => t.id === id)
      if (index !== -1) {
        tasks.value[index] = updatedTask
      }
      if (currentTask.value?.id === id) {
        currentTask.value = updatedTask
      }
      return updatedTask
    } catch (err) {
      error.value = err.message || 'Failed to update task'
      console.error('Error updating task:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateTaskStatus = async (id, status) => {
    error.value = null
    try {
      const updatedTask = await api.updateTaskStatus(id, status)
      const task = tasks.value.find(t => t.id === id)
      if (task) {
        task.status = status
        task.updated_at = updatedTask.updated_at
      }
      if (currentTask.value?.id === id) {
        currentTask.value.status = status
        currentTask.value.updated_at = updatedTask.updated_at
      }
      return updatedTask
    } catch (err) {
      error.value = err.message || 'Failed to update task status'
      console.error('Error updating task status:', err)
      throw err
    }
  }

  const deleteTask = async (id) => {
    loading.value = true
    error.value = null
    try {
      await api.deleteTask(id)
      tasks.value = tasks.value.filter(t => t.id !== id)
      if (currentTask.value?.id === id) {
        currentTask.value = null
      }
    } catch (err) {
      error.value = err.message || 'Failed to delete task'
      console.error('Error deleting task:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const clearError = () => {
    error.value = null
  }

  return {
    tasks,
    currentTask,
    loading,
    error,
    fetchTasks,
    fetchTask,
    fetchProjectTasks,
    fetchAgentTasks,
    createTask,
    updateTask,
    updateTaskStatus,
    deleteTask,
    clearError
  }
})
