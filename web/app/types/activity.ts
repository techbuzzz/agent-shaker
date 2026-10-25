/**
 * The append-only journal — the wire shapes of what the server records.
 *
 * The journal is read-only by construction: there is no endpoint that can
 * change a row. The interface mirrors that — it shows the trail and the
 * anchor, and offers nothing that edits it.
 */

/** One journalled change. */
export interface AuditEvent {
  id: number
  occurred_at: string
  actor_type: string
  actor_id: string | null
  project_id: string | null
  entity_type: string
  entity_id: string | null
  /** What happened, in the domain's vocabulary: task.claimed, agent.stale. */
  action: string
  /** The entity as it was, when the publisher knew. */
  before: Record<string, unknown> | null
  /** The entity as it became. */
  after: Record<string, unknown> | null
  trace_id: string
  request_id: string
  /** The hash of the row before this one. */
  prev_hash: string
  /** This row's hash, over its own content and its predecessor's. */
  hash: string
}

/**
 * The published head of a project's chain.
 *
 * The anchor is the value an operator pins outside the database: a trail
 * rewritten inside the database cannot be rewritten with it, because the head
 * it would have to agree with lives somewhere else.
 */
export interface AuditChainHead {
  project_id: string
  head_hash: string
  row_id: number
  recorded_at: string
}

export interface AuditEventsResponse {
  events: AuditEvent[]
}
