/**
 * Type contract mirroring the Go backend JSON shapes (cmd/server/main.go + internal/handlers).
 * Keep in sync with backend changes — no runtime import.
 *
 * Status vocabularies are declared as `as const` tuples first and the types are
 * derived from them, so a status cannot be added to the type and forgotten in
 * the dropdowns. The authoritative list is internal/models/status.go on the Go
 * side; these must match it exactly.
 */

// A type-only import, deliberately: the admin vocabulary is needed by the
// connection response shape, and a runtime import would pull it into every
// consumer of this file.
import type { IssuedAPIKey } from './admin'

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

/**
 * Agent tool vocabulary — which client an agent runs in.
 *
 * Mirrors `models.AllToolTypes` and the `tool_type` CHECK in
 * migrations/017_federation_identity.up.sql. Declared as a tuple before the type
 * is derived from it, the same way the status vocabularies above are, so a value
 * cannot reach the type without also having a label and a dropdown entry.
 *
 * The last four are recognised but not yet renderable: OpenCode, Windsurf and
 * both Copilot surfaces are Phase 3 and Phase 4 work, and 'a2a' is not an MCP
 * host at all. An agent on any of them still gets a working generic `.mcp.json`.
 */
export const AGENT_TOOL_TYPES = [
  'claude-code',
  'codex',
  'opencode',
  'copilot-cli',
  'copilot-vs',
  'cursor',
  'windsurf',
  'a2a',
  'generic-mcp',
  'unknown'
] as const
export type AgentToolType = (typeof AGENT_TOOL_TYPES)[number]

/**
 * Labels for the tool select.
 *
 * Declared as a `Record<AgentToolType, string>` on purpose: a new constant
 * without an entry here is a typecheck failure, not a blank dropdown row.
 */
export const AGENT_TOOL_TYPE_LABELS: Record<AgentToolType, string> = {
  'claude-code': 'Claude Code',
  codex: 'Codex CLI',
  opencode: 'OpenCode',
  'copilot-cli': 'Copilot CLI',
  'copilot-vs': 'Copilot in Visual Studio',
  cursor: 'Cursor',
  windsurf: 'Windsurf',
  a2a: 'Custom A2A agent',
  'generic-mcp': 'Generic MCP host',
  unknown: 'Unknown'
}

/**
 * Mirrors `models.ToolType.SupportsConfigGeneration`. An agent on anything else
 * falls back to `generic-mcp`, which is what most MCP hosts read — so this is
 * about which file is written, not about whether an agent can be connected.
 */
export const RENDERABLE_TOOL_TYPES: readonly AgentToolType[] = [
  'claude-code',
  'codex',
  'opencode',
  'cursor',
  'generic-mcp'
]

/** Config formats the server can render. Mirrors `models.AllConfigFormats`. */
export const CONFIG_FORMATS = ['claude', 'codex', 'cursor', 'opencode', 'generic'] as const
export type ConfigFormat = (typeof CONFIG_FORMATS)[number]

export const CONFIG_FORMAT_LABELS: Record<ConfigFormat, string> = {
  claude: 'Claude Code (.claude/mcp.json)',
  codex: 'Codex CLI (~/.codex/mcp.json)',
  cursor: 'Cursor (.cursor/mcp.json)',
  // OpenCode keeps MCP servers inside its one settings document, so there is no
  // dedicated mcp file to point at.
  opencode: 'OpenCode (opencode.json)',
  generic: 'Generic MCP host (.mcp.json)'
}

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

  /**
   * Which client this agent runs in — Claude Code, Codex, Cursor and so on.
   *
   * What makes the generated configuration correct rather than merely
   * plausible: the file layout is chosen from this, so "the file we told you to
   * write" and "what this agent is" cannot drift apart.
   */
  tool_type: AgentToolType

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

  /** The agent's own last report — progress, ETA, and whatever it chose to send. */
  heartbeat_payload: Record<string, unknown> | null

  /**
   * Progress and ETA from the newest heartbeat, joined in by the server.
   *
   * They come from the heartbeat rather than the agent row on purpose: a ping
   * every 30s would otherwise write the agent table forever for data that is
   * only ever read as "what did it last say".
   */
  last_progress_pct: number | null
  last_heartbeat_eta: string | null
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
  /**
   * Optional. Omitted means 'unknown', which the server normalises for us —
   * an older client that has never heard of the field keeps working.
   */
  tool_type?: AgentToolType
}

/**
 * One generated configuration file — mirrors `models.ConfigFile`.
 *
 * `content` never contains a credential. It references
 * `${env:AGENT_SHAKER_API_KEY}`, which is what makes a downloaded file safe to
 * commit; the plaintext is substituted by the browser at copy time from what
 * `IssuedAPIKey.secret` held a moment earlier.
 */
export interface AgentConfigFile {
  format: ConfigFormat
  /** Repository-relative path. Empty for user-level files. */
  filename: string
  /** Where a user-level file goes, e.g. `~/.codex/mcp.json`. */
  target?: string
  description: string
  content: string
  install?: string
}

/** GET /api/agents/{id}/config */
export interface AgentConfigResponse {
  agent: Agent
  format: ConfigFormat
  files: AgentConfigFile[]
  env_var: string
  endpoints: { mcp: string }
}

/**
 * POST /api/agents/{id}/connection — mirrors `models.AgentConnectionResponse`.
 *
 * `key.secret` is the only copy of the plaintext that will ever exist. The
 * server keeps the hash, so this value is gone the moment the modal closes.
 */
export interface AgentConnectionResponse {
  key: IssuedAPIKey
  agent: Agent
  format: ConfigFormat
  files: AgentConfigFile[]
}

/** POST /api/agents/{id}/connection request. Carries no scopes, deliberately. */
export interface AgentConnectionInput {
  format?: ConfigFormat
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
