package models

import (
	"time"

	"github.com/google/uuid"
)

// BlockerType is what kind of help an agent needs.
//
// The vocabulary is closed, in Go and in the database (migration 015), because
// each type routes to a different human: an "approval" is a deploy gate and
// blocks on a specific person, a "question" is a design decision that blocks
// the whole task, a "resource" is a provisioning request. An unknown type would
// be a blocker nobody knows how to action, which is worse than no blocker.
type BlockerType string

const (
	BlockerQuestion   BlockerType = "question"
	BlockerDependency BlockerType = "dependency"
	BlockerResource   BlockerType = "resource"
	BlockerDecision   BlockerType = "decision"
	BlockerApproval   BlockerType = "approval"
)

// AllBlockerTypes is the authoritative vocabulary, used to build the MCP tool
// enum and the validation message so the two cannot drift.
var AllBlockerTypes = []BlockerType{
	BlockerQuestion,
	BlockerDependency,
	BlockerResource,
	BlockerDecision,
	BlockerApproval,
}

// IsValidBlockerType reports whether s is a member of the blocker vocabulary.
func IsValidBlockerType(s string) bool {
	for _, valid := range AllBlockerTypes {
		if string(valid) == s {
			return true
		}
	}
	return false
}

// BlockerStatus is where a blocker is in its lifecycle.
//
//	open ──┬──► in_review ──► resolved
//	       └──► dismissed
//
// "dismissed" is separate from "resolved" on purpose: a resolved blocker
// carries an answer the agent can act on, a dismissed one was closed because it
// was not worth answering. Collapsing them would leave agents unable to tell
// "here is what you do" from "this is no longer your problem".
type BlockerStatus string

const (
	BlockerOpen      BlockerStatus = "open"
	BlockerInReview  BlockerStatus = "in_review"
	BlockerResolved  BlockerStatus = "resolved"
	BlockerDismissed BlockerStatus = "dismissed"
)

// AllBlockerStatuses is the authoritative status vocabulary.
var AllBlockerStatuses = []BlockerStatus{
	BlockerOpen,
	BlockerInReview,
	BlockerResolved,
	BlockerDismissed,
}

// IsValidBlockerStatus reports whether s is a member of the blocker vocabulary.
func IsValidBlockerStatus(s string) bool {
	for _, valid := range AllBlockerStatuses {
		if string(valid) == s {
			return true
		}
	}
	return false
}

// IsOpenBlockerStatus reports whether a blocker still needs attention. Only an
// open or in-review blocker keeps its task in the blocked state and its agent
// reporting "blocked"; resolving one releases both.
func IsOpenBlockerStatus(s string) bool {
	return s == string(BlockerOpen) || s == string(BlockerInReview)
}

// Blocker is a structured request from an agent for something only a human can
// decide or provide.
//
// It is not the same thing as an agent status: "blocked" on an agent says that
// work has stopped, a blocker says why and what would unstop it. That is the
// distinction the human actually acts on.
type Blocker struct {
	ID          uuid.UUID     `json:"id" db:"id"`
	TaskID      uuid.UUID     `json:"task_id" db:"task_id"`
	AgentID     uuid.UUID     `json:"agent_id" db:"agent_id"`
	ProjectID   uuid.UUID     `json:"project_id" db:"project_id"`
	Type        BlockerType   `json:"blocker_type" db:"blocker_type"`
	Title       string        `json:"title" db:"title"`
	Description string        `json:"description" db:"description"`
	Status      BlockerStatus `json:"status" db:"status"`
	Resolution  *string       `json:"resolution" db:"resolution"`
	ResolvedBy  *uuid.UUID    `json:"resolved_by" db:"resolved_by"`
	ResolvedAt  *time.Time    `json:"resolved_at" db:"resolved_at"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" db:"updated_at"`
}

// CreateBlockerRequest is the agent-facing payload for raising a blocker.
type CreateBlockerRequest struct {
	TaskID      uuid.UUID   `json:"task_id"`
	BlockerType BlockerType `json:"blocker_type"`
	Title       string      `json:"title"`
	Description string      `json:"description,omitempty"`
}

// ResolveBlockerRequest closes a blocker with an answer the agent can act on.
type ResolveBlockerRequest struct {
	Resolution string    `json:"resolution"`
	ResolvedBy uuid.UUID `json:"resolved_by,omitempty"`
}

// DismissBlockerRequest closes a blocker without an answer. A reason is still
// required: "why was this dropped" is the only thing that lets an agent stop
// re-raising the same blocker tomorrow.
type DismissBlockerRequest struct {
	Reason     string    `json:"reason"`
	ResolvedBy uuid.UUID `json:"resolved_by,omitempty"`
}
