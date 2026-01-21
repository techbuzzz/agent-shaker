package models

import "time"

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
