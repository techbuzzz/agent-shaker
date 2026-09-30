package models

import (
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
