package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TaskStatus is defined in status.go, which is the single source of truth
// for status vocabularies. It used to be defined here alongside constants
// (pending/in_progress/completed/failed) that nothing referenced and that
// contradicted the vocabulary the REST API actually accepts. Those constants
// are gone; use the StatusTask* constants in status.go.

type Task struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	ProjectID   uuid.UUID      `json:"project_id" db:"project_id"`
	Title       string         `json:"title" db:"title"`
	Description string         `json:"description" db:"description"`
	Status      TaskStatus     `json:"status" db:"status"`
	Priority    string         `json:"priority" db:"priority"`
	CreatedBy   uuid.UUID      `json:"created_by" db:"created_by"`
	AssignedTo  *uuid.UUID     `json:"assigned_to" db:"assigned_to"`
	Output      string         `json:"output" db:"output"`
	MilestoneID *uuid.UUID     `json:"milestone_id" db:"milestone_id"`
	Tags        pq.StringArray `json:"tags" db:"tags"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`

	// --- Coordination Plane (migration 015) ---

	// PredecessorTaskID is the task this one was handed off from, if any. It is
	// recorded even when there is no dependency edge, because "this task exists
	// because that one finished" is a fact about the history of the work that
	// survives a dependency being added after the fact.
	PredecessorTaskID *uuid.UUID `json:"predecessor_task_id" db:"predecessor_task_id"`

	// HandoffContext is what the predecessor left behind for whoever picks this
	// up: the API shape, the branch, the gotcha. A successor task without it
	// forces the next agent to rediscover what the previous one already learned,
	// which is the whole cost the handoff exists to avoid.
	HandoffContext string `json:"handoff_context" db:"handoff_context"`
}

type CreateTaskRequest struct {
	ProjectID   uuid.UUID  `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	AssignedTo  *uuid.UUID `json:"assigned_to"`
	MilestoneID *uuid.UUID `json:"milestone_id"`
	Tags        []string   `json:"tags"`
}

type UpdateTaskRequest struct {
	Status TaskStatus `json:"status"`
	Output string     `json:"output"`
}

type ReassignTaskRequest struct {
	AssignedTo uuid.UUID `json:"assigned_to"`
}

// CompleteTaskWithHandoffRequest closes a task and opens its successor in one
// call.
//
// The two halves belong together on purpose. Creating the successor as a
// separate call would leave a window where the work is finished but nothing
// knows about the next step, and an agent that crashes in that window leaves a
// completed task and no follow-up.
type CompleteTaskWithHandoffRequest struct {
	TaskID uuid.UUID `json:"task_id"`
	Output string    `json:"output"`

	SuccessorTitle       string     `json:"successor_title"`
	SuccessorDescription string     `json:"successor_description"`
	SuccessorAssignedTo  *uuid.UUID `json:"successor_assigned_to,omitempty"`
	SuccessorPriority    string     `json:"successor_priority,omitempty"`
	SuccessorMilestoneID *uuid.UUID `json:"successor_milestone_id,omitempty"`

	// HandoffContext is the part that is not re-derivable from the work itself.
	HandoffContext string `json:"handoff_context"`
}
