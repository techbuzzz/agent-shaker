package models

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ProjectID   uuid.UUID  `json:"project_id" db:"project_id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	Status      string     `json:"status" db:"status"`
	Priority    string     `json:"priority" db:"priority"`
	CreatedBy   uuid.UUID  `json:"created_by" db:"created_by"`
	AssignedTo  *uuid.UUID `json:"assigned_to" db:"assigned_to"`
	Output      string     `json:"output" db:"output"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

type CreateTaskRequest struct {
	ProjectID   uuid.UUID  `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	AssignedTo  *uuid.UUID `json:"assigned_to"`
}

type UpdateTaskRequest struct {
	Status string `json:"status"`
	Output string `json:"output"`
}
