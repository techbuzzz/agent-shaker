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
  // Federation Phase 2 (migrations/017)
  AgentConfigResponse, AgentConnectionInput, AgentConnectionResponse, ConfigFormat,
  CreateTaskInput, UpdateTaskInput, UpdateTaskStatusInput, ReassignTaskInput,
  CreateContextInput, CreateStandupInput, CreateHeartbeatInput,
  // Mesh-app additions — Phase 1/2/3
  Milestone, CreateMilestoneInput, UpdateMilestoneStatusInput,
  ProjectRepo, CreateProjectRepoInput,
  GlobalContext, CreateGlobalContextInput, UpdateGlobalContextInput,
  // Coordination Plane (migrations/015)
  Blocker, BlockerStatus, ResolveBlockerInput, DismissBlockerInput
} from '~/types/api'
import type {
  Person, CreatePersonInput, APIKey, CreateKeyInput, IssuedAPIKey,
  PeopleResponse, KeysResponse
} from '~/types/admin'

export class ApiConnectionError extends Error {
  constructor(message: string, public readonly url: string) {
    super(message)
    this.name = 'ApiConnectionError'
  }
}

export interface UseApi {
  /**
   * Reactive API base (`/api` for same-origin, or an absolute URL when the
   * user points the app at a different backend from Settings). Pages pass this
   * straight into `useFetch({ baseURL })`, so it must stay a ref: a plain
   * string would go stale the moment the server-URL cookie changes.
   */
  apiBase: Ref<string>
  /** Snapshot of `apiBase` at call time. Prefer `apiBase` in new code. */
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
  /**
   * GET /api/agents/{id}/config — the generated MCP files for one agent.
   *
   * `format` is optional; omitted, the server picks the layout matching the
   * agent's tool_type, which is the whole reason that field exists.
   */
  getAgentConfig: (id: string, format?: ConfigFormat) => Promise<AgentConfigResponse>
  /**
   * POST /api/agents/{id}/connection — mints a credential bound to this agent
   * and returns it, once, along with the rendered files.
   *
   * NOT idempotent: a retry mints a second credential. The caller is expected to
   * warn before sending and to revoke the unused one.
   */
  mintAgentConnection: (id: string, input?: AgentConnectionInput) => Promise<AgentConnectionResponse>
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
  /**
   * Full-text search within one project. `project_id` is not optional here even
   * though it is on `listContexts`: a context belongs to a project, and search
   * across the boundary would put one team's notes into another team's view.
   */
  searchContexts: (filters: { project_id: string; q: string; limit?: number }) => Promise<Context[]>
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
  // Coordination Plane (migrations/015)
  //
  // Blockers are project-scoped in the URL, not filtered by query only: the
  // handler checks the blocker against the project in the path and answers 404
  // on a mismatch, so a caller cannot read another project's blocker by getting
  // the id right and the path wrong.
  listBlockers: (projectId: string, status?: BlockerStatus) => Promise<Blocker[]>
  getBlocker: (projectId: string, id: string) => Promise<Blocker>
  /**
   * Closing a blocker needs a written answer, and the server rejects an empty
   * one: a blocker resolved with no resolution leaves the agent with nothing to
   * act on, which is the same state as leaving it open but with less visibility.
   */
  resolveBlocker: (projectId: string, id: string, input: ResolveBlockerInput) => Promise<Blocker>
  /** Closing without an answer needs a reason, for the same reason. */
  dismissBlocker: (projectId: string, id: string, input: DismissBlockerInput) => Promise<Blocker>
  // People & keys (gated by the admin scope)
  listPeople: () => Promise<Person[]>
  getPerson: (id: string) => Promise<Person>
  createPerson: (input: CreatePersonInput) => Promise<Person>
  /**
   * Revoked keys are hidden unless `include_revoked` is true — the handler
   * parses that flag strictly, so a malformed value is a 400 rather than a
   * silently-false one. The flag is stringified here because the query is
   * serialised, and `include_revoked=false` must stay distinguishable from
   * "not sent" only in intent, not in syntax.
   */
  listKeys: (filters?: { principal_id?: string; include_revoked?: boolean }) => Promise<APIKey[]>
  createKey: (input: CreateKeyInput) => Promise<IssuedAPIKey>
  revokeKey: (id: string) => Promise<APIKey>
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
      // Nuxt's $fetch on the server uses Nitro's devProxy; on the client the
      // proxy is also wired.
      //
      // `as T` is required because ofetch narrows the return to
      // TypedInternalResponse<...> which TypeScript cannot prove is assignable
      // to an unconstrained generic T. The call site controls T, so the cast
      // is sound — the alternative (an explicit cast to unknown first) adds
      // noise without adding safety.
      return await $fetch(path, { baseURL: apiBase.value, ...opts }) as T
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
      // Probe an *authenticated* endpoint, not /health.
      //
      // /health is intentionally unauthenticated so orchestrators and load
      // balancers can reach it. Probing it from the browser therefore cannot
      // tell the user whether their credential is accepted: with a bad key the
      // indicator would read "connected" while every page silently rendered
      // empty, which is a far more confusing failure than a clear
      // "disconnected".
      //
      // /api/projects is the cheapest authenticated route and exercises the
      // same path the app uses for everything else, credential injection
      // included.
      await $fetch('/projects', {
        baseURL: apiBase.value,
        method: 'GET',
        // An empty project list is a valid response; do not treat it as failure.
        timeout: 5000,
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
    apiBase,
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
    getAgentConfig:      (id, format) => request<AgentConfigResponse>(`/agents/${id}/config`,
      { params: toParams({ format }) }),
    mintAgentConnection: (id, input = {}) => request<AgentConnectionResponse>(`/agents/${id}/connection`,
      { method: 'POST', body: input }),

    listTasks:           (filters = {}) => request<Task[]>('/tasks', { params: toParams(filters) }),
    getTask:             (id)            => request<Task>(`/tasks/${id}`),
    createTask:          (input)         => request<Task>('/tasks', { method: 'POST', body: input }),
    updateTask:          (id, input)     => request<Task>(`/tasks/${id}`, { method: 'PUT', body: input }),
    updateTaskStatus:    (id, status)    => request<Task>(`/tasks/${id}/status`, { method: 'PUT', body: { status } }),
    reassignTask:        (id, assignedTo) => request<Task>(`/tasks/${id}/reassign`, { method: 'PUT', body: { assigned_to: assignedTo } }),
    deleteTask:          async (id) => { await request<void>(`/tasks/${id}`, { method: 'DELETE' }) },

    listContexts:        (filters = {}) => request<Context[]>('/contexts', { params: toParams(filters) }),
    searchContexts:      (filters)      => request<Context[]>('/contexts/search', { params: toParams({ q: filters.q, project_id: filters.project_id, limit: filters.limit }) }),
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
    deleteGlobalContext: async (id)      => { await request<void>(`/global_contexts/${id}`, { method: 'DELETE' }) },

    // Coordination Plane. The paths are project-scoped because the handler
    // checks the blocker against the project in the path and answers 404 on a
    // mismatch — a caller cannot read another project's blocker by having the
    // id right and the project wrong.
    listBlockers: (projectId, status) =>
      request<Blocker[]>(`/projects/${projectId}/blockers`, { params: status ? { status } : undefined }),
    getBlocker: (projectId, id) =>
      request<Blocker>(`/projects/${projectId}/blockers/${id}`),
    resolveBlocker: (projectId, id, input) =>
      request<Blocker>(`/projects/${projectId}/blockers/${id}/resolve`, { method: 'PUT', body: input }),
    dismissBlocker: (projectId, id, input) =>
      request<Blocker>(`/projects/${projectId}/blockers/${id}/dismiss`, { method: 'PUT', body: input }),

    // People & keys
    //
    // These two unwrap the list envelopes the handlers return — unlike every
    // other method above, /people and /keys answer `{"people": [...]}` and
    // `{"keys": [...]}` rather than a bare array. Unwrapping here keeps the
    // page components free of envelope plumbing.
    listPeople:         ()      => request<PeopleResponse>('/people').then(r => r.people),
    getPerson:          (id)    => request<Person>(`/people/${id}`),
    createPerson:       (input) => request<Person>('/people', { method: 'POST', body: input }),
    listKeys:           (filters = {}) => request<KeysResponse>('/keys', {
      params: toParams({
        principal_id: filters.principal_id,
        include_revoked: filters.include_revoked === undefined ? undefined : String(filters.include_revoked)
      })
    }).then(r => r.keys),
    createKey:          (input) => request<IssuedAPIKey>('/keys', { method: 'POST', body: input }),
    revokeKey:          (id)    => request<APIKey>(`/keys/${id}`, { method: 'DELETE' })
  }
}
