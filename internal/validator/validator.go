package validator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

var (
	ErrEmptyName        = errors.New("name cannot be empty")
	ErrNameTooLong      = errors.New("name cannot exceed 255 characters")
	ErrEmptyTitle       = errors.New("title cannot be empty")
	ErrTitleTooLong     = errors.New("title cannot exceed 255 characters")
	ErrInvalidPriority  = errors.New("priority must be low, medium, or high")
	ErrInvalidStatus    = errors.New("invalid status value")
	ErrInvalidProjectID = errors.New("project_id is required")
	ErrInvalidAgentID   = errors.New("agent_id is required")
	// ErrInvalidToolType wraps models.ErrUnknownToolType, whose message already
	// enumerates the accepted values. Declared here so the HTTP layer can tell
	// this apart from a malformed field without inspecting prose.
	ErrInvalidToolType = errors.New("invalid tool_type")
)

// ValidateCreateProjectRequest validates project creation request
func ValidateCreateProjectRequest(req *models.CreateProjectRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return ErrEmptyName
	}
	if len(req.Name) > 255 {
		return ErrNameTooLong
	}
	return nil
}

// ValidateCreateAgentRequest validates agent creation request
func ValidateCreateAgentRequest(req *models.CreateAgentRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return ErrEmptyName
	}
	if len(req.Name) > 255 {
		return ErrNameTooLong
	}
	if req.ProjectID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidProjectID
	}
	// Parsed rather than merely range-checked, so an unrecognised value is
	// refused at the boundary with a message listing what is accepted, instead
	// of reaching the agents_tool_type_check constraint and surfacing as a 500
	// naming a database constraint. Empty is legal and normalises to 'unknown' —
	// see models.ParseToolType.
	if _, err := models.ParseToolType(string(req.ToolType)); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidToolType, err)
	}
	return nil
}

// ValidateCreateTaskRequest validates task creation request
func ValidateCreateTaskRequest(req *models.CreateTaskRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return ErrEmptyTitle
	}
	if len(req.Title) > 255 {
		return ErrTitleTooLong
	}
	if req.ProjectID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidProjectID
	}
	if req.CreatedBy.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidAgentID
	}
	if req.Priority != "" && req.Priority != "low" && req.Priority != "medium" && req.Priority != "high" {
		return ErrInvalidPriority
	}
	return nil
}

// ValidateUpdateTaskRequest validates task update request
func ValidateUpdateTaskRequest(req *models.UpdateTaskRequest) error {
	if !models.IsValidTaskStatus(string(req.Status)) {
		return fmt.Errorf("%w: invalid status %q; must be one of: %s",
			ErrInvalidStatus, req.Status, models.JoinValues(models.AllTaskStatuses))
	}
	return nil
}

// ValidateUpdateAgentStatusRequest validates agent status update request.
//
// The message names the accepted values and the one it was given. A bare
// "invalid status value" leaves the caller with nothing to act on, and this
// error is rendered verbatim in the UI — the same reasoning as JoinValues on
// the MCP surface, where the vocabulary has to be discoverable from the failure.
func ValidateUpdateAgentStatusRequest(req *models.UpdateAgentStatusRequest) error {
	if !models.IsValidAgentStatus(req.Status) {
		return fmt.Errorf("%w: invalid status %q; must be one of: %s",
			ErrInvalidStatus, req.Status, models.JoinValues(models.AllAgentStatuses))
	}
	return nil
}

// ValidateCreateContextRequest validates context creation request
func ValidateCreateContextRequest(req *models.CreateContextRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return ErrEmptyTitle
	}
	if len(req.Title) > 255 {
		return ErrTitleTooLong
	}
	if req.ProjectID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidProjectID
	}
	if req.AgentID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidAgentID
	}
	return nil
}

// ValidateUpdateContextRequest validates context update request
func ValidateUpdateContextRequest(req *models.UpdateContextRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return ErrEmptyTitle
	}
	if len(req.Title) > 255 {
		return ErrTitleTooLong
	}
	return nil
}

// ValidateCreateMilestoneRequest validates milestone creation.
func ValidateCreateMilestoneRequest(req *models.CreateMilestoneRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return ErrEmptyTitle
	}
	if len(req.Title) > 255 {
		return ErrTitleTooLong
	}
	if req.ProjectID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidProjectID
	}
	if req.CreatedBy.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidAgentID
	}
	return nil
}

// ValidateCreateProjectRepoRequest validates project_repo creation.
func ValidateCreateProjectRepoRequest(req *models.CreateProjectRepoRequest) error {
	if strings.TrimSpace(req.URL) == "" {
		return errors.New("url cannot be empty")
	}
	if req.ProjectID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidProjectID
	}
	if req.Role != "" && !models.ValidProjectRepoRoles[req.Role] {
		return fmt.Errorf("role must be one of: code, infra, docs, design")
	}
	return nil
}

// ValidateCreateGlobalContextRequest validates global_contexts creation.
//
// Cross-field rule: scope='global' ⇒ project_id must be nil; scope='project'
// ⇒ project_id must be non-zero. We surface a 400 instead of letting the
// DB CHECK constraint fail with 500.
func ValidateCreateGlobalContextRequest(req *models.CreateGlobalContextRequest) error {
	if strings.TrimSpace(req.Title) == "" {
		return ErrEmptyTitle
	}
	if len(req.Title) > 255 {
		return ErrTitleTooLong
	}
	if req.AgentID.String() == "00000000-0000-0000-0000-000000000000" {
		return ErrInvalidAgentID
	}
	if !models.ValidGlobalContextScopes[req.Scope] {
		return fmt.Errorf("scope must be 'global' or 'project'")
	}
	if req.Scope == models.ScopeGlobal && req.ProjectID != nil {
		return errors.New("scope=global requires project_id to be omitted")
	}
	if req.Scope == models.ScopeProject && (req.ProjectID == nil || req.ProjectID.String() == "00000000-0000-0000-0000-000000000000") {
		return errors.New("scope=project requires project_id")
	}
	return nil
}
