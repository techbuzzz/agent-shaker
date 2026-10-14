package models

import (
	"time"

	"github.com/google/uuid"
)

// TaskDependency is one edge of the task DAG: TaskID cannot start until
// DependsOnTaskID is done.
//
// It is a separate table rather than a TEXT[] column on tasks for two reasons
// that are mechanical, not stylistic: the reverse lookup ("completing this task
// — what does it unblock?") is the query that runs on every single completion
// and needs an index on the dependency side, and a database-level foreign key is
// what makes a deleted task take its edges with it instead of leaving edges
// pointing at nothing.
type TaskDependency struct {
	ID uuid.UUID `json:"id" db:"id"`
	// TaskID is the task that waits.
	TaskID uuid.UUID `json:"task_id" db:"task_id"`
	// DependsOnTaskID is the task it waits for.
	DependsOnTaskID uuid.UUID `json:"depends_on_task_id" db:"depends_on_task_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// AddTaskDependencyRequest declares that task_id waits for depends_on_task_id.
type AddTaskDependencyRequest struct {
	TaskID          uuid.UUID `json:"task_id"`
	DependsOnTaskID uuid.UUID `json:"depends_on_task_id"`
}
