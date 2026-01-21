package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Context struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	ProjectID uuid.UUID      `json:"project_id" db:"project_id"`
	AgentID   uuid.UUID      `json:"agent_id" db:"agent_id"`
	TaskID    *uuid.UUID     `json:"task_id" db:"task_id"`
	Title     string         `json:"title" db:"title"`
	Content   string         `json:"content" db:"content"`
	Tags      pq.StringArray `json:"tags" db:"tags"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

type CreateContextRequest struct {
	ProjectID uuid.UUID  `json:"project_id"`
	AgentID   uuid.UUID  `json:"agent_id"`
	TaskID    *uuid.UUID `json:"task_id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Tags      []string   `json:"tags"`
}
