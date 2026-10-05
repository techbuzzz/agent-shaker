package models

import "time"

// SendMessageRequest represents a request to send a message/task to the agent
type SendMessageRequest struct {
	Message  Message  `json:"message"`
	Metadata Metadata `json:"metadata,omitempty"`
}

// Message contains the content and context of a task request
type Message struct {
	Content string         `json:"content"`
	Context map[string]any `json:"context,omitempty"`
	Format  string         `json:"format,omitempty"` // "text", "markdown"
}

// Metadata contains optional metadata for a task request
type Metadata struct {
	Priority    string            `json:"priority,omitempty"` // "low", "medium", "high"
	Timeout     int               `json:"timeout,omitempty"`  // seconds
	CallbackURL string            `json:"callback_url,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

// SendMessageResponse represents the response after sending a message
type SendMessageResponse struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// TaskStatus represents the status of an A2A task
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCanceled  TaskStatus = "canceled"
)

// IsTerminal reports whether no further transition is possible for a status.
//
// Cancellation is terminal but is not a failure: a caller asked for the work to
// stop, and conflating the two would make "did this succeed?" unanswerable
// from the task record alone. The store persists status as free-form TEXT, so
// the closed set lives here rather than in the schema.
func (s TaskStatus) IsTerminal() bool {
	switch s {
	case TaskStatusCompleted, TaskStatusFailed, TaskStatusCanceled:
		return true
	default:
		return false
	}
}

// Task represents an A2A task
type Task struct {
	ID          string     `json:"id"`
	Status      TaskStatus `json:"status"`
	Message     Message    `json:"message"`
	Result      *Result    `json:"result,omitempty"`
	Artifacts   []Artifact `json:"artifacts,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Result represents the result of a completed task
type Result struct {
	Content string         `json:"content"`
	Format  string         `json:"format"`
	Data    map[string]any `json:"data,omitempty"`
}

// TaskListResponse represents the response for listing tasks
type TaskListResponse struct {
	Tasks      []Task `json:"tasks"`
	TotalCount int    `json:"total_count"`
}
