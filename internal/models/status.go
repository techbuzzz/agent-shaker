package models

import "strings"

// Status vocabularies live here, in one place, because they used to be
// hardcoded in at least three places that had drifted apart: the REST
// validator accepted one set, the MCP tools another, and the database
// accepted anything. An agent could therefore be told by our own generated
// instructions to send a status that the REST API rejected.
//
// Every surface — REST validator, MCP tool schemas, MCP runtime validation,
// SQL queries, and the database CHECK constraints added in migration 014 —
// must derive from the slices below. Adding a status means adding it here
// and in the migration, never at a call site.

// TaskStatus is the lifecycle state of a coordination task (the `tasks`
// table). This is deliberately distinct from the A2A protocol's own task
// status vocabulary in internal/a2a/models, which is fixed by the protocol.
type TaskStatus string

const (
	StatusTaskPending    TaskStatus = "pending"
	StatusTaskInProgress TaskStatus = "in_progress"
	StatusTaskBlocked    TaskStatus = "blocked"
	StatusTaskDone       TaskStatus = "done"
	StatusTaskCancelled  TaskStatus = "cancelled"

	// StatusTaskReady is a task whose dependencies are all satisfied and which
	// nobody has claimed yet.
	//
	// Added by the Coordination Plane (migration 015). It is deliberately not
	// folded into "pending": "pending" means nobody has picked this up, "ready"
	// means it is claimable. The reaper returns work to pending and the
	// dependency resolver promotes work to ready, and those two facts are what
	// the dashboard reports on.
	StatusTaskReady TaskStatus = "ready"
)

// AllTaskStatuses is the authoritative task vocabulary, in display order.
// Used to build MCP tool enums and error messages so they cannot drift.
var AllTaskStatuses = []TaskStatus{
	StatusTaskPending,
	StatusTaskBlocked,
	StatusTaskReady,
	StatusTaskInProgress,
	StatusTaskDone,
	StatusTaskCancelled,
}

// IsValidTaskStatus reports whether s is a member of the task vocabulary.
func IsValidTaskStatus(s string) bool {
	for _, valid := range AllTaskStatuses {
		if string(valid) == s {
			return true
		}
	}
	return false
}

// IsTerminalTaskStatus reports whether no further work is expected. Terminal
// tasks are not claimable and do not count towards open work.
func IsTerminalTaskStatus(s string) bool {
	return s == string(StatusTaskDone) || s == string(StatusTaskCancelled)
}

// AgentStatus is the presence state of an agent (the `agents` table).
//
// "blocked" was added because the MCP surface already offered it and the
// generated agent instructions already told agents to report it — but the
// REST validator did not accept it, so an agent following our own
// onboarding text could write a value the API then rejected.
type AgentStatus string

const (
	StatusAgentActive  AgentStatus = "active"
	StatusAgentIdle    AgentStatus = "idle"
	StatusAgentBlocked AgentStatus = "blocked"
	StatusAgentOffline AgentStatus = "offline"
)

// AllAgentStatuses is the authoritative agent vocabulary, in display order.
var AllAgentStatuses = []AgentStatus{
	StatusAgentActive,
	StatusAgentIdle,
	StatusAgentBlocked,
	StatusAgentOffline,
}

// IsValidAgentStatus reports whether s is a member of the agent vocabulary.
func IsValidAgentStatus(s string) bool {
	for _, valid := range AllAgentStatuses {
		if string(valid) == s {
			return true
		}
	}
	return false
}

// StringSlice renders any status slice as []string, for JSON Schema enums.
// MCP tool schemas need []string, and converting from the typed slices here
// means a new status cannot be added to one surface and forgotten on another.
func StringSlice[T ~string](values []T) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	return out
}

// JoinValues renders a string-like slice for use inside a human-readable
// message, e.g. "must be one of: pending, in_progress, blocked". Generic over
// the vocabularies it serves — statuses and scopes alike.
func JoinValues[T ~string](values []T) string {
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = string(v)
	}
	return strings.Join(parts, ", ")
}
