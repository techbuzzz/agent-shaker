<template>
  <div class="project-detail">
    <div class="container">
      <div v-if="loading" class="text-center py-12 text-gray-500">Loading project...</div>
      <div v-else-if="error" class="p-4 bg-red-50 text-red-600 rounded-md mb-4">{{ error }}</div>
      
      <div v-else-if="project">
        <div class="flex justify-between items-start mb-8">
          <div class="flex-1">
            <div class="flex items-center gap-4 mb-2">
              <h2 class="text-3xl font-bold text-gray-900">{{ project.name }}</h2>
              <div class="flex items-center gap-2 px-3 py-1.5 bg-slate-100 rounded-full">
                <div :class="['w-2 h-2 rounded-full transition-colors duration-200', isConnected ? 'bg-green-500 animate-pulse' : 'bg-slate-400']"></div>
                <span class="text-xs font-medium text-slate-600">{{ isConnected ? 'Connected' : 'Disconnected' }}</span>
              </div>
            </div>
            <p class="text-gray-600 mt-2">{{ project.description }}</p>
          </div>
          <div class="flex items-center gap-3">
            <span :class=" [
              'px-3 py-1 rounded-full text-sm font-semibold uppercase',
              project.status === 'active' ? 'bg-green-100 text-green-800' : 
              project.status === 'completed' ? 'bg-blue-100 text-blue-800' :
              'bg-gray-100 text-gray-800'
            ]">{{ project.status }}</span>
            <div class="relative">
              <button @click="showProjectMenu = !showProjectMenu" class="p-2 hover:bg-gray-100 rounded-md transition-colors">
                <span class="text-xl">⋮</span>
              </button>
              <div v-if="showProjectMenu" class="absolute right-0 top-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg py-2 z-10 min-w-[200px]">
                <button 
                  v-if="project.status === 'active'"
                  @click="handleProjectAction('completed')" 
                  class="w-full px-4 py-2 text-left hover:bg-gray-50 flex items-center gap-2 text-blue-600"
                >
                  <span>✓</span> Mark as Completed
                </button>
                <button 
                  v-if="project.status !== 'archived'"
                  @click="handleProjectAction('archived')" 
                  class="w-full px-4 py-2 text-left hover:bg-gray-50 flex items-center gap-2 text-gray-600"
                >
                  <span>📦</span> Archive Project
                </button>
                <button 
                  v-if="project.status === 'archived' || project.status === 'completed'"
                  @click="handleProjectAction('active')" 
                  class="w-full px-4 py-2 text-left hover:bg-gray-50 flex items-center gap-2 text-green-600"
                >
                  <span>↻</span> Reactivate
                </button>
                <div class="border-t border-gray-200 my-1"></div>
                <button 
                  @click="confirmDeleteProject" 
                  class="w-full px-4 py-2 text-left hover:bg-red-50 flex items-center gap-2 text-red-600"
                >
                  <span>🗑️</span> Delete Project
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="flex gap-0 mb-6 border-b border-gray-200">
          <button 
            :class=" [
              'px-4 py-2 text-gray-600 hover:text-gray-900 border-b-2 border-transparent hover:border-gray-300 transition-colors',
              activeTab === 'agents' ? 'text-blue-600 border-blue-600' : ''
            ]"
            @click="activeTab = 'agents'"
          >
            Agents ({{ agents.length }})
          </button>
          <button 
            :class=" [
              'px-4 py-2 text-gray-600 hover:text-gray-900 border-b-2 border-transparent hover:border-gray-300 transition-colors',
              activeTab === 'tasks' ? 'text-blue-600 border-blue-600' : ''
            ]"
            @click="activeTab = 'tasks'"
          >
            Tasks ({{ tasks.length }})
          </button>
          <button 
            :class=" [
              'px-4 py-2 text-gray-600 hover:text-gray-900 border-b-2 border-transparent hover:border-gray-300 transition-colors',
              activeTab === 'contexts' ? 'text-blue-600 border-blue-600' : ''
            ]"
            @click="activeTab = 'contexts'"
          >
            Contexts ({{ contexts.length }})
          </button>
        </div>

        <!-- Agents Tab -->
        <div v-if="activeTab === 'agents'" class="py-6">
          <div class="flex justify-between items-center mb-6">
            <h3 class="text-xl font-semibold text-gray-900">Project Agents</h3>
            <button @click="openAddAgentModal" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors">
              + Add Agent
            </button>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <AgentCard 
              v-for="agent in agents" 
              :key="agent.id"
              :agent="agent"
              @setup="openMcpSetup"
              @edit="editAgent"
              @delete="confirmDeleteAgent"
            />
          </div>

          <div v-if="agents.length === 0" class="text-center py-12 text-gray-500">
            <p>No agents assigned to this project yet</p>
          </div>
        </div>

        <!-- Tasks Tab -->
        <div v-if="activeTab === 'tasks'" class="py-6">
          <div class="flex justify-between items-center mb-6">
            <h3 class="text-xl font-semibold text-gray-900">Project Tasks</h3>
            <button @click="openAddTaskModal" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors">
              + Create Task
            </button>
          </div>

          <div class="space-y-4">
            <TaskCard
              v-for="task in tasks"
              :key="task.id"
              :task="task"
              :agent-name="getAgentName(task.assigned_to)"
              @edit="editTask"
              @reassign="openReassignTaskModal"
              @delete="confirmDeleteTask"
            />
          </div>

          <div v-if="tasks.length === 0" class="text-center py-12 text-gray-500">
            <p>No tasks in this project yet</p>
          </div>
        </div>

        <!-- Contexts Tab -->
        <div v-if="activeTab === 'contexts'" class="py-6">
          <div class="flex justify-between items-center mb-6">
            <h3 class="text-xl font-semibold text-gray-900">Project Documentation / Context</h3>
            <button @click="showAddContextModal = true" class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md font-medium transition-colors">
              + Add Context
            </button>
          </div>

          <div class="flex gap-4 mb-6">
            <input 
              v-model="contextSearch" 
              type="text" 
              placeholder="Search contexts..." 
              class="px-4 py-2 border border-gray-300 rounded-md flex-1 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <select v-model="contextTagFilter" class="px-4 py-2 border border-gray-300 rounded-md bg-white focus:outline-none focus:ring-2 focus:ring-blue-500">
              <option value="">All Tags</option>
              <option v-for="tag in uniqueTags" :key="tag" :value="tag">
                {{ tag }}
              </option>
            </select>
          </div>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div v-for="context in filteredContexts" :key="context.id" class="bg-white p-6 rounded-lg shadow-sm">
              <div class="flex justify-between items-start mb-4">
                <h4 class="text-lg font-semibold text-gray-900">{{ context.title }}</h4>
                <div class="flex gap-2">
                  <button @click="viewContext(context)" class="p-2 text-gray-400 hover:text-gray-600 transition-colors" title="View">
                    👁️
                  </button>
                  <button @click="editContext(context)" class="p-2 text-gray-400 hover:text-gray-600 transition-colors" title="Edit">
                    ✏️
                  </button>
                  <button @click="confirmDeleteContext(context)" class="p-2 text-red-500 hover:text-red-700 transition-colors" title="Delete">
                    🗑️
                  </button>
                </div>
              </div>
              <div class="flex flex-wrap gap-2 mb-4">
                <span v-for="tag in context.tags" :key="tag" class="px-2 py-1 bg-gray-100 text-gray-800 rounded text-xs">{{ tag }}</span>
              </div>
              <div class="flex justify-between text-sm text-gray-500">
                <span>Agent: {{ getAgentName(context.agent_id) }}</span>
                <span v-if="context.task_id">Task: {{ getTaskTitle(context.task_id) }}</span>
                <span>{{ formatDate(context.created_at) }}</span>
              </div>
            </div>
          </div>

          <div v-if="filteredContexts.length === 0" class="text-center py-12 text-gray-500">
            <p>{{ contexts.length === 0 ? 'No contexts in this project yet' : 'No contexts match your search' }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Add/Edit Agent Modal -->
    <AgentModal 
      :show="showAddAgentModal"
      :agent="editingAgent"
      @close="showAddAgentModal = false"
      @save="handleSaveAgent"
    />

    <!-- Add/Edit Task Modal -->
    <TaskModal 
      :show="showAddTaskModal"
      :task="editingTask"
      :agents="agents"
      @close="showAddTaskModal = false"
      @save="handleSaveTask"
    />

    <!-- Reassign Task Modal -->
    <ReassignModal 
      :show="showReassignTaskModal"
      :task="reassigningTask"
      :agents="agents"
      :is-submitting="isReassigningTask"
      @close="showReassignTaskModal = false"
      @reassign="handleReassignTask"
    />

    <!-- Add/Edit Context Modal -->
    <ContextModal 
      :show="showAddContextModal"
      :context="editingContext"
      :agents="agents"
      :tasks="tasks"
      @close="showAddContextModal = false"
      @save="handleSaveContext"
    />

    <!-- View Context Modal -->
    <ContextViewer
      :show="showViewContextModal"
      :context="viewingContext"
      :agent-name="getAgentName(agents, viewingContext?.agent_id)"
      :task-name="viewingContext?.task_id ? getTaskTitle(tasks, viewingContext.task_id) : ''"
      @close="showViewContextModal = false"
    />

    <!-- Delete Context Confirmation -->
    <ConfirmModal
      :show="showDeleteConfirm"
      title="Confirm Delete"
      :message="`Are you sure you want to delete the context &quot;${deletingContext?.title}&quot;?`"
      confirm-text="Delete"
      @close="showDeleteConfirm = false"
      @confirm="handleDeleteContext"
    />

    <!-- Delete Agent Confirmation -->
    <ConfirmModal
      :show="showDeleteAgentConfirm"
      title="Delete Agent"
      :message="`Are you sure you want to delete the agent &quot;${deletingAgent?.name}&quot;?`"
      warning="This will also affect any tasks assigned to this agent."
      confirm-text="Delete Agent"
      @close="showDeleteAgentConfirm = false"
      @confirm="handleDeleteAgent"
    />

    <!-- Delete Task Confirmation -->
    <ConfirmModal
      :show="showDeleteTaskConfirm"
      title="Delete Task"
      :message="`Are you sure you want to delete the task &quot;${deletingTask?.title}&quot;?`"
      warning="This will also delete any related contexts."
      confirm-text="Delete Task"
      @close="showDeleteTaskConfirm = false"
      @confirm="handleDeleteTask"
    />

    <!-- MCP Setup Modal -->
    <McpSetupModal
      :show="showMcpSetupModal"
      :agent="mcpSetupAgent"
      :mcp-config="mcpConfig"
      @close="showMcpSetupModal = false"
      @download-file="downloadMcpFile"
      @download-all="handleDownloadAllMcpFiles"
    />

    <!-- Delete Project Confirmation -->
    <ConfirmModal
      :show="showDeleteProjectConfirm"
      title="Delete Project"
      :message="`Are you sure you want to permanently delete the project &quot;${project?.name}&quot;?`"
      warning="This action cannot be undone. All agents, tasks, and contexts will be deleted."
      confirm-text="Delete Project"
      @close="showDeleteProjectConfirm = false"
      @confirm="handleDeleteProject"
    />
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useProjectStore } from '../stores/projectStore'
import { useAgentStore } from '../stores/agentStore'
import { useTaskStore } from '../stores/taskStore'
import { useContextStore } from '../stores/contextStore'
import { useWebSocket } from '../composables/useWebSocket'
import AgentCard from '../components/AgentCard.vue'
import TaskCard from '../components/TaskCard.vue'
import AgentModal from '../components/AgentModal.vue'
import TaskModal from '../components/TaskModal.vue'
import ReassignModal from '../components/ReassignModal.vue'
import ContextModal from '../components/ContextModal.vue'
import ContextViewer from '../components/ContextViewer.vue'
import ConfirmModal from '../components/ConfirmModal.vue'
import McpSetupModal from '../components/McpSetupModal.vue'
import { formatDate, getUniqueTags } from '../utils/formatters'
import { getAgentName, getTaskTitle, filterContexts } from '../utils/dataHelpers'
import { useMcpSetup, downloadFile, downloadAllMcpFiles } from '../composables/useMcpSetup'
import api from '../services/api'

export default {
  name: 'ProjectDetail',
  components: {
    AgentCard,
    TaskCard,
    AgentModal,
    TaskModal,
    ReassignModal,
    ContextModal,
    ContextViewer,
    ConfirmModal,
    McpSetupModal
  },
  setup() {
    const route = useRoute()
    const projectStore = useProjectStore()
    const agentStore = useAgentStore()
    const taskStore = useTaskStore()
    const contextStore = useContextStore()
    const { connect, disconnect, on, isConnected } = useWebSocket(route.params.id)

    const activeTab = ref('agents')
    const showAddAgentModal = ref(false)
    const showAddTaskModal = ref(false)
    const showReassignTaskModal = ref(false)
    const isReassigningTask = ref(false)
    const showAddContextModal = ref(false)
    const showViewContextModal = ref(false)
    const showDeleteConfirm = ref(false)
    const showDeleteAgentConfirm = ref(false)
    const showDeleteTaskConfirm = ref(false)
    const showMcpSetupModal = ref(false)
    const showProjectMenu = ref(false)
    const showDeleteProjectConfirm = ref(false)
    const mcpSetupAgent = ref(null)

    const agentForm = ref({ name: '', role: 'frontend', team: '', status: 'active' })
    const editingAgent = ref(null)
    const deletingAgent = ref(null)
    
    const taskForm = ref({
      title: '',
      description: '',
      agent_id: '',
      priority: 'medium',
      status: 'pending'
    })
    const editingTask = ref(null)
    const reassigningTask = ref(null)
    const deletingTask = ref(null)
    const newTask = ref({ title: '', description: '', agent_id: '', priority: 'medium' })
    
    const contextForm = ref({
      title: '',
      agent_id: '',
      task_id: '',
      content: '',
      tagsString: ''
    })
    const editingContext = ref(null)
    const viewingContext = ref(null)
    const deletingContext = ref(null)
    
    const contextSearch = ref('')
    const contextTagFilter = ref('')

    const project = computed(() => projectStore.currentProject)
    const agents = computed(() => agentStore.agents)
    const tasks = computed(() => taskStore.tasks)
    const contexts = computed(() => contextStore.contexts)

    const uniqueTags = computed(() => {
      const tags = new Set()
      contexts.value.forEach(context => {
        if (context.tags && Array.isArray(context.tags)) {
          context.tags.forEach(tag => tags.add(tag))
        }
      })
      return Array.from(tags).sort()
    })

    const filteredContexts = computed(() => {
      let filtered = contexts.value

      // Filter by search
      if (contextSearch.value) {
        const search = contextSearch.value.toLowerCase()
        filtered = filtered.filter(context =>
          context.title.toLowerCase().includes(search) ||
          context.content.toLowerCase().includes(search)
        )
      }

      // Filter by tag
      if (contextTagFilter.value) {
        filtered = filtered.filter(context =>
          context.tags && context.tags.includes(contextTagFilter.value)
        )
      }

      return filtered
    })

    // MCP Setup configuration using composable
    const mcpApiUrl = computed(() => {
      return `${window.location.protocol}//${window.location.host}:8080`
    })

    const {
      mcpSettingsJson,
      mcpCopilotInstructions,
      mcpPowerShellScript,
      mcpBashScript,
      mcpVSCodeJson,
      mcpVS2026Json,
      mcpConfig,
      downloadFile,
      downloadAllMcpFiles,
      copyMcpFilesToProject
    } = useMcpSetup(mcpSetupAgent, project, mcpApiUrl, agents)

    onMounted(() => {
      const projectId = route.params.id
      projectStore.fetchProject(projectId)
      agentStore.fetchProjectAgents(projectId)
      taskStore.fetchProjectTasks(projectId)
      contextStore.fetchProjectContexts(projectId)
      
      // Connect to WebSocket for real-time updates
      connect()
      
      // Add listeners for real-time updates
      on('task_update', (data) => {
        console.log('Task update received:', data)
        taskStore.fetchProjectTasks(projectId)
      })
      
      on('agent_update', (data) => {
        console.log('Agent update received:', data)
        agentStore.fetchProjectAgents(projectId)
      })
      
      on('context_added', (data) => {
        console.log('Context added:', data)
        contextStore.fetchProjectContexts(projectId)
      })
    })

    onUnmounted(() => {
      disconnect()
    })

    // Agent management functions
    const openAddAgentModal = () => {
      editingAgent.value = null
      showAddAgentModal.value = true
    }

    const editAgent = (agent) => {
      editingAgent.value = agent
      showAddAgentModal.value = true
    }

    const handleSaveAgent = async (agentData) => {
      try {
        if (editingAgent.value) {
          await agentStore.updateAgent(editingAgent.value.id, agentData)
        } else {
          await agentStore.createAgent({
            ...agentData,
            project_id: route.params.id
          })
        }
        showAddAgentModal.value = false
        editingAgent.value = null
      } catch (error) {
        console.error('Failed to save agent:', error)
      }
    }

    const confirmDeleteAgent = (agent) => {
      deletingAgent.value = agent
      showDeleteAgentConfirm.value = true
    }

    const handleDeleteAgent = async () => {
      try {
        await agentStore.deleteAgent(deletingAgent.value.id)
        showDeleteAgentConfirm.value = false
        deletingAgent.value = null
      } catch (error) {
        console.error('Failed to delete agent:', error)
      }
    }

    // Task management functions
    const openAddTaskModal = () => {
      editingTask.value = null
      showAddTaskModal.value = true
    }

    const editTask = (task) => {
      editingTask.value = task
      showAddTaskModal.value = true
    }

    const openReassignTaskModal = (task) => {
      reassigningTask.value = task
      showReassignTaskModal.value = true
    }

    const handleReassignTask = async (reassignmentData) => {
      if (!reassigningTask.value) return

      isReassigningTask.value = true
      try {
        const updatedTask = await api.reassignTask(reassignmentData.taskId, reassignmentData.agentId)
        
        // Update the task store with the new task data
        const taskIndex = taskStore.tasks.findIndex(t => t.id === updatedTask.id)
        if (taskIndex !== -1) {
          taskStore.tasks[taskIndex] = updatedTask
        }

        showReassignTaskModal.value = false
        reassigningTask.value = null
        alert('Task reassigned successfully!')
      } catch (error) {
        console.error('Failed to reassign task:', error)
        alert('Failed to reassign task. Please try again.')
        
        // Call the error callback if provided
        if (reassignmentData.onError) {
          reassignmentData.onError()
        }
      } finally {
        isReassigningTask.value = false
      }
    }

    const handleSaveTask = async (taskData) => {
      try {
        if (editingTask.value) {
          await taskStore.updateTask(editingTask.value.id, taskData)
        } else {
          await taskStore.createTask({
            ...taskData,
            project_id: route.params.id
          })
        }
        showAddTaskModal.value = false
        editingTask.value = null
      } catch (error) {
        console.error('Failed to save task:', error)
        alert('Failed to save task. Please try again.')
      }
    }

    const confirmDeleteTask = (task) => {
      deletingTask.value = task
      showDeleteTaskConfirm.value = true
    }

    const handleDeleteTask = async () => {
      if (!deletingTask.value) return

      try {
        await taskStore.deleteTask(deletingTask.value.id)
        showDeleteTaskConfirm.value = false
        deletingTask.value = null
      } catch (error) {
        console.error('Failed to delete task:', error)
        alert('Failed to delete task. Please try again.')
      }
    }

    const handleAddTask = async () => {
      try {
        await taskStore.createTask({
          ...newTask.value,
          project_id: route.params.id,
          dependencies: []
        })
        showAddTaskModal.value = false
        newTask.value = { title: '', description: '', agent_id: '', priority: 'medium' }
      } catch (error) {
        console.error('Failed to create task:', error)
      }
    }

    const handleSaveContext = async () => {
      try {
        const tags = contextForm.value.tagsString
          .split(',')
          .map(tag => tag.trim())
          .filter(tag => tag.length > 0)

        const contextData = {
          project_id: route.params.id,
          agent_id: contextForm.value.agent_id,
          task_id: contextForm.value.task_id || null,
          title: contextForm.value.title,
          content: contextForm.value.content,
          tags: tags
        }

        if (editingContext.value) {
          await contextStore.updateContext(editingContext.value.id, contextData)
        } else {
          await contextStore.createContext(contextData)
        }

        closeContextModal()
      } catch (error) {
        console.error('Failed to save context:', error)
      }
    }

    const viewContext = (context) => {
      viewingContext.value = context
      showViewContextModal.value = true
    }

    const editContext = (context) => {
      editingContext.value = context
      contextForm.value = {
        title: context.title,
        agent_id: context.agent_id,
        task_id: context.task_id || '',
        content: context.content,
        tagsString: context.tags ? context.tags.join(', ') : ''
      }
      showAddContextModal.value = true
    }

    const confirmDeleteContext = (context) => {
      deletingContext.value = context
      showDeleteConfirm.value = true
    }

    const handleDeleteContext = async () => {
      try {
        await contextStore.deleteContext(deletingContext.value.id)
        showDeleteConfirm.value = false
        deletingContext.value = null
      } catch (error) {
        console.error('Failed to delete context:', error)
      }
    }

    const closeContextModal = () => {
      showAddContextModal.value = false
      editingContext.value = null
      contextForm.value = {
        title: '',
        agent_id: '',
        task_id: '',
        content: '',
        tagsString: ''
      }
    }

    const renderMarkdown = (content) => {
      if (!content) return ''
      const html = marked(content)
      return DOMPurify.sanitize(html)
    }

    const getAgentName = (agentId) => {
      const agent = agents.value.find(a => a.id === agentId)
      return agent ? agent.name : 'Unknown'
    }

    const getTaskTitle = (taskId) => {
      const task = tasks.value.find(t => t.id === taskId)
      return task ? task.title : 'Unknown'
    }

    const formatDate = (dateString) => {
      return new Date(dateString).toLocaleString()
    }

    // MCP Setup functions
    const openMcpSetup = (agent) => {
      mcpSetupAgent.value = agent
      showMcpSetupModal.value = true
    }

    const downloadMcpFile = (fileType) => {
      try {
        const config = mcpConfig.value
        switch (fileType) {
          case 'settings':
            downloadFile('settings.json', config.mcpSettingsJson, 'application/json')
            break
          case 'mcp':
            downloadFile('mcp.json', config.mcpVSCodeJson, 'application/json')
            break
          case '.mcp':
            downloadFile('.mcp.json', config.mcpVSCodeJson, 'application/json')
            break
          case 'copilot':
            downloadFile('copilot-instructions.md', config.mcpCopilotInstructions, 'text/markdown')
            break
          case 'powershell':
            downloadFile('mcp-agent.ps1', config.mcpPowerShellScript, 'text/plain')
            break
          case 'bash':
            downloadFile('mcp-agent.sh', config.mcpBashScript, 'text/plain')
            break
        }
      } catch (error) {
        console.error('Failed to download file:', error)
        alert(`Failed to download file: ${error.message}`)
      }
    }

    const handleDownloadAllMcpFiles = async () => {
      try {
        await downloadAllMcpFiles(mcpConfig.value, mcpSetupAgent.value.name)
      } catch (error) {
        console.error('Failed to download MCP files:', error)
        alert(`Failed to download MCP files: ${error.message}`)
      }
    }

    // Project action handlers
    const handleProjectAction = async (newStatus) => {
      showProjectMenu.value = false
      
      try {
        await projectStore.updateProjectStatus(route.params.id, newStatus)
        
        let message = ''
        if (newStatus === 'completed') {
          message = 'Project marked as completed! 🎉'
        } else if (newStatus === 'archived') {
          message = 'Project archived successfully'
        } else if (newStatus === 'active') {
          message = 'Project reactivated successfully'
        }
        
        if (message) {
          // You can add a toast notification here if you have one
          console.log(message)
        }
      } catch (error) {
        console.error('Failed to update project status:', error)
        alert('Failed to update project status. Please try again.')
      }
    }

    const confirmDeleteProject = () => {
      showProjectMenu.value = false
      showDeleteProjectConfirm.value = true
    }

    const handleDeleteProject = async () => {
      try {
        await projectStore.deleteProject(route.params.id)
        showDeleteProjectConfirm.value = false
        
        // Navigate back to projects list
        window.location.href = '/'
      } catch (error) {
        console.error('Failed to delete project:', error)
        alert('Failed to delete project. Please try again.')
      }
    }

    // Close project menu when clicking outside
    const handleClickOutside = (event) => {
      if (showProjectMenu.value && !event.target.closest('button')) {
        showProjectMenu.value = false
      }
    }

    onMounted(() => {
      const projectId = route.params.id
      projectStore.fetchProject(projectId)
      agentStore.fetchProjectAgents(projectId)
      taskStore.fetchProjectTasks(projectId)
      contextStore.fetchProjectContexts(projectId)
      
      // Connect to WebSocket for real-time updates
      connect()
      
      // Add listeners for real-time updates
      on('task_update', (data) => {
        console.log('Task update received:', data)
        taskStore.fetchProjectTasks(projectId)
      })
      
      on('agent_update', (data) => {
        console.log('Agent update received:', data)
        agentStore.fetchProjectAgents(projectId)
      })
      
      on('context_added', (data) => {
        console.log('Context added:', data)
        contextStore.fetchProjectContexts(projectId)
      })

      on('project_status_update', (data) => {
        console.log('Project status update received:', data)
        projectStore.fetchProject(projectId)
      })

      // Add click listener for closing menu
      document.addEventListener('click', handleClickOutside)
    })

    onUnmounted(() => {
      disconnect()
      document.removeEventListener('click', handleClickOutside)
    })

    return {
      project,
      agents,
      tasks,
      contexts,
      loading: computed(() => projectStore.loading),
      error: computed(() => projectStore.error),
      isConnected,
      activeTab,
      showAddAgentModal,
      showAddTaskModal,
      showReassignTaskModal,
      isReassigningTask,
      showAddContextModal,
      showViewContextModal,
      showDeleteConfirm,
      showDeleteAgentConfirm,
      showDeleteTaskConfirm,
      showMcpSetupModal,
      showProjectMenu,
      showDeleteProjectConfirm,
      mcpSetupAgent,
      agentForm,
      editingAgent,
      deletingAgent,
      taskForm,
      editingTask,
      reassigningTask,
      deletingTask,
      newTask,
      contextForm,
      editingContext,
      viewingContext,
      deletingContext,
      contextSearch,
      contextTagFilter,
      uniqueTags,
      filteredContexts,
      mcpConfig,
      openAddAgentModal,
      editAgent,
      handleSaveAgent,
      confirmDeleteAgent,
      handleDeleteAgent,
      openAddTaskModal,
      editTask,
      openReassignTaskModal,
      handleReassignTask,
      handleSaveTask,
      confirmDeleteTask,
      handleDeleteTask,
      handleAddTask,
      handleSaveContext,
      viewContext,
      editContext,
      confirmDeleteContext,
      handleDeleteContext,
      closeContextModal,
      renderMarkdown,
      getAgentName,
      getTaskTitle,
      formatDate,
      openMcpSetup,
      downloadMcpFile,
      handleDownloadAllMcpFiles,
      handleProjectAction,
      confirmDeleteProject,
      handleDeleteProject
    }
  }
}
</script>

<style scoped>
.project-detail {
  min-height: 100vh;
  background: #f5f7fa;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.container {
  max-width: 1200px;
  margin: 0 auto;
  padding: 2rem;
}

.loading, .error {
  text-align: center;
  padding: 2rem;
  font-size: 1.1rem;
}

.error {
  color: #e74c3c;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 2rem;
  background: white;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.page-header h2 {
  margin: 0 0 0.5rem 0;
  color: #2c3e50;
}

.subtitle {
  color: #7f8c8d;
  margin: 0;
}

.badge {
  padding: 0.5rem 1rem;
  border-radius: 20px;
  font-size: 0.85rem;
  font-weight: 600;
  text-transform: uppercase;
}

.badge.active { background: #d4edda; color: #155724; }
.badge.completed { background: #cce5ff; color: #004085; }
.badge.archived { background: #d6d8db; color: #383d41; }

.tabs {
  display: flex;
  gap: 1rem;
  margin-bottom: 2rem;
  background: white;
  padding: 1rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.tabs button {
  padding: 0.75rem 1.5rem;
  border: none;
  background: transparent;
  color: #7f8c8d;
  font-weight: 500;
  cursor: pointer;
  border-radius: 4px;
  transition: all 0.3s;
}

.tabs button:hover {
  background: #ecf0f1;
  color: #2c3e50;
}

.tabs button.active {
  background: #3498db;
  color: white;
}

.tab-content {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.section-header h3 {
  margin: 0;
  color: #2c3e50;
}

.btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 500;
  transition: all 0.3s;
}

.btn-primary {
  background: #3498db;
  color: white;
}

.btn-primary:hover {
  background: #2980b9;
}

.btn-secondary {
  background: #95a5a6;
  color: white;
}

.btn-secondary:hover {
  background: #7f8c8d;
}

.btn-danger {
  background: #e74c3c;
  color: white;
}

.btn-danger:hover {
  background: #c0392b;
}

.btn-icon {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 1.2rem;
  padding: 0.25rem 0.5rem;
  opacity: 0.7;
  transition: opacity 0.3s;
}

.btn-icon:hover {
  opacity: 1;
}

.btn-close {
  background: none;
  border: none;
  font-size: 2rem;
  cursor: pointer;
  color: #95a5a6;
  line-height: 1;
  padding: 0;
}

.btn-close:hover {
  color: #7f8c8d;
}

.agents-grid, .contexts-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
}

.agent-card, .context-card {
  background: #f8f9fa;
  padding: 1.5rem;
  border-radius: 8px;
  transition: transform 0.2s, box-shadow 0.2s;
}

.agent-card:hover, .context-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 8px rgba(0,0,0,0.1);
}

.agent-header, .context-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.agent-header h4, .context-header h4 {
  margin: 0;
  color: #2c3e50;
}

.context-actions {
  display: flex;
  gap: 0.25rem;
}

.status-badge {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-badge.active { background: #d4edda; color: #155724; }
.status-badge.idle { background: #fff3cd; color: #856404; }
.status-badge.offline { background: #f8d7da; color: #721c24; }

.agent-details p {
  margin: 0.5rem 0;
  color: #7f8c8c;
}

.badge.frontend { background: #e3f2fd; color: #1976d2; }
.badge.backend { background: #f3e5f5; color: #7b1fa2; }
.badge.devops { background: #e8f5e9; color: #388e3c; }

.tasks-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.task-card {
  background: #f8f9fa;
  padding: 1.5rem;
  border-radius: 8px;
}

.task-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.task-header h4 {
  margin: 0;
  color: #2c3e50;
}

.task-badges {
  display: flex;
  gap: 0.5rem;
}

.priority, .status {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
}

.priority.high { background: #f8d7da; color: #721c24; }
.priority.medium { background: #fff3cd; color: #856404; }
.priority.low { background: #d1e7dd; color: #0f5132; }

.status.pending { background: #cfe2ff; color: #084298; }
.status.in_progress { background: #fff3cd; color: #856404; }
.status.done { background: #d1e7dd; color: #0f5132; }
.status.blocked { background: #f8d7da; color: #721c24; }

.task-footer {
  display: flex;
  justify-content: space-between;
  color: #7f8c8d;
  font-size: 0.85rem;
  margin-top: 1rem;
  padding-top: 1rem;
  border-top: 1px solid #dee2e6;
}

.context-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.tag {
  background: #e3f2fd;
  color: #1976d2;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 500;
}

.context-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  color: #7f8c8d;
  font-size: 0.85rem;
}

.search-filter {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.search-input, .filter-select {
  padding: 0.75rem 1rem;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  font-size: 0.95rem;
}

.search-input {
  flex: 1;
}

.filter-select {
  min-width: 200px;
}

.empty-state {
  text-align: center;
  padding: 3rem;
  color: #7f8c8d;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  max-width: 500px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
}

.modal-large {
  max-width: 800px;
}

.modal-small {
  max-width: 400px;
}

.modal h3 {
  margin: 0 0 1.5rem 0;
  color: #2c3e50;
}

.modal-header-flex {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.modal-header-flex h3 {
  margin: 0;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  color: #2c3e50;
  font-weight: 500;
}

.form-group input,
.form-group textarea,
.form-group select {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  font-size: 0.95rem;
  font-family: inherit;
}

.form-group textarea {
  resize: vertical;
  font-family: 'Courier New', monospace;
}

.help-text {
  display: block;
  margin-top: 0.5rem;
  color: #7f8c8d;
}

.prose {
  max-width: 100%;
  line-height: 1.6;
}

.prose h1, .prose h2, .prose h3, .prose h4 {
  margin-top: 1.5rem;
  margin-bottom: 0.75rem;
  color: #2c3e50;
}

.prose h1 {
  font-size: 1.875rem;
  font-weight: 700;
}

.prose h2 {
  font-size: 1.5rem;
  font-weight: 600;
}

.prose h3 {
  font-size: 1.25rem;
  font-weight: 500;
}

.prose h4 {
  font-size: 1.125rem;
  font-weight: 500;
}

.prose p {
  margin-bottom: 1rem;
  color: #34495e;
}

.prose a {
  color: #3498db;
  text-decoration: underline;
}

.prose img {
  max-width: 100%;
  height: auto;
  border-radius: 8px;
}

.prose pre {
  background: #f8f9fa;
  padding: 1rem;
  border-radius: 6px;
  overflow-x: auto;
}

.prose code {
  font-family: 'Courier New', monospace;
  background: #eef2f3;
  padding: 0.2rem 0.4rem;
  border-radius: 4px;
}

.prose blockquote {
  border-left: 4px solid #3498db;
  padding-left: 1rem;
  margin: 0;
  color: #7f8c8d;
}

.prose ul, .prose ol {
  margin-left: 1.5rem;
  margin-bottom: 1rem;
}

.prose li {
  margin-bottom: 0.5rem;
}
</style>
