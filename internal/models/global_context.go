package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// GlobalContextScope marks whether a doc is shared server-wide or only
// inside one project.
type GlobalContextScope string

const (
	ScopeGlobal  GlobalContextScope = "global"
	ScopeProject GlobalContextScope = "project"
)

// ValidGlobalContextScopes is the allow-list used by validators.
var ValidGlobalContextScopes = map[GlobalContextScope]bool{
	ScopeGlobal:  true,
	ScopeProject: true,
}

// GlobalContext is a server-wide or project-wide markdown doc.
type GlobalContext struct {
	ID        uuid.UUID          `json:"id" db:"id"`
	Scope     GlobalContextScope `json:"scope" db:"scope"`
	ProjectID *uuid.UUID         `json:"project_id" db:"project_id"`
	AgentID   uuid.UUID          `json:"agent_id" db:"agent_id"`
	Title     string             `json:"title" db:"title"`
	Content   string             `json:"content" db:"content"`
	Tags      pq.StringArray     `json:"tags" db:"tags"`
	CreatedAt time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" db:"updated_at"`
}

// CreateGlobalContextRequest is the REST/MCP input for new docs.
type CreateGlobalContextRequest struct {
	Scope     GlobalContextScope `json:"scope"`
	ProjectID *uuid.UUID         `json:"project_id"`
	AgentID   uuid.UUID          `json:"agent_id"`
	Title     string             `json:"title"`
	Content   string             `json:"content"`
	Tags      []string           `json:"tags"`
}

// UpdateGlobalContextRequest is the REST/MCP input for content edits.
type UpdateGlobalContextRequest struct {
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}
