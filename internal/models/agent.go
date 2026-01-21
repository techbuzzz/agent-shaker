package models

import (
	"time"

	"github.com/google/uuid"
)

// AgentRole represents the role of an agent
type AgentRole string

const (
	RoleBackend  AgentRole = "backend"
	RoleFrontend AgentRole = "frontend"
)

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
