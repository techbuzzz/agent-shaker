<script setup lang="ts">
import type { Project, Agent, Task, Context, Standup,
  CreateAgentInput, CreateTaskInput, CreateContextInput, CreateStandupInput, TaskStatus } from '~/types/api'

const route = useRoute()
const projectId = computed(() => route.params.id as string)

useHead({ title: 'Project · Agent Shaker' })

const api = useApi()
const realtime = useRealtime()
const toast = useToastBus()
const settings = useSettingsStore()

// Project meta
const { data: project, error: projectError } = await useFetch<Project>(`/projects/${projectId.value}`, {
  baseURL: api.apiBase,
  key: () => `project:${projectId.value}`,
  watch: [projectId]
})
if (projectError.value) throw createError({ statusCode: 404, statusMessage: 'Project not found' })
if (project.value) settings.rememberProject(project.value.id)

// Tab data
const { data: agents,    refresh: refreshAgents    } = await useFetch<Agent[]>(   '/agents',    { baseURL: api.apiBase, key: () => `agents:${projectId.value}`,    params: { project_id: projectId.value } })
const { data: tasks,     refresh: refreshTasks     } = await useFetch<Task[]>(    '/tasks',     { baseURL: api.apiBase, key: () => `tasks:${projectId.value}`,     params: { project_id: projectId.value } })
const { data: contexts,  refresh: refreshContexts  } = await useFetch<Context[]>( '/contexts',  { baseURL: api.apiBase, key: () => `contexts:${projectId.value}`,  params: { project_id: projectId.value } })
const { data: standups,  refresh: refreshStandups  } = await useFetch<Standup[]>( '/standups',  { baseURL: api.apiBase, key: () => `standups:${projectId.value}`,  params: { project_id: projectId.value } })

// Realtime: subscribe to this project's events
onMounted(() => realtime.connect(projectId.value))
onBeforeUnmount(() => realtime.disconnect())

const refreshAll = () => Promise.all([refreshAgents(), refreshTasks(), refreshContexts(), refreshStandups()])

// Wire WS events to refresh + toasts
realtime.on('task_update',  (e) => { refreshTasks();     toast.info(`Task ${e.payload.action}: ${e.payload.task.title}`) })
realtime.on('agent_update', (e) => { refreshAgents();    toast.info(`Agent ${e.payload.action}: ${e.payload.agent.name}`) })
realtime.on('context_added',(e) => { refreshContexts();  toast.info(`New context: ${e.payload.context.title}`) })

// Tabs (UTabs in v3)
const tabItems = [
  { label: 'Overview',   value: 'overview', icon: 'i-lucide-info' },
  { label: 'Agents',     value: 'agents',   icon: 'i-lucide-bot' },
  { label: 'Tasks',      value: 'tasks',    icon: 'i-lucide-list-checks' },
  { label: 'Contexts',   value: 'contexts', icon: 'i-lucide-book-text' },
  { label: 'Standups',   value: 'standups', icon: 'i-lucide-calendar-days' }
]
const activeTab = ref<string>('overview')

// --- Modal state ---
const showAgent = ref(false)
const editingAgent = ref<Agent | null>(null)
const deletingAgent = ref<Agent | null>(null)
const deletingAgentOpen = computed({
  get: () => deletingAgent.value !== null,
  set: (v: boolean) => { if (!v) deletingAgent.value = null }
})

const showTask = ref(false)
const editingTask = ref<Task | null>(null)
const deletingTask = ref<Task | null>(null)
const deletingTaskOpen = computed({
  get: () => deletingTask.value !== null,
  set: (v: boolean) => { if (!v) deletingTask.value = null }
})

const showContext = ref(false)
const editingContext = ref<Context | null>(null)
const viewingContext = ref<Context | null>(null)
const deletingContext = ref<Context | null>(null)
const viewingContextOpen = computed({
  get: () => viewingContext.value !== null,
  set: (v: boolean) => { if (!v) viewingContext.value = null }
})
const deletingContextOpen = computed({
  get: () => deletingContext.value !== null,
  set: (v: boolean) => { if (!v) deletingContext.value = null }
})

const showStandup = ref(false)

const agentsForSelect = computed(() => (agents.value ?? []).map((a) => ({ id: a.id, name: a.name })))
const agentsById = computed(() => Object.fromEntries((agents.value ?? []).map((a) => [a.id, a.name])))

// --- Handlers ---
async function handleCreateAgent(input: CreateAgentInput) {
  await api.createAgent(input); await refreshAgents(); toast.success('Agent registered')
}
async function handleCreateTask(input: CreateTaskInput) {
  await api.createTask(input); await refreshTasks(); toast.success('Task created')
}
async function handleUpdateTask(id: string, input: Partial<CreateTaskInput>) {
  await api.updateTask(id, input as any); await refreshTasks(); toast.success('Task updated')
}
async function handleStatusChange({ task, status }: { task: Task; status: TaskStatus }) {
  await api.updateTaskStatus(task.id, status); await refreshTasks(); toast.success(`Task moved to ${status.replace('_', ' ')}`)
}
async function handleCreateContext(input: CreateContextInput) {
  await api.createContext(input); await refreshContexts(); toast.success('Context published')
}
async function handleCreateStandup(input: CreateStandupInput) {
  await api.createStandup(input); await refreshStandups(); toast.success('Standup submitted')
}

async function deleteAgent() {
  if (!deletingAgent.value) return
  await api.deleteAgent(deletingAgent.value.id); await refreshAgents()
  deletingAgent.value = null; toast.success('Agent removed')
}
async function deleteTask() {
  if (!deletingTask.value) return
  await api.deleteTask(deletingTask.value.id); await refreshTasks()
  deletingTask.value = null; toast.success('Task removed')
}
async function deleteContext() {
  if (!deletingContext.value) return
  await api.deleteContext(deletingContext.value.id); await refreshContexts()
  deletingContext.value = null; toast.success('Context removed')
}
</script>

<template>
  <div>
    <div class="mb-6">
      <NuxtLink to="/projects" class="text-sm text-muted hover:text-primary inline-flex items-center gap-1 mb-3">
        <UIcon name="i-lucide-arrow-left" class="w-3.5 h-3.5" /> All projects
      </NuxtLink>
      <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
        <div>
          <h1 class="text-2xl font-semibold">{{ project?.name }}</h1>
          <p class="text-muted mt-1">{{ project?.description || 'No description' }}</p>
          <UBadge class="mt-2" :color="project?.status === 'active' ? 'success' : 'neutral'" variant="subtle" size="sm">
            {{ project?.status }}
          </UBadge>
        </div>
        <UButton to="/settings" variant="ghost" color="neutral" icon="i-lucide-settings" size="sm">Settings</UButton>
      </div>
    </div>

    <UTabs v-model="activeTab" :items="tabItems">
      <!-- OVERVIEW -->
      <template #content="{ item }">
        <div v-if="item.value === 'overview'" class="grid grid-cols-2 md:grid-cols-4 gap-4 py-4">
          <StatCard title="Agents" :value="agents?.length ?? 0" icon="i-lucide-bot" accent="success" />
          <StatCard title="Tasks" :value="tasks?.length ?? 0" icon="i-lucide-list-checks" accent="warning" />
          <StatCard title="Contexts" :value="contexts?.length ?? 0" icon="i-lucide-book-text" accent="info" />
          <StatCard title="Standups" :value="standups?.length ?? 0" icon="i-lucide-calendar-days" accent="primary" />
        </div>

        <!-- AGENTS -->
        <div v-else-if="item.value === 'agents'" class="py-4">
          <div class="flex justify-end mb-4">
            <UButton icon="i-lucide-plus" size="sm" color="primary" @click="editingAgent = null; showAgent = true">Register agent</UButton>
          </div>
          <EmptyState v-if="!agents?.length" icon="i-lucide-bot" title="No agents yet" />
          <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            <AgentCard
              v-for="a in agents" :key="a.id" :agent="a"
              @edit="(ag) => { editingAgent = ag; showAgent = true }"
              @delete="(ag) => deletingAgent = ag"
            />
          </div>
        </div>

        <!-- TASKS -->
        <div v-else-if="item.value === 'tasks'" class="py-4">
          <div class="flex justify-end mb-4">
            <UButton icon="i-lucide-plus" size="sm" color="primary" @click="editingTask = null; showTask = true">Create task</UButton>
          </div>
          <EmptyState v-if="!tasks?.length" icon="i-lucide-list-checks" title="No tasks yet" />
          <div v-else class="space-y-3">
            <TaskCard
              v-for="t in tasks" :key="t.id" :task="t" :assignee-name="agentsById[t.assigned_to]"
              @edit="(tk) => { editingTask = tk; showTask = true }"
              @delete="(tk) => deletingTask = tk"
              @change-status="handleStatusChange"
            />
          </div>
        </div>

        <!-- CONTEXTS -->
        <div v-else-if="item.value === 'contexts'" class="py-4">
          <div class="flex justify-end mb-4">
            <UButton icon="i-lucide-plus" size="sm" color="primary" @click="editingContext = null; showContext = true">Share context</UButton>
          </div>
          <EmptyState v-if="!contexts?.length" icon="i-lucide-book-text" title="No contexts yet" description="Document decisions, notes, or APIs." />
          <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <ContextCard
              v-for="c in contexts" :key="c.id" :context="c" :agent-name="agentsById[c.agent_id]"
              @view="(c) => viewingContext = c"
              @delete="(c) => deletingContext = c"
            />
          </div>
        </div>

        <!-- STANDUPS -->
        <div v-else-if="item.value === 'standups'" class="py-4">
          <div class="flex justify-end mb-4">
            <UButton icon="i-lucide-plus" size="sm" color="primary" @click="showStandup = true">Submit standup</UButton>
          </div>
          <EmptyState v-if="!standups?.length" icon="i-lucide-calendar-days" title="No standups yet" />
          <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <StandupCard v-for="s in standups" :key="s.id" :standup="s" :agent-name="agentsById[s.agent_id]" @delete="(s) => api.deleteStandup(s.id).then(refreshStandups)" />
          </div>
        </div>
      </template>
    </UTabs>

    <!-- Modals -->
    <AgentFormModal v-model="showAgent" :project-id="projectId" :initial="editingAgent" @submit="editingAgent ? api.updateAgentStatus(editingAgent.id, $event.status || 'active') : handleCreateAgent($event)" />
    <TaskFormModal v-model="showTask" :project-id="projectId" :agents="agentsForSelect" :initial="editingTask" @submit="editingTask ? handleUpdateTask(editingTask.id, $event) : handleCreateTask($event)" />
    <ContextFormModal v-model="showContext" :project-id="projectId" :agents="agentsForSelect" :initial="editingContext" @submit="editingContext ? api.updateContext(editingContext.id, $event).then(refreshContexts) : handleCreateContext($event)" />
    <StandupFormModal v-model="showStandup" :project-id="projectId" :agents="agentsForSelect" @submit="handleCreateStandup" />

    <ContextViewer v-model="viewingContextOpen" :context="viewingContext" :agent-name="viewingContext ? agentsById[viewingContext.agent_id] : undefined" />

    <ConfirmDialog v-model="deletingAgentOpen" :title="`Delete agent ${deletingAgent?.name}?`" confirm-label="Delete" @confirm="deleteAgent" />
    <ConfirmDialog v-model="deletingTaskOpen"  :title="`Delete task ${deletingTask?.title}?`"  confirm-label="Delete" @confirm="deleteTask" />
    <ConfirmDialog v-model="deletingContextOpen" :title="`Delete context ${deletingContext?.title}?`" confirm-label="Delete" @confirm="deleteContext" />
  </div>
</template>
