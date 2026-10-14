package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AgentRole represents the role of an agent.
//
// The set is intentionally open (the column type is VARCHAR in the DB),
// but these are the well-known values the mesh app gates on:
//
//   - pm       — Project Manager. Has MCP write-tools for milestones,
//     global context, and cross-agent task assignment.
//   - backend  — typical backend implementation agent.
//   - frontend — typical frontend implementation agent.
//
// New role values are accepted at the REST boundary; MCP and the Nuxt UI
// gate specific tools and UI surfaces on these constants.
type AgentRole string

const (
	RolePM       AgentRole = "pm"
	RoleBackend  AgentRole = "backend"
	RoleFrontend AgentRole = "frontend"
)

// IsPM returns true if the role is the PM role.
func (r AgentRole) IsPM() bool {
	return r == RolePM
}

type Agent struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ProjectID uuid.UUID `json:"project_id" db:"project_id"`
	Name      string    `json:"name" db:"name"`
	Role      AgentRole `json:"role" db:"role"`
	Team      string    `json:"team" db:"team"`
	Status    string    `json:"status" db:"status"`
	LastSeen  time.Time `json:"last_seen" db:"last_seen"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	// --- Coordination Plane (migration 015) ---

	// LeaseExpiresAt is the deadline the server returned with the last
	// heartbeat. It is the only thing the reaper reads, and it is what makes
	// liveness a lease rather than a timeout heuristic: the deadline is set by
	// the server that answered the ping, so clock skew between an agent and the
	// server cannot expire a live lease.
	//
	// NULL means the agent never heartbeated. Agents created through the UI have
	// no runtime of their own to send one, so the reaper must not evict them.
	LeaseExpiresAt *time.Time `json:"lease_expires_at" db:"lease_expires_at"`

	// CurrentTaskID is what the agent last reported working on. Denormalised
	// from the heartbeat so "who is on what" is one probe rather than a scan of
	// the heartbeat history.
	CurrentTaskID *uuid.UUID `json:"current_task_id" db:"current_task_id"`

	// HeartbeatPayload is the agent's own last report — progress, ETA, and
	// whatever else it chose to send. Stored as JSONB because its shape is the
	// agent's business, not ours; the server reads two known keys and ignores
	// the rest.
	HeartbeatPayload []byte `json:"heartbeat_payload" db:"heartbeat_payload"`
}

// SendHeartbeatRequest is what an agent reports about itself.
//
// It is the coordination-plane request. The older standup-feature request
// (CreateHeartbeatRequest in standup.go) recorded that an agent was alive and
// nothing more; this one also says what it is working on, so the two cannot be
// served by the same call without losing the difference between "ping" and
// "report".
type SendHeartbeatRequest struct {
	Status      string          `json:"status"`
	TaskID      *uuid.UUID      `json:"task_id,omitempty"`
	ProgressPct *int            `json:"progress_pct,omitempty"`
	ETA         *time.Time      `json:"eta,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

// HeartbeatResponse is the lease the agent has to renew.
//
// Returning the deadline explicitly — rather than letting the agent compute it
// from its own TTL constant — keeps a client that hardcodes the TTL working
// after the server's configuration changes, and tells a drifting agent exactly
// how far behind it already is.
type HeartbeatResponse struct {
	LeaseTTLSeconds     int       `json:"lease_ttl_seconds"`
	NextHeartbeatBefore time.Time `json:"next_heartbeat_before"`
}

type CreateAgentRequest struct {
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
	Role      AgentRole `json:"role"`
	Team      string    `json:"team"`
}

type UpdateAgentStatusRequest struct {
	Status string `json:"status"`
}
