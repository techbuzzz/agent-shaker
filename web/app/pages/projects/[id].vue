<script setup lang="ts">
import type { Project, Agent, Task, Context, Standup,
  CreateAgentInput, CreateTaskInput, CreateContextInput, CreateStandupInput, TaskStatus,
  Milestone, CreateMilestoneInput, MilestoneStatus,
  ProjectRepo, CreateProjectRepoInput,
  GlobalContext, CreateGlobalContextInput
} from '~/types/api'

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
const { data: milestones, refresh: refreshMilestones } = await useFetch<Milestone[]>('/milestones', { baseURL: api.apiBase, key: () => `milestones:${projectId.value}`, params: { project_id: projectId.value } })
const { data: repos,      refresh: refreshRepos      } = await useFetch<ProjectRepo[]>('/project_repos', { baseURL: api.apiBase, key: () => `repos:${projectId.value}`, params: { project_id: projectId.value } })
const { data: projectGctx, refresh: refreshGctx     } = await useFetch<GlobalContext[]>('/global_contexts', { baseURL: api.apiBase, key: () => `gctx:${projectId.value}`, params: { scope: 'project', project_id: projectId.value } })

// Realtime: subscribe to this project's events
onMounted(() => realtime.connect(projectId.value))
onBeforeUnmount(() => realtime.disconnect())

const refreshAll = () => Promise.all([refreshAgents(), refreshTasks(), refreshContexts(), refreshStandups(), refreshMilestones(), refreshRepos(), refreshGctx()])

// Wire WS events to refresh + toasts
realtime.on('task_update',  (e) => { refreshTasks();     toast.info(`Task ${e.payload.action}: ${e.payload.task.title}`) })
realtime.on('agent_update', (e) => { refreshAgents();    toast.info(`Agent ${e.payload.action}: ${e.payload.agent.name}`) })
realtime.on('context_added',(e) => { refreshContexts();  toast.info(`New context: ${e.payload.context.title}`) })
realtime.on('milestone_added',  () => refreshMilestones())
realtime.on('milestone_updated',() => refreshMilestones())
realtime.on('milestone_deleted',() => refreshMilestones())
realtime.on('project_repo_added',  () => refreshRepos())
realtime.on('project_repo_updated',() => refreshRepos())
realtime.on('project_repo_deleted',() => refreshRepos())
realtime.on('global_context_added',  () => refreshGctx())
realtime.on('global_context_updated',() => refreshGctx())
realtime.on('global_context_deleted',() => refreshGctx())
realtime.on('task_added', () => refreshTasks())

// Tabs (UTabs in v3)
const tabItems = [
  { label: 'Overview',   value: 'overview', icon: 'i-lucide-info' },
  { label: 'Agents',     value: 'agents',   icon: 'i-lucide-bot' },
  { label: 'Tasks',      value: 'tasks',    icon: 'i-lucide-list-checks' },
  { label: 'Features',   value: 'features', icon: 'i-lucide-layout-grid' },
  { label: 'Milestones', value: 'milestones', icon: 'i-lucide-flag' },
  { label: 'Repos',      value: 'repos',    icon: 'i-lucide-git-branch' },
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

const showMilestone = ref(false)
const editingMilestone = ref<Milestone | null>(null)
const deletingMilestone = ref<Milestone | null>(null)
const deletingMilestoneOpen = computed({
  get: () => deletingMilestone.value !== null,
  set: (v: boolean) => { if (!v) deletingMilestone.value = null }
})

const showRepo = ref(false)
const deletingRepo = ref<ProjectRepo | null>(null)
const deletingRepoOpen = computed({
  get: () => deletingRepo.value !== null,
  set: (v: boolean) => { if (!v) deletingRepo.value = null }
})

const showProjectGctx = ref(false)
const deletingGctx = ref<GlobalContext | null>(null)
const viewingGctx = ref<GlobalContext | null>(null)
const deletingGctxOpen = computed({
  get: () => deletingGctx.value !== null,
  set: (v: boolean) => { if (!v) deletingGctx.value = null }
})
const viewingGctxOpen = computed({
  get: () => viewingGctx.value !== null,
  set: (v: boolean) => { if (!v) viewingGctx.value = null }
})

const agentsForSelect = computed(() => (agents.value ?? []).map((a) => ({ id: a.id, name: a.name })))
const agentsById = computed(() => Object.fromEntries((agents.value ?? []).map((a) => [a.id, a.name])))

// Per-milestone task stats. Derived client-side from the `tasks` list.
const milestoneStats = computed(() => {
  const out = new Map<string, { total: number; open: number }>()
  for (const t of tasks.value ?? []) {
    if (!t.milestone_id) continue
    const cur = out.get(t.milestone_id) ?? { total: 0, open: 0 }
    cur.total += 1
    if (t.status !== 'done' && t.status !== 'cancelled') cur.open += 1
    out.set(t.milestone_id, cur)
  }
  return out
})

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

async function handleCreateMilestone(input: CreateMilestoneInput) {
  await api.createMilestone(input); await refreshMilestones(); toast.success('Milestone created')
}
async function handleMilestoneStatusChange({ milestone, status }: { milestone: Milestone; status: MilestoneStatus }) {
  try {
    await api.updateMilestoneStatus(milestone.id, { status, description: milestone.description })
    await refreshMilestones()
    toast.success(`Milestone moved to ${status}`)
  } catch (err: unknown) {
    const e = err as { data?: { message?: string }; message?: string }
    toast.error(e?.data?.message ?? e?.message ?? 'Failed to update milestone')
    throw err
  }
}

async function handleCreateRepo(input: CreateProjectRepoInput) {
  await api.createProjectRepo(input); await refreshRepos(); toast.success('Repo added')
}
async function handleCreateProjectGctx(input: CreateGlobalContextInput) {
  await api.createGlobalContext(input); await refreshGctx(); toast.success('Doc published')
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
async function deleteMilestone() {
  if (!deletingMilestone.value) return
  await api.deleteMilestone(deletingMilestone.value.id); await refreshMilestones()
  deletingMilestone.value = null; toast.success('Milestone removed')
}
async function deleteRepo() {
  if (!deletingRepo.value) return
  await api.deleteProjectRepo(deletingRepo.value.id); await refreshRepos()
  deletingRepo.value = null; toast.success('Repo removed')
}
async function deleteGctx() {
  if (!deletingGctx.value) return
  await api.deleteGlobalContext(deletingGctx.value.id); await refreshGctx()
  deletingGctx.value = null; toast.success('Doc removed')
}

// PM-role detection (used to gate "publish global" + "create milestone" CTAs).
// Best-effort: we look for an agent whose name starts with "pm-" or whose
// role is exactly "pm" in the loaded agent list.
const currentAgentIsPM = computed(() => {
  const stored = settings.recentAgentId
  if (!stored) return false
  const me = (agents.value ?? []).find((a) => a.id === stored)
  return !!me && me.role === 'pm'
})
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
          <StatCard title="Milestones" :value="milestones?.length ?? 0" icon="i-lucide-flag" accent="info" />
          <StatCard title="Repos" :value="repos?.length ?? 0" icon="i-lucide-git-branch" accent="success" />
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

        <!-- FEATURES (derived) -->
        <div v-else-if="item.value === 'features'" class="py-4">
          <p class="text-sm text-muted mb-3">
            Features are derived from task tags prefixed with <code class="font-mono">feature:</code>.
            Tag a task in the task editor (or via the MCP <code class="font-mono">create_task</code> tool) to roll it up here.
          </p>
          <FeatureList :tasks="tasks ?? []" />
        </div>

        <!-- MILESTONES -->
        <div v-else-if="item.value === 'milestones'" class="py-4">
          <div class="flex justify-end mb-4">
            <UButton
              icon="i-lucide-plus" size="sm" color="primary"
              :disabled="!currentAgentIsPM"
              @click="editingMilestone = null; showMilestone = true"
            >
              {{ currentAgentIsPM ? 'Create milestone' : 'PM-only' }}
            </UButton>
          </div>
          <EmptyState
            v-if="!milestones?.length"
            icon="i-lucide-flag"
            title="No milestones yet"
            description="Milestones group tasks by a target date."
          />
          <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <MilestoneCard
              v-for="m in milestones" :key="m.id"
              :milestone="m"
              :task-count="milestoneStats.get(m.id)?.total ?? 0"
              :open-task-count="milestoneStats.get(m.id)?.open ?? 0"
              @change-status="handleMilestoneStatusChange"
              @delete="(m) => deletingMilestone = m"
            />
          </div>
        </div>

        <!-- REPOS -->
        <div v-else-if="item.value === 'repos'" class="py-4">
          <div class="flex justify-end mb-4">
            <UButton icon="i-lucide-plus" size="sm" color="primary" @click="showRepo = true">Add repo</UButton>
          </div>
          <EmptyState
            v-if="!repos?.length"
            icon="i-lucide-git-branch"
            title="No repos registered"
            description="Link the git repos that participate in this project."
          />
          <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <ProjectRepoCard
              v-for="r in repos" :key="r.id"
              :repo="r"
              :agent-name="r.agent_id ? agentsById[r.agent_id] : undefined"
              @delete="(r) => deletingRepo = r"
            />
          </div>
        </div>

        <!-- CONTEXTS -->
        <div v-else-if="item.value === 'contexts'" class="py-4">
          <div class="flex justify-end mb-4 gap-2">
            <UButton icon="i-lucide-plus" size="sm" color="primary" @click="editingContext = null; showContext = true">Share context</UButton>
            <UButton icon="i-lucide-globe" size="sm" color="neutral" variant="outline" @click="showProjectGctx = true">Project doc</UButton>
          </div>

          <div v-if="projectGctx?.length" class="mb-6">
            <h3 class="text-sm font-medium mb-2 text-muted">Project docs (global_contexts, scope=project)</h3>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
              <GlobalContextCard
                v-for="g in projectGctx" :key="g.id"
                :context="g" :agent-name="agentsById[g.agent_id]"
                @view="(g) => viewingGctx = g"
                @delete="(g) => deletingGctx = g"
              />
            </div>
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
    <TaskFormModal v-model="showTask" :project-id="projectId" :agents="agentsForSelect" :milestones="milestones ?? []" :initial="editingTask" @submit="editingTask ? handleUpdateTask(editingTask.id, $event) : handleCreateTask($event)" />
    <ContextFormModal v-model="showContext" :project-id="projectId" :agents="agentsForSelect" :initial="editingContext" @submit="editingContext ? api.updateContext(editingContext.id, $event).then(refreshContexts) : handleCreateContext($event)" />
    <StandupFormModal v-model="showStandup" :project-id="projectId" :agents="agentsForSelect" @submit="handleCreateStandup" />

    <MilestoneFormModal
      v-model="showMilestone"
      :project-id="projectId"
      :created-by="settings.recentAgentId ?? ''"
      :initial="editingMilestone"
      @submit="handleCreateMilestone"
    />

    <ProjectRepoFormModal
      v-model="showRepo"
      :project-id="projectId"
      :agents="agentsForSelect"
      @submit="handleCreateRepo"
    />

    <GlobalContextFormModal
      v-model="showProjectGctx"
      :agent-id="settings.recentAgentId ?? ''"
      :project-id="projectId"
      initial-scope="project"
      @submit="handleCreateProjectGctx"
    />

    <ContextViewer v-model="viewingContextOpen" :context="viewingContext" :agent-name="viewingContext ? agentsById[viewingContext.agent_id] : undefined" />
    <ContextViewer v-model="viewingGctxOpen" :context="viewingGctx" :agent-name="viewingGctx ? agentsById[viewingGctx.agent_id] : undefined" />

    <ConfirmDialog v-model="deletingAgentOpen" :title="`Delete agent ${deletingAgent?.name}?`" confirm-label="Delete" @confirm="deleteAgent" />
    <ConfirmDialog v-model="deletingTaskOpen"  :title="`Delete task ${deletingTask?.title}?`"  confirm-label="Delete" @confirm="deleteTask" />
    <ConfirmDialog v-model="deletingContextOpen" :title="`Delete context ${deletingContext?.title}?`" confirm-label="Delete" @confirm="deleteContext" />
    <ConfirmDialog v-model="deletingMilestoneOpen" :title="`Delete milestone ${deletingMilestone?.title}?`" confirm-label="Delete" @confirm="deleteMilestone" />
    <ConfirmDialog v-model="deletingRepoOpen" :title="`Remove repo ${deletingRepo?.url}?`" confirm-label="Remove" @confirm="deleteRepo" />
    <ConfirmDialog v-model="deletingGctxOpen" :title="`Delete doc ${deletingGctx?.title}?`" confirm-label="Delete" @confirm="deleteGctx" />
  </div>
</template>
