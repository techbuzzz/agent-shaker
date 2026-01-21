package models

import "time"

// Project represents a project in the system
type Project struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AgentRole represents the role of an agent
type AgentRole string

const (
	RoleBackend  AgentRole = "backend"
	RoleFrontend AgentRole = "frontend"
)

// Agent represents an AI agent in the system
type Agent struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Name      string    `json:"name"`
	Role      AgentRole `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
)

// Task represents a task assigned to an agent
type Task struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	AgentID     string     `json:"agent_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Priority    int        `json:"priority"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

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
