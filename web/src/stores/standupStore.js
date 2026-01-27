import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../services/api'

export const useStandupStore = defineStore('standups', () => {
  const standups = ref([])
  const currentStandup = ref(null)
  const loading = ref(false)
  const error = ref(null)

  const fetchStandups = async (filters = {}) => {
    loading.value = true
    error.value = null
    try {
      standups.value = await api.getStandups(filters)
      return standups.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch standups'
      console.error('Error fetching standups:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchStandup = async (id) => {
    loading.value = true
    error.value = null
    try {
      currentStandup.value = await api.getStandup(id)
      return currentStandup.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch standup'
      console.error('Error fetching standup:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const createStandup = async (data) => {
    loading.value = true
    error.value = null
    try {
      const standup = await api.createStandup(data)
      standups.value.unshift(standup)
      return standup
    } catch (err) {
      error.value = err.message || 'Failed to create standup'
      console.error('Error creating standup:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateStandup = async (id, data) => {
    loading.value = true
    error.value = null
    try {
      const updatedStandup = await api.updateStandup(id, data)
      const index = standups.value.findIndex(s => s.id === id)
      if (index !== -1) {
        standups.value[index] = { ...standups.value[index], ...updatedStandup }
      }
      return updatedStandup
    } catch (err) {
      error.value = err.message || 'Failed to update standup'
      console.error('Error updating standup:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const deleteStandup = async (id) => {
    loading.value = true
    error.value = null
    try {
      await api.deleteStandup(id)
      standups.value = standups.value.filter(s => s.id !== id)
    } catch (err) {
      error.value = err.message || 'Failed to delete standup'
      console.error('Error deleting standup:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const recordHeartbeat = async (data) => {
    try {
      return await api.recordHeartbeat(data)
    } catch (err) {
      console.error('Error recording heartbeat:', err)
      throw err
    }
  }

  const getAgentHeartbeats = async (agentId, limit = 50) => {
    try {
      return await api.getAgentHeartbeats(agentId, limit)
    } catch (err) {
      console.error('Error fetching heartbeats:', err)
      throw err
    }
  }

  const clearError = () => {
    error.value = null
  }

  return {
    standups,
    currentStandup,
    loading,
    error,
    fetchStandups,
    fetchStandup,
    createStandup,
    updateStandup,
    deleteStandup,
    recordHeartbeat,
    getAgentHeartbeats,
    clearError
  }
})
