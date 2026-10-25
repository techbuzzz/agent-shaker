// Package events defines the domain event vocabulary and the broker that
// carries those events to their subscribers.
//
// A domain event and a WebSocket broadcast are two views of the same change,
// and they are deliberately not the same thing. The broadcast is the browser's
// notification: it carries whatever shape the UI happens to want, under
// whatever name the UI happens to use. A domain event is a typed statement
// about what happened, with one canonical name per change.
//
// The hook layer consumes domain events. A hook that had to parse a UI
// payload to find out which task completed would break every time the
// frontend changed a field, and would silently stop firing on the changes
// nobody remembered to update it for.
package events

import (
	"time"

	"github.com/google/uuid"
)

// The canonical event vocabulary.
//
// One name per domain change. The dotted form is not decoration: it groups
// related events in a log, and it makes the difference between "one event with
// two names" (task_update and task_updated, which the server used to emit
// side by side) visible in the first character.
const (
	TaskClaimed      = "task.claimed"
	TaskCompleted    = "task.completed"
	TaskHandoff      = "task.handoff"
	TaskReassigned   = "task.reassigned"
	TaskBlocked      = "task.blocked"
	TaskReady        = "task.ready"
	TaskUpdated      = "task.updated"
	BlockerRaised    = "blocker.raised"
	BlockerResolved  = "blocker.resolved"
	AgentStale       = "agent.stale"
	AgentHeartbeat   = "agent.heartbeat"
	ContextPublished = "context.published"
)

// Entity and actor types carried on an event.
const (
	EntityTask    = "task"
	EntityAgent   = "agent"
	EntityBlocker = "blocker"
	EntityContext = "context"

	ActorAgent   = "agent"
	ActorHuman   = "human"
	ActorService = "service"
)

// The payload keys publishers set.
//
// The list is short on purpose. The hook templates read these by name, so
// every key here is a published interface: renaming one changes what an
// operator's hook sees.
const (
	PayloadTitle     = "title"
	PayloadStatus    = "status"
	PayloadOutput    = "output"
	PayloadActorName = "actor_name"
	PayloadActorRole = "actor_role"
)

// Event is one domain state change.
//
// Payload carries the change-specific detail. The keys are stable per event
// type and are documented with the publisher, because the hook templates read
// them by name.
type Event struct {
	Type       string
	ProjectID  uuid.UUID
	EntityType string
	EntityID   uuid.UUID
	ActorType  string
	ActorID    uuid.UUID
	Payload    map[string]any
	OccurredAt time.Time
}

// Options carries the optional parts of an event.
//
// A struct rather than a fourth and fifth positional argument: the fields are
// all optional, and a caller that only wants to set the entity should not
// have to count arguments to find out which one it is.
type Options struct {
	EntityType string
	EntityID   uuid.UUID
	ActorType  string
	ActorID    uuid.UUID
	Payload    map[string]any
}

// New builds an event with the given type and project.
//
// The payload map is copied, so a caller can reuse the map it passed in
// without the event changing under it.
func New(eventType string, projectID uuid.UUID, opts Options) Event {
	payload := make(map[string]any, len(opts.Payload))
	for k, v := range opts.Payload {
		payload[k] = v
	}

	return Event{
		Type:       eventType,
		ProjectID:  projectID,
		EntityType: opts.EntityType,
		EntityID:   opts.EntityID,
		ActorType:  opts.ActorType,
		ActorID:    opts.ActorID,
		Payload:    payload,
		OccurredAt: time.Now().UTC(),
	}
}

// IsKnown reports whether the event type is part of the canonical vocabulary.
//
// The hook layer uses it to reject a hook bound to an event that can never
// fire, which is a configuration error worth reporting at creation time
// rather than discovering on the first run.
func IsKnown(eventType string) bool {
	switch eventType {
	case TaskClaimed, TaskCompleted, TaskHandoff, TaskReassigned,
		TaskBlocked, TaskReady, TaskUpdated,
		BlockerRaised, BlockerResolved,
		AgentStale, AgentHeartbeat,
		ContextPublished:
		return true
	default:
		return false
	}
}

// KnownTypes returns every event type a hook may bind to, in a stable order.
func KnownTypes() []string {
	return []string{
		TaskClaimed,
		TaskCompleted,
		TaskHandoff,
		TaskReassigned,
		TaskBlocked,
		TaskReady,
		BlockerResolved,
		AgentStale,
		ContextPublished,
	}
}
