// Package audit writes the append-only journal.
//
// The journal is the project's evidence layer: a record of what happened,
// written by the thing that happened it, that nobody can quietly rewrite
// afterwards. Everything built on top of it — cost accounting, approvals,
// meaningful tracing — needs that property first, so it is here before those.
//
// Two rules shape the package:
//
//   - A journal write must never fail the business operation that caused it.
//     The trail is worth less than the work, and a trail that can take the
//     product down is a liability rather than an audit.
//   - The journal is append-only. There is no update and no delete, anywhere,
//     and a test pins that.
package audit

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/events"
)

// Event is one journal entry.
type Event struct {
	// Action is what happened, in the domain's own vocabulary:
	// task.claimed, context.published, agent.stale.
	Action string

	// ProjectID is the project the change belongs to. The zero uuid means
	// "no project", which the recorder treats as its own chain.
	ProjectID uuid.UUID

	// EntityType and EntityID are what the change was to.
	EntityType string
	EntityID   uuid.UUID

	// ActorType and ActorID are who did it. The zero values mean nobody —
	// a reaper releasing a stale agent's tasks has no actor to name.
	ActorType string
	ActorID   uuid.UUID

	// Before and After are the entity as it was and as it became. Either
	// may be nil: a creation has no before, and an event that carries only
	// "this happened" has no diff to record.
	Before map[string]any
	After  map[string]any

	// TraceID follows one request across the system; RequestID is the
	// server's own handle on it.
	TraceID   string
	RequestID string

	// OccurredAt is when it happened. The zero value means now.
	OccurredAt time.Time
}

// Recorder writes events to the journal.
type Recorder interface {
	// Record writes one event. An error is logged and swallowed by the
	// callers: the journal is not allowed to fail the operation that caused
	// the event.
	Record(ctx context.Context, ev Event) error
}

// Subscriber records the domain events the broker publishes.
//
// It is the bridge between the two halves of the coordination plane: the
// broker knows what happened, and the journal is where it is kept.
type Subscriber struct {
	recorder Recorder
}

// NewSubscriber returns a broker subscriber that writes to the recorder.
func NewSubscriber(recorder Recorder) *Subscriber {
	return &Subscriber{recorder: recorder}
}

// HandleEvent implements events.Subscriber.
func (s *Subscriber) HandleEvent(ctx context.Context, ev events.Event) {
	if s == nil || s.recorder == nil {
		return
	}

	record := Event{
		Action:     ev.Type,
		ProjectID:  ev.ProjectID,
		EntityType: ev.EntityType,
		EntityID:   ev.EntityID,
		ActorType:  ev.ActorType,
		ActorID:    ev.ActorID,
		After:      ev.Payload,
		OccurredAt: ev.OccurredAt,
	}

	if err := s.recorder.Record(ctx, record); err != nil {
		// Deliberately swallowed here. The caller of the broker is a request
		// handler, and a journal that can fail a request is not a journal, it
		// is an outage.
		return
	}
}
