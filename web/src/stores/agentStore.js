import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../services/api'

export const useAgentStore = defineStore('agents', () => {
  const agents = ref([])
  const currentAgent = ref(null)
  const loading = ref(false)
  const error = ref(null)

  const fetchAgents = async (projectId = null) => {
    loading.value = true
    error.value = null
    try {
      agents.value = await api.getAgents(projectId)
      return agents.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch agents'
      console.error('Error fetching agents:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchAgent = async (id) => {
    loading.value = true
    error.value = null
    try {
      currentAgent.value = await api.getAgent(id)
      return currentAgent.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch agent'
      console.error('Error fetching agent:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchProjectAgents = async (projectId) => {
    loading.value = true
    error.value = null
    try {
      agents.value = await api.getProjectAgents(projectId)
      return agents.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch project agents'
      console.error('Error fetching project agents:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const createAgent = async (data) => {
    loading.value = true
    error.value = null
    try {
      const agent = await api.createAgent(data)
      agents.value.unshift(agent)
      return agent
    } catch (err) {
      error.value = err.message || 'Failed to create agent'
      console.error('Error creating agent:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateAgent = async (id, data) => {
    loading.value = true
    error.value = null
    try {
      const updatedAgent = await api.updateAgent(id, data)
      const index = agents.value.findIndex(a => a.id === id)
      if (index !== -1) {
        agents.value[index] = updatedAgent
      }
      if (currentAgent.value?.id === id) {
        currentAgent.value = updatedAgent
      }
      return updatedAgent
    } catch (err) {
      error.value = err.message || 'Failed to update agent'
      console.error('Error updating agent:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateAgentStatus = async (id, status) => {
    error.value = null
    try {
      const updatedAgent = await api.updateAgentStatus(id, status)
      const agent = agents.value.find(a => a.id === id)
      if (agent) {
        agent.status = status
        agent.last_seen = updatedAgent.last_seen
      }
      if (currentAgent.value?.id === id) {
        currentAgent.value.status = status
        currentAgent.value.last_seen = updatedAgent.last_seen
      }
      return updatedAgent
    } catch (err) {
      error.value = err.message || 'Failed to update agent status'
      console.error('Error updating agent status:', err)
      throw err
    }
  }

  const deleteAgent = async (id) => {
    loading.value = true
    error.value = null
    try {
      await api.deleteAgent(id)
      agents.value = agents.value.filter(a => a.id !== id)
      if (currentAgent.value?.id === id) {
        currentAgent.value = null
      }
    } catch (err) {
      error.value = err.message || 'Failed to delete agent'
      console.error('Error deleting agent:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const clearError = () => {
    error.value = null
  }

  return {
    agents,
    currentAgent,
    loading,
    error,
    fetchAgents,
    fetchAgent,
    fetchProjectAgents,
    createAgent,
    updateAgent,
    updateAgentStatus,
    deleteAgent,
    clearError
  }
})
