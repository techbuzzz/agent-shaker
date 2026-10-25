package mcp

import (
	"context"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/events"
)

// publishTaskEvent publishes a domain event about a task.
//
// The event is the hook layer's view of the same change the WebSocket
// broadcast reports. They are not interchangeable: the broadcast carries
// whatever shape the browser wants, while the event carries the vocabulary a
// command can bind to — "task.claimed" means one thing, and a hook that
// had to infer it from a generic task update would be guessing.
//
// The ids arrive as strings because that is how the MCP context carries them,
// and parsing them here keeps every call site free of error handling for a
// value the tool has already validated.
func (h *MCPHandler) publishTaskEvent(ctx MCPContext, eventType, projectID, taskID, title, status string) {
	events.Send(ctx.Context, h.broker, events.New(eventType, parseUUID(projectID), events.Options{
		EntityType: events.EntityTask,
		EntityID:   parseUUID(taskID),
		ActorType:  events.ActorAgent,
		ActorID:    parseUUID(ctx.AgentID),
		Payload: map[string]any{
			events.PayloadTitle:  title,
			events.PayloadStatus: status,
		},
	}))
}

// publishBlockerEvent publishes a domain event about a blocker.
//
// It is published as both blocker.raised and task.blocked: the first names
// what happened, the second names what it means for the work. A hook that
// only cares about "the team is stuck" binds to the second.
func (h *MCPHandler) publishBlockerEvent(ctx MCPContext, projectID, taskID, blockerID, title, blockerType string) {
	events.Send(ctx.Context, h.broker, events.New(events.BlockerRaised, parseUUID(projectID), events.Options{
		EntityType: events.EntityBlocker,
		EntityID:   parseUUID(blockerID),
		ActorType:  events.ActorAgent,
		ActorID:    parseUUID(ctx.AgentID),
		Payload: map[string]any{
			events.PayloadTitle:  title,
			events.PayloadStatus: blockerType,
		},
	}))

	events.Send(ctx.Context, h.broker, events.New(events.TaskBlocked, parseUUID(projectID), events.Options{
		EntityType: events.EntityTask,
		EntityID:   parseUUID(taskID),
		ActorType:  events.ActorAgent,
		ActorID:    parseUUID(ctx.AgentID),
		Payload: map[string]any{
			events.PayloadTitle:  title,
			events.PayloadStatus: "blocked",
		},
	}))
}

// parseUUID parses an id, returning the nil uuid when it is not one.
//
// A malformed id is not an error here: the caller has already validated it
// by the time these helpers run, and an event with an unknown actor is still
// a better record than a panic.
func parseUUID(id string) uuid.UUID {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil
	}
	return parsed
}

// publishServiceEvent publishes a domain event for a path that carries no
// request context.
//
// The REST and administrative paths have no MCP context to borrow one from,
// and an event with a service actor is more honest than one with a borrowed
// request context that has already been answered.
func (h *MCPHandler) publishServiceEvent(eventType, projectID, entityType, entityID, title, status string) {
	events.Send(context.Background(), h.broker, events.New(eventType, parseUUID(projectID), events.Options{
		EntityType: entityType,
		EntityID:   parseUUID(entityID),
		ActorType:  events.ActorService,
		Payload: map[string]any{
			events.PayloadTitle:  title,
			events.PayloadStatus: status,
		},
	}))
}
