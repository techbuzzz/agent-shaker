package models

import (
	"time"

	"github.com/google/uuid"
)

// ProjectRepo is one git repository participating in a project.
type ProjectRepo struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	ProjectID uuid.UUID  `json:"project_id" db:"project_id"`
	URL       string     `json:"url" db:"url"`
	Branch    string     `json:"branch" db:"branch"`
	Role      string     `json:"role" db:"role"`
	AgentID   *uuid.UUID `json:"agent_id" db:"agent_id"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateProjectRepoRequest is the REST/MCP input for new repo rows.
type CreateProjectRepoRequest struct {
	ProjectID uuid.UUID  `json:"project_id"`
	URL       string     `json:"url"`
	Branch    string     `json:"branch"`
	Role      string     `json:"role"`
	AgentID   *uuid.UUID `json:"agent_id"`
}

// ValidProjectRepoRoles is the allow-list used by the validator layer.
var ValidProjectRepoRoles = map[string]bool{
	"":       true, // default = "code" via DB
	"code":   true,
	"infra":  true,
	"docs":   true,
	"design": true,
}
