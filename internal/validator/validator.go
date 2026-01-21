package validator

import (
	"errors"
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
	validStatuses := map[string]bool{
		"pending":     true,
		"in_progress": true,
		"blocked":     true,
		"done":        true,
		"cancelled":   true,
	}
	if !validStatuses[req.Status] {
		return ErrInvalidStatus
	}
	return nil
}

// ValidateUpdateAgentStatusRequest validates agent status update request
func ValidateUpdateAgentStatusRequest(req *models.UpdateAgentStatusRequest) error {
	validStatuses := map[string]bool{
		"active":  true,
		"idle":    true,
		"offline": true,
	}
	if !validStatuses[req.Status] {
		return ErrInvalidStatus
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
