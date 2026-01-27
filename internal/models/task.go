package models

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
)

type Task struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	ProjectID   uuid.UUID  `json:"project_id" db:"project_id"`
	Title       string     `json:"title" db:"title"`
	Description string     `json:"description" db:"description"`
	Status      TaskStatus `json:"status" db:"status"`
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
	Status TaskStatus `json:"status"`
	Output string     `json:"output"`
}

type ReassignTaskRequest struct {
	AssignedTo uuid.UUID `json:"assigned_to"`
}
