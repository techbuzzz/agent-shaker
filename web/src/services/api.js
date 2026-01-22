import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    console.error('API Error:', error)
    return Promise.reject(error)
  }
)

export default {
  // Projects
  getProjects() {
    return api.get('/projects')
  },
  getProject(id) {
    return api.get(`/projects/${id}`)
  },
  createProject(data) {
    return api.post('/projects', data)
  },
  updateProject(id, data) {
    return api.put(`/projects/${id}`, data)
  },
  deleteProject(id) {
    return api.delete(`/projects/${id}`)
  },
  updateProjectStatus(id, status) {
    return api.put(`/projects/${id}/status`, { status })
  },

  // Agents
  getAgents(projectId = null) {
    const params = projectId ? { project_id: projectId } : {}
    return api.get('/agents', { params })
  },
  getAgent(id) {
    return api.get(`/agents/${id}`)
  },
  getProjectAgents(projectId) {
    return api.get('/agents', { params: { project_id: projectId } })
  },
  createAgent(data) {
    return api.post('/agents', data)
  },
  updateAgent(id, data) {
    return api.put(`/agents/${id}`, data)
  },
  updateAgentStatus(id, status) {
    return api.put(`/agents/${id}/status`, { status })
  },
  deleteAgent(id) {
    return api.delete(`/agents/${id}`)
  },

  // Tasks
  getTasks(params = {}) {
    return api.get('/tasks', { params })
  },
  getTask(id) {
    return api.get(`/tasks/${id}`)
  },
  getProjectTasks(projectId) {
    return api.get('/tasks', { params: { project_id: projectId } })
  },
  getAgentTasks(agentId) {
    return api.get('/tasks', { params: { agent_id: agentId } })
  },
  createTask(data) {
    return api.post('/tasks', data)
  },
  updateTask(id, data) {
    return api.put(`/tasks/${id}`, data)
  },
  updateTaskStatus(id, status) {
    return api.put(`/tasks/${id}/status`, { status })
  },
  deleteTask(id) {
    return api.delete(`/tasks/${id}`)
  },

  // Context/Documentation
  createDocumentation(data) {
    return api.post('/documentation', data)
  },
  getTaskDocumentation(taskId) {
    return api.get(`/tasks/${taskId}/documentation`)
  },

  // Contexts
  getContexts() {
    return api.get('/contexts')
  },
  getContext(id) {
    return api.get(`/contexts/${id}`)
  },
  getProjectContexts(projectId) {
    return api.get('/contexts', { params: { project_id: projectId } })
  },
  createContext(data) {
    return api.post('/contexts', data)
  },
  updateContext(id, data) {
    return api.put(`/contexts/${id}`, data)
  },
  deleteContext(id) {
    return api.delete(`/contexts/${id}`)
  },

  // Dashboard
  getDashboardStats() {
    return api.get('/dashboard')
  },

  // Health
  checkHealth() {
    return axios.get('/health')
  },
}
