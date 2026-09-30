/**
 * Thin wrapper over Nuxt's $fetch with:
 *   - baseURL resolved from the server-URL cookie (set in Settings)
 *   - typed error normalization
 *   - convenience methods that mirror the old services/api.js
 *
 * Replaces web/src/services/api.js. No axios.
 */
import type {
  Project, Agent, Task, Context, Standup, Heartbeat, DashboardStats,
  CreateProjectInput, UpdateProjectStatusInput,
  CreateAgentInput, UpdateAgentStatusInput,
  CreateTaskInput, UpdateTaskInput, UpdateTaskStatusInput, ReassignTaskInput,
  CreateContextInput, CreateStandupInput, CreateHeartbeatInput,
  // Mesh-app additions — Phase 1/2/3
  Milestone, CreateMilestoneInput, UpdateMilestoneStatusInput,
  ProjectRepo, CreateProjectRepoInput,
  GlobalContext, CreateGlobalContextInput, UpdateGlobalContextInput
} from '~/types/api'

export class ApiConnectionError extends Error {
  constructor(message: string, public readonly url: string) {
    super(message)
    this.name = 'ApiConnectionError'
  }
}

export interface UseApi {
  base: string
  isConnected: Ref<boolean>
  lastCheckedAt: Ref<number | null>
  checkHealth: () => Promise<boolean>
  // Projects
  listProjects: () => Promise<Project[]>
  getProject: (id: string) => Promise<Project>
  createProject: (input: CreateProjectInput) => Promise<Project>
  updateProjectStatus: (id: string, status: UpdateProjectStatusInput['status']) => Promise<Project>
  deleteProject: (id: string) => Promise<void>
  // Agents
  listAgents: (projectId?: string) => Promise<Agent[]>
  getAgent: (id: string) => Promise<Agent>
  createAgent: (input: CreateAgentInput) => Promise<Agent>
  updateAgentStatus: (id: string, status: UpdateAgentStatusInput['status']) => Promise<Agent>
  deleteAgent: (id: string) => Promise<void>
  // Tasks
  listTasks: (filters?: { project_id?: string; status?: string; priority?: string; assigned_to?: string; agent_id?: string }) => Promise<Task[]>
  getTask: (id: string) => Promise<Task>
  createTask: (input: CreateTaskInput) => Promise<Task>
  updateTask: (id: string, input: UpdateTaskInput) => Promise<Task>
  updateTaskStatus: (id: string, status: UpdateTaskStatusInput['status']) => Promise<Task>
  reassignTask: (id: string, assignedTo: string) => Promise<Task>
  deleteTask: (id: string) => Promise<void>
  // Contexts
  listContexts: (filters?: { project_id?: string; tags?: string }) => Promise<Context[]>
  getContext: (id: string) => Promise<Context>
  createContext: (input: CreateContextInput) => Promise<Context>
  updateContext: (id: string, input: Partial<CreateContextInput>) => Promise<Context>
  deleteContext: (id: string) => Promise<void>
  // Standups
  listStandups: (filters?: { project_id?: string; agent_id?: string; date?: string }) => Promise<Standup[]>
  getStandup: (id: string) => Promise<Standup>
  createStandup: (input: CreateStandupInput) => Promise<Standup>
  updateStandup: (id: string, input: Partial<CreateStandupInput>) => Promise<Standup>
  deleteStandup: (id: string) => Promise<void>
  // Heartbeats
  recordHeartbeat: (input: CreateHeartbeatInput) => Promise<Heartbeat>
  getAgentHeartbeats: (agentId: string, limit?: number) => Promise<Heartbeat[]>
  // Dashboard
  getDashboardStats: () => Promise<DashboardStats>
  // Milestones (Phase 1)
  listMilestones: (projectId: string) => Promise<Milestone[]>
  getMilestone: (id: string) => Promise<Milestone>
  createMilestone: (input: CreateMilestoneInput) => Promise<Milestone>
  updateMilestoneStatus: (id: string, input: UpdateMilestoneStatusInput) => Promise<Milestone>
  deleteMilestone: (id: string) => Promise<void>
  // Project repos (Phase 2)
  listProjectRepos: (projectId: string) => Promise<ProjectRepo[]>
  getProjectRepo: (id: string) => Promise<ProjectRepo>
  createProjectRepo: (input: CreateProjectRepoInput) => Promise<ProjectRepo>
  updateProjectRepo: (id: string, input: CreateProjectRepoInput) => Promise<ProjectRepo>
  deleteProjectRepo: (id: string) => Promise<void>
  // Global contexts (Phase 3)
  listGlobalContexts: (filters?: { scope?: 'global' | 'project'; project_id?: string; tag_prefix?: string }) => Promise<GlobalContext[]>
  getGlobalContext: (id: string) => Promise<GlobalContext>
  createGlobalContext: (input: CreateGlobalContextInput) => Promise<GlobalContext>
  updateGlobalContext: (id: string, input: UpdateGlobalContextInput) => Promise<GlobalContext>
  deleteGlobalContext: (id: string) => Promise<void>
}

function toParams(obj: Record<string, string | number | undefined | null>): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [k, v] of Object.entries(obj)) {
    if (v !== undefined && v !== null && v !== '') out[k] = String(v)
  }
  return out
}

export function useApi(): UseApi {
  const { apiBase } = useServerUrl()

  const isConnected = useState<boolean>('api:connected', () => false)
  const lastCheckedAt = useState<number | null>('api:last-check', () => null)

  async function request<T>(path: string, opts: Record<string, unknown> = {}): Promise<T> {
    try {
      // Nuxt's $fetch on the server uses Nitro's devProxy; on the client the proxy is also wired.
      return await $fetch<T>(path, { baseURL: apiBase.value, ...opts })
    } catch (err: unknown) {
      // $fetch errors: FetchError with status/data/cause; on network error, status undefined.
      const fe = err as { status?: number; statusCode?: number; data?: { message?: string }; cause?: unknown; message?: string }
      const isConn = !fe?.status && !fe?.statusCode
      if (isConn) {
        isConnected.value = false
        throw new ApiConnectionError(fe?.message || 'Cannot reach MCP server', apiBase.value)
      }
      throw err
    }
  }

  async function checkHealth(): Promise<boolean> {
    try {
      // Hit /health directly (outside /api) so the dashboard can probe before any project load.
      await $fetch('/health', {
        baseURL: apiBase.value.replace(/\/api$/, ''),
        method: 'GET'
      })
      isConnected.value = true
      lastCheckedAt.value = Date.now()
      return true
    } catch {
      isConnected.value = false
      lastCheckedAt.value = Date.now()
      return false
    }
  }

  return {
    base: apiBase.value,
    isConnected,
    lastCheckedAt,
    checkHealth,

    listProjects:        ()       => request<Project[]>('/projects'),
    getProject:          (id)     => request<Project>(`/projects/${id}`),
    createProject:       (input)  => request<Project>('/projects', { method: 'POST', body: input }),
    updateProjectStatus: (id, s)  => request<Project>(`/projects/${id}/status`, { method: 'PUT', body: { status: s } }),
    deleteProject:       async (id) => { await request<void>(`/projects/${id}`, { method: 'DELETE' }) },

    listAgents:          (projectId) => request<Agent[]>('/agents', { params: toParams({ project_id: projectId }) }),
    getAgent:            (id)         => request<Agent>(`/agents/${id}`),
    createAgent:         (input)      => request<Agent>('/agents', { method: 'POST', body: input }),
    updateAgentStatus:   (id, s)      => request<Agent>(`/agents/${id}/status`, { method: 'PUT', body: { status: s } }),
    deleteAgent:         async (id) => { await request<void>(`/agents/${id}`, { method: 'DELETE' }) },

    listTasks:           (filters = {}) => request<Task[]>('/tasks', { params: toParams(filters) }),
    getTask:             (id)            => request<Task>(`/tasks/${id}`),
    createTask:          (input)         => request<Task>('/tasks', { method: 'POST', body: input }),
    updateTask:          (id, input)     => request<Task>(`/tasks/${id}`, { method: 'PUT', body: input }),
    updateTaskStatus:    (id, status)    => request<Task>(`/tasks/${id}/status`, { method: 'PUT', body: { status } }),
    reassignTask:        (id, assignedTo) => request<Task>(`/tasks/${id}/reassign`, { method: 'PUT', body: { assigned_to: assignedTo } }),
    deleteTask:          async (id) => { await request<void>(`/tasks/${id}`, { method: 'DELETE' }) },

    listContexts:        (filters = {}) => request<Context[]>('/contexts', { params: toParams(filters) }),
    getContext:          (id)            => request<Context>(`/contexts/${id}`),
    createContext:       (input)         => request<Context>('/contexts', { method: 'POST', body: input }),
    updateContext:       (id, input)     => request<Context>(`/contexts/${id}`, { method: 'PUT', body: input }),
    deleteContext:       async (id) => { await request<void>(`/contexts/${id}`, { method: 'DELETE' }) },

    listStandups:        (filters = {}) => request<Standup[]>('/standups', { params: toParams(filters) }),
    getStandup:          (id)            => request<Standup>(`/standups/${id}`),
    createStandup:       (input)         => request<Standup>('/standups', { method: 'POST', body: input }),
    updateStandup:       (id, input)     => request<Standup>(`/standups/${id}`, { method: 'PUT', body: input }),
    deleteStandup:       async (id) => { await request<void>(`/standups/${id}`, { method: 'DELETE' }) },

    recordHeartbeat:     (input)       => request<Heartbeat>('/heartbeats', { method: 'POST', body: input }),
    getAgentHeartbeats:  (agentId, limit = 50) => request<Heartbeat[]>(`/agents/${agentId}/heartbeats`, { params: toParams({ limit }) }),

    getDashboardStats:   () => request<DashboardStats>('/dashboard'),

    // Milestones
    listMilestones:      (projectId)   => request<Milestone[]>('/milestones', { params: { project_id: projectId } }),
    getMilestone:        (id)          => request<Milestone>(`/milestones/${id}`),
    createMilestone:     (input)       => request<Milestone>('/milestones', { method: 'POST', body: input }),
    updateMilestoneStatus: (id, input) => request<Milestone>(`/milestones/${id}/status`, { method: 'PUT', body: input }),
    deleteMilestone:     async (id)     => { await request<void>(`/milestones/${id}`, { method: 'DELETE' }) },

    // Project repos
    listProjectRepos:    (projectId)   => request<ProjectRepo[]>('/project_repos', { params: { project_id: projectId } }),
    getProjectRepo:      (id)          => request<ProjectRepo>(`/project_repos/${id}`),
    createProjectRepo:   (input)       => request<ProjectRepo>('/project_repos', { method: 'POST', body: input }),
    updateProjectRepo:   (id, input)   => request<ProjectRepo>(`/project_repos/${id}`, { method: 'PUT', body: input }),
    deleteProjectRepo:   async (id)     => { await request<void>(`/project_repos/${id}`, { method: 'DELETE' }) },

    // Global contexts
    listGlobalContexts:  (filters = {}) => request<GlobalContext[]>('/global_contexts', { params: toParams(filters) }),
    getGlobalContext:    (id)           => request<GlobalContext>(`/global_contexts/${id}`),
    createGlobalContext: (input)        => request<GlobalContext>('/global_contexts', { method: 'POST', body: input }),
    updateGlobalContext: (id, input)    => request<GlobalContext>(`/global_contexts/${id}`, { method: 'PUT', body: input }),
    deleteGlobalContext: async (id)      => { await request<void>(`/global_contexts/${id}`, { method: 'DELETE' }) }
  }
}
