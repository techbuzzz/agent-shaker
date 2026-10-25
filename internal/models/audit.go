package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditEvent is one row of the append-only journal.
//
// Before and After are the entity as it was and as it became. Either may be
// nil: a creation has no before, and an event that carries only "this
// happened" has no diff to record.
//
// PrevHash and Hash are the chain. Hash covers the row's own content and its
// predecessor's hash, so editing or removing any row breaks every hash after
// it.
type AuditEvent struct {
	ID         int64          `json:"id"`
	OccurredAt time.Time      `json:"occurred_at"`
	ActorType  string         `json:"actor_type"`
	ActorID    *uuid.UUID     `json:"actor_id"`
	ProjectID  *uuid.UUID     `json:"project_id"`
	EntityType string         `json:"entity_type"`
	EntityID   *uuid.UUID     `json:"entity_id"`
	Action     string         `json:"action"`
	Before     map[string]any `json:"before"`
	After      map[string]any `json:"after"`
	TraceID    string         `json:"trace_id"`
	RequestID  string         `json:"request_id"`
	PrevHash   string         `json:"prev_hash"`
	Hash       string         `json:"hash"`
}

// AuditChainHead is the published head of a project's chain.
//
// It is the anchor: the value an operator pins outside the database, so that
// a trail rewritten inside the database cannot be rewritten with it.
type AuditChainHead struct {
	ProjectID  uuid.UUID `json:"project_id"`
	HeadHash   string    `json:"head_hash"`
	RowID      int64     `json:"row_id"`
	RecordedAt time.Time `json:"recorded_at"`
}
