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
	HeartbeatPayload map[string]interface{} `json:"heartbeat_payload" db:"heartbeat_payload"`

	// LastProgressPct and LastHeartbeatETA come from the newest heartbeat row,
	// joined in the read query. They live on the heartbeat rather than here:
	// denormalising them onto the agents table would mean a column per field the
	// heartbeat ever carries, written on every ping.
	LastProgressPct  *int       `json:"last_progress_pct"`
	LastHeartbeatETA *time.Time `json:"last_heartbeat_eta"`
}

// StaleAgent is one agent whose lease has expired, as the reaper sees it.
//
// It carries only what the reaper has to act on and broadcast. Pulling the full
// agent row would mean an extra round trip per stale agent for fields the
// reaper never reads.
type StaleAgent struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Name      string    `json:"name"`
}

// HeartbeatStatus is what an agent says it is doing right now.
//
// It is deliberately NOT AgentStatus. An agent's stored status answers "is it
// available to the fleet" and is constrained by the database to
// (active, idle, blocked, offline). A heartbeat answers a different question —
// "what is it doing" — and its most useful answer, "working", has no
// counterpart in the stored vocabulary. Migration 014 normalised a hand-written
// 'working' down to 'active' for exactly this reason, and a heartbeat that
// could not say "working" would leave the dashboard unable to distinguish an
// agent deep in a task from one idling with a finished lease.
type HeartbeatStatus string

const (
	HeartbeatWorking HeartbeatStatus = "working"
	HeartbeatIdle    HeartbeatStatus = "idle"
	HeartbeatBlocked HeartbeatStatus = "blocked"
)

// AllHeartbeatStatuses is the authoritative heartbeat vocabulary, used to build
// the MCP tool enum and the validation message so the two cannot drift.
var AllHeartbeatStatuses = []HeartbeatStatus{
	HeartbeatWorking,
	HeartbeatIdle,
	HeartbeatBlocked,
}

// IsValidHeartbeatStatus reports whether s is a member of the heartbeat
// vocabulary.
//
// "offline" is absent on purpose: an agent that is heartbeating is by
// definition not offline. Letting it report that state would be a way to take
// itself out of the fleet's accounting while still holding a live lease.
func IsValidHeartbeatStatus(s string) bool {
	for _, valid := range AllHeartbeatStatuses {
		if string(valid) == s {
			return true
		}
	}
	return false
}

// ToAgentStatus maps a reported activity onto the stored agent status.
//
// working → active, because "active" is what the stored vocabulary calls an
// agent that is present and busy; anything reported as blocked keeps the agent
// blocked so the fleet's view and the agent's own view agree.
func (h HeartbeatStatus) ToAgentStatus() string {
	if h == HeartbeatBlocked {
		return "blocked"
	}
	if h == HeartbeatIdle {
		return "idle"
	}
	return "active"
}

// SendHeartbeatRequest is what an agent reports about itself.
//
// It is the coordination-plane request. The older standup-feature request
// (CreateHeartbeatRequest in standup.go) recorded that an agent was alive and
// nothing more; this one also says what it is working on, so the two cannot be
// served by the same call without losing the difference between "ping" and
// "report".
type SendHeartbeatRequest struct {
	// Status is a HeartbeatStatus ("working", "idle", "blocked"), not an
	// AgentStatus. See HeartbeatStatus for why the two vocabularies differ.
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
