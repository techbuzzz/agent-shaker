/**
 * WebSocket event payload types.
 *
 * The Go server (internal/websocket/hub.go) emits JSON frames with a
 * `type` discriminator and a `payload`. See README for the supported
 * event types. The mesh-app additions (Phase 1/2/3 of the agent-shaker
 * mesh plan) introduced new event types for milestones, project repos,
 * global contexts, and the `task_added` shortcut.
 */

import type { Task, Agent, Context, Milestone, ProjectRepo, GlobalContext } from './api'

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

export type WsEventHandler<E extends WsEvent = WsEvent> = (event: E) => void
