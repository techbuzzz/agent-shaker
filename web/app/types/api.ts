/**
 * Type contract mirroring the Go backend JSON shapes (cmd/server/main.go + internal/handlers).
 * Keep in sync with backend changes — no runtime import.
 *
 * Status vocabularies are declared as `as const` tuples first and the types are
 * derived from them, so a status cannot be added to the type and forgotten in
 * the dropdowns. The authoritative list is internal/models/status.go on the Go
 * side; these must match it exactly.
 */

export type ProjectStatus = 'active' | 'archived'

export const AGENT_STATUSES = ['active', 'idle', 'blocked', 'offline'] as const
export type AgentStatus = (typeof AGENT_STATUSES)[number]

/**
 * Task status vocabulary.
 *
 * 'ready' joined the list with the Coordination Plane: a task whose
 * dependencies are all satisfied and which nobody has claimed. It is kept
 * separate from 'pending' on purpose — 'pending' means nobody picked it up,
 * 'ready' means it is claimable, and the dashboard reports on that difference.
 * Keep this in step with internal/models/status.go and the tasks_status_check
 * constraint in migrations/015_coordination_plane.up.sql.
 */
export const TASK_STATUSES = ['pending', 'blocked', 'ready', 'in_progress', 'done', 'cancelled'] as const
export type TaskStatus = (typeof TASK_STATUSES)[number]

/** Human-readable labels for the select menus, keyed by status value. */
export const AGENT_STATUS_LABELS: Record<AgentStatus, string> = {
  active: 'Active',
  idle: 'Idle',
  blocked: 'Blocked',
  offline: 'Offline'
}

export const TASK_STATUS_LABELS: Record<TaskStatus, string> = {
  pending: 'Pending',
  blocked: 'Blocked',
  ready: 'Ready',
  in_progress: 'In progress',
  done: 'Done',
  cancelled: 'Cancelled'
}

/** Builds a USelect items array from a status tuple and its label map. */
export function statusItems<S extends string>(
  values: readonly S[],
  labels: Record<S, string>
): { label: string; value: S }[] {
  return values.map((value) => ({ label: labels[value], value }))
}

export type AgentRole = 'pm' | 'backend' | 'frontend' | 'devops' | 'design' | 'qa' | (string & {})
export type TaskPriority = 'low' | 'medium' | 'high'
export type MilestoneStatus = 'planned' | 'active' | 'done' | 'dropped'
export type GlobalContextScope = 'global' | 'project'
export type ProjectRepoRole = 'code' | 'infra' | 'docs' | 'design'

export interface Project {
  id: string
  name: string
  description: string
  status: ProjectStatus
  created_at: string
  updated_at: string
}

export interface Agent {
  id: string
  project_id: string
  name: string
  role: AgentRole
  team: string
  status: AgentStatus
  last_seen: string
  created_at: string

  // --- Coordination Plane (migrations/015) ---

  /**
   * The deadline the server set when it last answered a heartbeat.
   *
   * null means the agent has never heartbeated — it was created through the UI
   * and has no runtime of its own to ping from. The UI must not show those as
   * stale: an agent that never claimed a lease cannot lose one.
   */
  lease_expires_at: string | null

  /** What the agent last reported working on, or null. */
  current_task_id: string | null

  /** The agent's own last report — progress, ETA, whatever it chose to send. */
  heartbeat_payload: Record<string, unknown> | null
}

export interface Task {
  id: string
  project_id: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  created_by: string
  assigned_to: string
  output: string
  milestone_id?: string
  tags: string[]
  created_at: string
  updated_at: string

  // --- Coordination Plane (migrations/015) ---

  /** The task this one was handed off from, if any. */
  predecessor_task_id: string | null

  /**
   * What the predecessor left behind: the API shape, the branch, the gotcha.
   * Empty on any task that was not created by a handoff, which is most of them.
   */
  handoff_context: string
}

// --- Coordination Plane types (migrations/015) ---

export const BLOCKER_TYPES = ['question', 'dependency', 'resource', 'decision', 'approval'] as const
export type BlockerType = (typeof BLOCKER_TYPES)[number]

export const BLOCKER_TYPE_LABELS: Record<BlockerType, string> = {
  question: 'Question',
  dependency: 'Dependency',
  resource: 'Resource',
  decision: 'Decision',
  approval: 'Approval'
}

export const BLOCKER_STATUSES = ['open', 'in_review', 'resolved', 'dismissed'] as const
export type BlockerStatus = (typeof BLOCKER_STATUSES)[number]

export const BLOCKER_STATUS_LABELS: Record<BlockerStatus, string> = {
  open: 'Open',
  in_review: 'In review',
  resolved: 'Resolved',
  dismissed: 'Dismissed'
}

export interface Blocker {
  id: string
  task_id: string
  agent_id: string
  project_id: string
  blocker_type: BlockerType
  title: string
  description: string
  status: BlockerStatus
  /** The answer an agent can act on. null until the blocker is closed. */
  resolution: string | null
  resolved_by: string | null
  resolved_at: string | null
  created_at: string
  updated_at: string
}

export interface ResolveBlockerInput {
  resolution: string
}

/**
 * A dismissal always carries a reason: it is the only thing that tells an agent
 * to stop re-raising the same blocker tomorrow.
 */
export interface DismissBlockerInput {
  reason: string
}

/** One edge of the task DAG, as the MCP `get_task_dependencies` tool returns it. */
export interface TaskDependency {
  task_id: string
  title?: string
  status?: TaskStatus
}

export interface Context {
  id: string
  project_id: string
  agent_id: string
  task_id?: string
  title: string
  content: string
  tags: string[]
  created_at: string
}

export interface Standup {
  id: string
  project_id: string
  agent_id: string
  date: string
  yesterday: string
  today: string
  blockers: string
  created_at: string
}

export interface Heartbeat {
  id: string
  agent_id: string
  recorded_at: string
  status: AgentStatus
}

// Mesh-app additions — Phase 1/2/3 of the agent-shaker mesh plan.

export interface Milestone {
  id: string
  project_id: string
  title: string
  description: string
  status: MilestoneStatus
  target_date?: string
  created_by: string
  created_at: string
  updated_at: string
}

export interface CreateMilestoneInput {
  project_id: string
  title: string
  description?: string
  status?: MilestoneStatus
  target_date?: string
  created_by: string
}

export interface UpdateMilestoneStatusInput {
  status: MilestoneStatus
  description?: string
}

export interface ProjectRepo {
  id: string
  project_id: string
  url: string
  branch: string
  role: ProjectRepoRole | string
  agent_id?: string
  created_at: string
  updated_at: string
}

export interface CreateProjectRepoInput {
  project_id: string
  url: string
  branch?: string
  role?: ProjectRepoRole | string
  agent_id?: string
}

export interface GlobalContext {
  id: string
  scope: GlobalContextScope
  project_id?: string
  agent_id: string
  title: string
  content: string
  tags: string[]
  created_at: string
  updated_at: string
}

export interface CreateGlobalContextInput {
  scope: GlobalContextScope
  project_id?: string
  agent_id: string
  title: string
  content: string
  tags?: string[]
}

export interface UpdateGlobalContextInput {
  content: string
  tags?: string[]
}

export interface DashboardStats {
  projects: { total: number; active: number; archived: number }
  agents: { total: number; active: number; idle: number; blocked: number; offline: number }
  tasks: { total: number; pending: number; in_progress: number; done: number; blocked: number; cancelled: number }
  contexts: { total: number }
}

// Input shapes (for mutations)
export interface CreateProjectInput {
  name: string
  description: string
  status?: ProjectStatus
}
export interface UpdateProjectStatusInput {
  status: ProjectStatus
}

export interface CreateAgentInput {
  project_id: string
  name: string
  role: AgentRole
  team: string
  status?: AgentStatus
}
export interface UpdateAgentStatusInput {
  status: AgentStatus
}

export interface CreateTaskInput {
  project_id: string
  title: string
  description: string
  priority?: TaskPriority
  created_by?: string
  assigned_to?: string
  milestone_id?: string
  tags?: string[]
}
export interface UpdateTaskInput extends Partial<CreateTaskInput> {
  status?: TaskStatus
  output?: string
}
export interface UpdateTaskStatusInput {
  status: TaskStatus
}
export interface ReassignTaskInput {
  assigned_to: string
}

export interface CreateContextInput {
  project_id: string
  agent_id: string
  task_id?: string
  title: string
  content: string
  tags?: string[]
}

export interface CreateStandupInput {
  project_id: string
  agent_id: string
  date: string
  yesterday: string
  today: string
  blockers: string
}

export interface CreateHeartbeatInput {
  agent_id: string
  status?: AgentStatus
}

/**
 * Common error shape returned by the Go backend.
 * @see internal/httpx/errors.go
 */
export interface ApiErrorBody {
  error: string
  message?: string
  details?: Record<string, unknown>
}
