/**
 * Type contract mirroring the Go backend JSON shapes (cmd/server/main.go + internal/handlers).
 * Keep in sync with backend changes — no runtime import.
 */

export type ProjectStatus = 'active' | 'archived'
export type AgentStatus = 'active' | 'idle' | 'offline'
export type AgentRole = 'backend' | 'frontend' | 'devops' | 'design' | 'qa' | (string & {})
export type TaskStatus = 'pending' | 'in_progress' | 'blocked' | 'done' | 'cancelled'
export type TaskPriority = 'low' | 'medium' | 'high'

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
  created_at: string
  updated_at: string
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

export interface DashboardStats {
  projects: { total: number; active: number; archived: number }
  agents: { total: number; active: number; idle: number; offline: number }
  tasks: { total: number; pending: number; in_progress: number; done: number; blocked: number }
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
