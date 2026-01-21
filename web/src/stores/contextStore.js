import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../services/api'

export const useContextStore = defineStore('contexts', () => {
  const contexts = ref([])
  const currentContext = ref(null)
  const loading = ref(false)
  const error = ref(null)

  const fetchContexts = async () => {
    loading.value = true
    error.value = null
    try {
      contexts.value = await api.getContexts()
      return contexts.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch contexts'
      console.error('Error fetching contexts:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchProjectContexts = async (projectId) => {
    loading.value = true
    error.value = null
    try {
      contexts.value = await api.getProjectContexts(projectId)
      return contexts.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch project contexts'
      console.error('Error fetching project contexts:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchContext = async (id) => {
    loading.value = true
    error.value = null
    try {
      currentContext.value = await api.getContext(id)
      return currentContext.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch context'
      console.error('Error fetching context:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const createContext = async (data) => {
    loading.value = true
    error.value = null
    try {
      const context = await api.createContext(data)
      contexts.value.unshift(context)
      return context
    } catch (err) {
      error.value = err.message || 'Failed to create context'
      console.error('Error creating context:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateContext = async (id, data) => {
    loading.value = true
    error.value = null
    try {
      const updatedContext = await api.updateContext(id, data)
      const index = contexts.value.findIndex(c => c.id === id)
      if (index !== -1) {
        contexts.value[index] = updatedContext
      }
      if (currentContext.value?.id === id) {
        currentContext.value = updatedContext
      }
      return updatedContext
    } catch (err) {
      error.value = err.message || 'Failed to update context'
      console.error('Error updating context:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const deleteContext = async (id) => {
    loading.value = true
    error.value = null
    try {
      await api.deleteContext(id)
      contexts.value = contexts.value.filter(c => c.id !== id)
      if (currentContext.value?.id === id) {
        currentContext.value = null
      }
    } catch (err) {
      error.value = err.message || 'Failed to delete context'
      console.error('Error deleting context:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const clearError = () => {
    error.value = null
  }

  return {
    contexts,
    currentContext,
    loading,
    error,
    fetchContexts,
    fetchProjectContexts,
    fetchContext,
    createContext,
    updateContext,
    deleteContext,
    clearError
  }
})
