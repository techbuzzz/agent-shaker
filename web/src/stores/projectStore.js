import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../services/api'

export const useProjectStore = defineStore('projects', () => {
  const projects = ref([])
  const currentProject = ref(null)
  const loading = ref(false)
  const error = ref(null)

  const fetchProjects = async () => {
    loading.value = true
    error.value = null
    try {
      projects.value = await api.getProjects()
      return projects.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch projects'
      console.error('Error fetching projects:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const fetchProject = async (id) => {
    loading.value = true
    error.value = null
    try {
      currentProject.value = await api.getProject(id)
      return currentProject.value
    } catch (err) {
      error.value = err.message || 'Failed to fetch project'
      console.error('Error fetching project:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const createProject = async (data) => {
    loading.value = true
    error.value = null
    try {
      const project = await api.createProject(data)
      projects.value.unshift(project)
      return project
    } catch (err) {
      error.value = err.message || 'Failed to create project'
      console.error('Error creating project:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateProject = async (id, data) => {
    loading.value = true
    error.value = null
    try {
      const updatedProject = await api.updateProject(id, data)
      const index = projects.value.findIndex(p => p.id === id)
      if (index !== -1) {
        projects.value[index] = updatedProject
      }
      if (currentProject.value?.id === id) {
        currentProject.value = updatedProject
      }
      return updatedProject
    } catch (err) {
      error.value = err.message || 'Failed to update project'
      console.error('Error updating project:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const deleteProject = async (id) => {
    loading.value = true
    error.value = null
    try {
      await api.deleteProject(id)
      projects.value = projects.value.filter(p => p.id !== id)
      if (currentProject.value?.id === id) {
        currentProject.value = null
      }
    } catch (err) {
      error.value = err.message || 'Failed to delete project'
      console.error('Error deleting project:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const updateProjectStatus = async (id, status) => {
    loading.value = true
    error.value = null
    try {
      const updatedProject = await api.updateProjectStatus(id, status)
      const index = projects.value.findIndex(p => p.id === id)
      if (index !== -1) {
        projects.value[index] = updatedProject
      }
      if (currentProject.value?.id === id) {
        currentProject.value = updatedProject
      }
      return updatedProject
    } catch (err) {
      error.value = err.message || 'Failed to update project status'
      console.error('Error updating project status:', err)
      throw err
    } finally {
      loading.value = false
    }
  }

  const clearError = () => {
    error.value = null
  }

  return {
    projects,
    currentProject,
    loading,
    error,
    fetchProjects,
    fetchProject,
    createProject,
    updateProject,
    deleteProject,
    updateProjectStatus,
    clearError
  }
})
