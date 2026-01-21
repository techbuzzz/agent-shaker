package models

import "time"

// AgentRole represents the role of an agent
type AgentRole string

const (
	RoleBackend  AgentRole = "backend"
	RoleFrontend AgentRole = "frontend"
)

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
)

// Documentation represents markdown documentation for a task
type Documentation struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	Content   string    `json:"content"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskUpdate represents a task status update event
type TaskUpdate struct {
	TaskID    string     `json:"task_id"`
	AgentID   string     `json:"agent_id"`
	Status    TaskStatus `json:"status"`
	Message   string     `json:"message"`
	Timestamp time.Time  `json:"timestamp"`
}
