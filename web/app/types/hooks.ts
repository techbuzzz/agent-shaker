/**
 * Hook layer contract — the wire shapes of the server-side hook runner.
 *
 * A hook is a command the server runs when a domain event happens. The command
 * is an argument vector, never a shell string: the server hands the process the
 * arguments it stored and never involves a shell, so there is no quoting step
 * where a task title could become a command injection. The UI follows the same
 * rule — it edits one argument at a time and never offers a "command line"
 * field that could be mistaken for a shell command.
 *
 * Mirrors the Go side: migrations/019_hooks.up.sql (the tables and their
 * constraints) and internal/events/events.go (the event vocabulary).
 */

/**
 * The canonical event vocabulary a hook may bind to.
 *
 * A closed set on purpose: a hook bound to an event the server can never
 * publish is a hook that silently never fires, so the server rejects it at
 * creation and the UI offers nothing else. The order matches
 * events.KnownTypes() on the Go side.
 */
export const HOOK_EVENTS = [
  'task.claimed',
  'task.completed',
  'task.handoff',
  'task.reassigned',
  'task.blocked',
  'task.ready',
  'blocker.resolved',
  'agent.stale',
  'context.published'
] as const

export type HookEvent = (typeof HOOK_EVENTS)[number]

/** Human-readable labels for the event select, keyed by event name. */
export const HOOK_EVENT_LABELS: Record<HookEvent, string> = {
  'task.claimed': 'Task claimed',
  'task.completed': 'Task completed',
  'task.handoff': 'Task handed off',
  'task.reassigned': 'Task reassigned',
  'task.blocked': 'Task blocked',
  'task.ready': 'Task ready',
  'blocker.resolved': 'Blocker resolved',
  'agent.stale': 'Agent went stale',
  'context.published': 'Context published'
}

/** Builds a USelect items array from the event vocabulary. */
export function hookEventItems(): { label: string; value: HookEvent }[] {
  return HOOK_EVENTS.map((value) => ({ label: HOOK_EVENT_LABELS[value], value }))
}

/**
 * The normalised outcome of a run.
 *
 * The raw exit code is kept beside it because the two ecosystems disagree:
 * `mcode exec` uses 2 for an invalid invocation and Claude Code uses 2 for a
 * blocking decision.
 */
export const HOOK_RUN_STATUSES = ['succeeded', 'failed', 'timeout', 'cancelled', 'blocked'] as const
export type HookRunStatus = (typeof HOOK_RUN_STATUSES)[number]

export const HOOK_RUN_STATUS_LABELS: Record<HookRunStatus, string> = {
  succeeded: 'Succeeded',
  failed: 'Failed',
  timeout: 'Timed out',
  cancelled: 'Cancelled',
  blocked: 'Blocked'
}

/**
 * Optional narrowing: a regular expression over one named field of the entity.
 * `null` matches every event of this type.
 */
export interface HookMatcher {
  field: string
  pattern: string
}

/**
 * How the event reaches the process.
 *
 * `event` writes the whole event as JSON on stdin, which is what a plain script
 * wants. `prompt` renders a template instead, which is what a headless agent
 * CLI wants — it reads a prompt, not an event.
 */
export type HookStdinMode = 'event' | 'prompt'

export interface HookStdin {
  mode: HookStdinMode
  /** Only meaningful in `prompt` mode. */
  template?: string
}

/**
 * What to do with the command's stdout.
 *
 * `context` writes the output back into the domain as a context on the
 * entity's project, which is what turns a hook from a notification into a
 * reviewer whose verdict the worker reads on its next turn.
 */
export interface HookPost {
  kind: 'context'
  title: string
}

export interface Hook {
  id: string
  project_id: string
  event: HookEvent
  matcher: HookMatcher | null
  /** An argv array. Never a shell string. */
  command: string[]
  /**
   * The environment the command runs with.
   *
   * A value written as `{secret:NAME}` is a reference to a secret the server
   * resolves at run time; the resolved value is injected into the process and
   * redacted from the captured output, and it is never stored here.
   *
   * The UI does not edit this map, but it round-trips it on update: the
   * server's update path replaces the map with whatever the request body
   * carries, so omitting it would silently drop a hook's secret references
   * the first time somebody edited its command.
   */
  env?: Record<string, string> | null
  stdin: HookStdin
  post: HookPost | null
  timeout_seconds: number
  enabled: boolean
  created_by: string | null
  created_at: string
  updated_at: string
}

export interface HookRun {
  id: string
  /** Null once the hook it belongs to has been deleted — the run outlives it. */
  hook_id: string | null
  delivery_id: string
  event: HookEvent
  entity_type: string
  entity_id: string | null
  status: HookRunStatus
  exit_code: number | null
  stdout: string
  stderr: string
  duration_ms: number | null
  started_at: string
  finished_at: string | null
}

/**
 * The body of a create request.
 *
 * `project_id` is not here: the project is in the path
 * (`POST /api/projects/{project_id}/hooks`), and sending it twice invites the
 * two to disagree.
 */
export interface CreateHookInput {
  event: HookEvent
  matcher: HookMatcher | null
  command: string[]
  /** Only sent when the hook already has environment entries to preserve. */
  env?: Record<string, string>
  stdin: HookStdin
  post: HookPost | null
  timeout_seconds: number
  enabled: boolean
}

/** PATCH body: every field is optional, and only what is sent is changed. */
export type UpdateHookInput = Partial<CreateHookInput>

/** The envelopes the hook endpoints answer with. */
export interface HooksResponse {
  hooks: Hook[]
}
export interface HookResponse {
  hook: Hook
}
export interface HookRunsResponse {
  runs: HookRun[]
}
export interface HookRunResponse {
  run: HookRun
}

/**
 * Whether the server runs this project's hooks.
 *
 * The switch is per project. Creating a hook turns it on, and this is how the
 * interface learns a state it did not set itself — an operator who turned
 * hooks off for a project needs to see that, or the next hook they create
 * will appear to do nothing.
 */
export interface HookSettingsResponse {
  project_id: string
  enabled: boolean
}

/**
 * The default and the ceiling for `timeout_seconds`.
 *
 * Both come from the CHECK constraint on hooks.timeout_seconds in
 * migrations/019_hooks.up.sql: a hook cannot be configured to pin a worker
 * for an hour.
 */
export const HOOK_DEFAULT_TIMEOUT_SECONDS = 120
export const HOOK_MAX_TIMEOUT_SECONDS = 3600

/**
 * The placeholders the server substitutes.
 *
 * They are substituted per argument element as well as in the prompt template,
 * so a placeholder that resolves to a path containing a space is still one
 * argument — the substitution never re-splits the vector.
 */
export const HOOK_PLACEHOLDERS: readonly { token: string; description: string }[] = [
  { token: '{{project.id}}', description: 'Id of the project the event belongs to' },
  { token: '{{project.name}}', description: 'Name of the project' },
  { token: '{{project.repo_path}}', description: 'Path of a registered repository of the project' },
  { token: '{{entity.id}}', description: 'Id of the entity the event is about' },
  { token: '{{entity.type}}', description: 'Type of the entity, for example task' },
  { token: '{{entity.title}}', description: 'Title of the entity' },
  { token: '{{entity.status}}', description: 'Status of the entity' },
  { token: '{{entity.output}}', description: 'Output the entity produced' },
  { token: '{{actor.id}}', description: 'Id of the agent or person that caused the event' },
  { token: '{{actor.name}}', description: 'Name of the actor' },
  { token: '{{actor.role}}', description: 'Role of the actor' },
  { token: '{{event}}', description: 'Name of the event, for example task.completed' },
  { token: '{{delivery_id}}', description: 'Id of the delivery this run belongs to' },
  { token: '{{occurred_at}}', description: 'When the event happened' }
]
