package models

import (
	"time"

	"github.com/google/uuid"
)

// MilestoneStatus is the lifecycle of a milestone.
type MilestoneStatus string

const (
	MilestonePlanned MilestoneStatus = "planned"
	MilestoneActive  MilestoneStatus = "active"
	MilestoneDone    MilestoneStatus = "done"
	MilestoneDropped MilestoneStatus = "dropped"
)

// ValidMilestoneStatuses is the set used by validators and the MCP layer.
var ValidMilestoneStatuses = map[MilestoneStatus]bool{
	MilestonePlanned: true,
	MilestoneActive:  true,
	MilestoneDone:    true,
	MilestoneDropped: true,
}

// Milestone is a time-bounded group of tasks under a project.
type Milestone struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	ProjectID   uuid.UUID       `json:"project_id" db:"project_id"`
	Title       string          `json:"title" db:"title"`
	Description string          `json:"description" db:"description"`
	Status      MilestoneStatus `json:"status" db:"status"`
	TargetDate  *time.Time      `json:"target_date" db:"target_date"`
	CreatedBy   uuid.UUID       `json:"created_by" db:"created_by"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// CreateMilestoneRequest is the REST/MCP input for new milestones.
type CreateMilestoneRequest struct {
	ProjectID   uuid.UUID       `json:"project_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      MilestoneStatus `json:"status"`
	TargetDate  *time.Time      `json:"target_date"`
	CreatedBy   uuid.UUID       `json:"created_by"`
}

// UpdateMilestoneRequest is the REST/MCP input for status edits.
type UpdateMilestoneRequest struct {
	Status      MilestoneStatus `json:"status"`
	Description string          `json:"description"`
}
