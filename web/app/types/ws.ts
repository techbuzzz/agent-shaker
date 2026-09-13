/**
 * WebSocket event payload types.
 *
 * The Go server (internal/websocket/hub.go) emits JSON frames with a
 * `type` discriminator and a `payload`. See README for the supported
 * event types: task_update, agent_update, context_added.
 */

import type { Task, Agent, Context } from './api'

export type WsEventType = 'task_update' | 'agent_update' | 'context_added' | 'heartbeat'

interface BaseEvent<T extends WsEventType, P> {
  type: T
  project_id: string
  payload: P
}

export type TaskEvent = BaseEvent<'task_update', { task: Task; action: 'created' | 'updated' | 'deleted' }>
export type AgentEvent = BaseEvent<'agent_update', { agent: Agent; action: 'created' | 'updated' | 'status_changed' | 'deleted' }>
export type ContextEvent = BaseEvent<'context_added', { context: Context }>
export type HeartbeatEvent = BaseEvent<'heartbeat', { agent_id: string; status: string }>

export type WsEvent = TaskEvent | AgentEvent | ContextEvent | HeartbeatEvent

export type WsEventHandler<E extends WsEvent = WsEvent> = (event: E) => void
