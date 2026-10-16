/**
 * WebSocket event payload types.
 *
 * The Go server (internal/websocket/hub.go) emits JSON frames with a
 * `type` discriminator and a `payload`. See README for the supported
 * event types. The mesh-app additions (Phase 1/2/3 of the agent-shaker
 * mesh plan) introduced new event types for milestones, project repos,
 * global contexts, and the `task_added` shortcut.
 */

import type { Task, Agent, Context, Milestone, ProjectRepo, GlobalContext, Blocker, BlockerType } from './api'

export type WsEventType =
  | 'task_update'
  | 'task_added'
  | 'agent_update'
  | 'context_added'
  | 'milestone_added'
  | 'milestone_updated'
  | 'milestone_deleted'
  | 'project_repo_added'
  | 'project_repo_updated'
  | 'project_repo_deleted'
  | 'global_context_added'
  | 'global_context_updated'
  | 'global_context_deleted'
  | 'heartbeat'
  // Coordination Plane (migrations/015). Each one is a state change a human has
  // to react to, which is why they are broadcast at all: nothing here is
  // recoverable by polling.
  | 'blocker_raised'
  | 'blocker_resolved'
  | 'task_ready'
  | 'agent_gone_stale'
  | 'agent_heartbeat'

interface BaseEvent<T extends WsEventType, P> {
  type: T
  project_id: string
  payload: P
}

export type TaskEvent = BaseEvent<'task_update', { task: Task; action: 'created' | 'updated' | 'deleted' }>
export type TaskAddedEvent = BaseEvent<'task_added', Task>
export type AgentEvent = BaseEvent<'agent_update', { agent: Agent; action: 'created' | 'updated' | 'status_changed' | 'deleted' }>
export type ContextEvent = BaseEvent<'context_added', { context: Context }>
export type MilestoneAddedEvent = BaseEvent<'milestone_added', Milestone>
export type MilestoneUpdatedEvent = BaseEvent<'milestone_updated', Milestone>
export type MilestoneDeletedEvent = BaseEvent<'milestone_deleted', { id: string }>
export type ProjectRepoAddedEvent = BaseEvent<'project_repo_added', ProjectRepo>
export type ProjectRepoUpdatedEvent = BaseEvent<'project_repo_updated', ProjectRepo>
export type ProjectRepoDeletedEvent = BaseEvent<'project_repo_deleted', { id: string }>
export type GlobalContextAddedEvent = BaseEvent<'global_context_added', GlobalContext>
export type GlobalContextUpdatedEvent = BaseEvent<'global_context_updated', GlobalContext>
export type GlobalContextDeletedEvent = BaseEvent<'global_context_deleted', { id: string }>
export type HeartbeatEvent = BaseEvent<'heartbeat', { agent_id: string; status: string }>

// --- Coordination Plane events -------------------------------------------
//
// blocker_raised and blocker_resolved carry the whole blocker rather than an
// id: the panel renders the card from the event, so an operator who is looking
// at the screen when an agent gets stuck sees it appear without waiting for a
// refetch.

export interface BlockerRaisedEvent extends BaseEvent<
  'blocker_raised',
  {
    blocker_id: string
    task_id: string
    agent_id: string
    agent_name: string
    blocker_type: BlockerType
    title: string
    description?: string
  }
> {}

/**
 * Emitted by BOTH resolve and dismiss.
 *
 * `status` is what tells them apart: a resolved blocker carries an answer the
 * agent acts on, a dismissed one carries only a reason it should stop asking.
 * Without it a client files every dismissal under "Answer".
 */
export interface BlockerResolvedEvent extends BaseEvent<
  'blocker_resolved',
  {
    blocker_id: string
    task_id: string
    agent_id: string
    status: 'resolved' | 'dismissed'
    resolution: string
  }
> {}

export interface TaskReadyEvent extends BaseEvent<
  'task_ready',
  { task_id: string; title: string; status: string; assigned_to: string }
> {}

/**
 * An agent whose lease expired. `reassigned_task` lists the tasks that went
 * back to the backlog — the reason this event matters to an operator, because
 * "an agent went offline" is only interesting if work was stranded.
 */
export interface AgentGoneStaleEvent extends BaseEvent<
  'agent_gone_stale',
  { agent_id: string; agent_name: string; reassigned_task: string[] }
> {}

export interface AgentHeartbeatEvent extends BaseEvent<
  'agent_heartbeat',
  {
    agent_id: string
    agent_name: string
    status: string
    task_id: string | null
    progress_pct: number | null
    /** When the agent expects to finish what it reported. */
    eta: string | null
    next_heartbeat_before: string | null
  }
> {}

export type WsEvent =
  | TaskEvent
  | TaskAddedEvent
  | AgentEvent
  | ContextEvent
  | MilestoneAddedEvent
  | MilestoneUpdatedEvent
  | MilestoneDeletedEvent
  | ProjectRepoAddedEvent
  | ProjectRepoUpdatedEvent
  | ProjectRepoDeletedEvent
  | GlobalContextAddedEvent
  | GlobalContextUpdatedEvent
  | GlobalContextDeletedEvent
  | HeartbeatEvent
  | BlockerRaisedEvent
  | BlockerResolvedEvent
  | TaskReadyEvent
  | AgentGoneStaleEvent
  | AgentHeartbeatEvent

/** The blocker kinds of payload the panel can build a card from. */
export type BlockerLike = Blocker

/**
 * Narrow the union to the single variant carrying a given `type` tag.
 *
 * This is what makes `on('task_update', e => e.payload.task.title)` typecheck:
 * the handler is generic over the event *name*, so `Extract` resolves the
 * payload to `{ task: Task; action: ... }` instead of the whole union.
 */
export type WsEventFor<T extends WsEventType> = Extract<WsEvent, { type: T }>

/**
 * A listener for a specific event name, or for any event when `T` is the full
 * `WsEventType` set (the `'*'` wildcard case).
 */
export type WsEventHandler<T extends WsEventType = WsEventType> =
  (event: WsEventFor<T>) => void
