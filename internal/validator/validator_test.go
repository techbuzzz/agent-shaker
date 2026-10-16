package validator

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/techbuzzz/agent-shaker/internal/models"
)

func TestValidateCreateProjectRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     models.CreateProjectRequest
		wantErr bool
	}{
		{
			name:    "valid project",
			req:     models.CreateProjectRequest{Name: "Test Project", Description: "Test"},
			wantErr: false,
		},
		{
			name:    "empty name",
			req:     models.CreateProjectRequest{Name: "", Description: "Test"},
			wantErr: true,
		},
		{
			name:    "whitespace name",
			req:     models.CreateProjectRequest{Name: "   ", Description: "Test"},
			wantErr: true,
		},
		{
			name:    "name too long",
			req:     models.CreateProjectRequest{Name: string(make([]byte, 256)), Description: "Test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateProjectRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateProjectRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCreateAgentRequest(t *testing.T) {
	validProjectID := uuid.New()
	zeroUUID := uuid.UUID{}

	tests := []struct {
		name    string
		req     models.CreateAgentRequest
		wantErr bool
	}{
		{
			name:    "valid agent",
			req:     models.CreateAgentRequest{Name: "Test Agent", ProjectID: validProjectID},
			wantErr: false,
		},
		{
			name:    "empty name",
			req:     models.CreateAgentRequest{Name: "", ProjectID: validProjectID},
			wantErr: true,
		},
		{
			name:    "zero project ID",
			req:     models.CreateAgentRequest{Name: "Test Agent", ProjectID: zeroUUID},
			wantErr: true,
		},
		{
			// tool_type arrived with migration 017. A client that predates it
			// sends nothing, and that must keep working — otherwise adding a
			// field would have been a breaking change.
			name:    "no tool type",
			req:     models.CreateAgentRequest{Name: "Test Agent", ProjectID: validProjectID},
			wantErr: false,
		},
		{
			name:    "known tool type",
			req:     models.CreateAgentRequest{Name: "Test Agent", ProjectID: validProjectID, ToolType: models.ToolClaudeCode},
			wantErr: false,
		},
		{
			// Refused here with a message listing the vocabulary, rather than at
			// the CHECK constraint — which would surface as a 500 naming a
			// database object the caller has no way to interpret.
			name:    "unknown tool type",
			req:     models.CreateAgentRequest{Name: "Test Agent", ProjectID: validProjectID, ToolType: "emacs"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateAgentRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateAgentRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			// The tool-type rejection has to be traceable to the vocabulary
			// error rather than being a generic validation failure: that is what
			// lets the HTTP layer answer 400 with the accepted list instead of
			// an opaque message.
			if tt.wantErr && tt.req.ToolType != "" {
				if !errors.Is(err, models.ErrUnknownToolType) {
					t.Errorf("error %v does not wrap models.ErrUnknownToolType", err)
				}
				if !strings.Contains(err.Error(), "claude-code") {
					t.Errorf("error %q does not list the accepted values", err)
				}
			}
		})
	}
}

func TestValidateCreateTaskRequest(t *testing.T) {
	validProjectID := uuid.New()
	validAgentID := uuid.New()
	zeroUUID := uuid.UUID{}

	tests := []struct {
		name    string
		req     models.CreateTaskRequest
		wantErr bool
	}{
		{
			name: "valid task",
			req: models.CreateTaskRequest{
				Title:     "Test Task",
				ProjectID: validProjectID,
				CreatedBy: validAgentID,
				Priority:  "medium",
			},
			wantErr: false,
		},
		{
			name: "empty title",
			req: models.CreateTaskRequest{
				Title:     "",
				ProjectID: validProjectID,
				CreatedBy: validAgentID,
			},
			wantErr: true,
		},
		{
			name: "invalid priority",
			req: models.CreateTaskRequest{
				Title:     "Test",
				ProjectID: validProjectID,
				CreatedBy: validAgentID,
				Priority:  "urgent",
			},
			wantErr: true,
		},
		{
			name: "zero project ID",
			req: models.CreateTaskRequest{
				Title:     "Test",
				ProjectID: zeroUUID,
				CreatedBy: validAgentID,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateTaskRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreateTaskRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateTaskRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     models.UpdateTaskRequest
		wantErr bool
	}{
		{
			name:    "valid status - pending",
			req:     models.UpdateTaskRequest{Status: "pending"},
			wantErr: false,
		},
		{
			name:    "valid status - in_progress",
			req:     models.UpdateTaskRequest{Status: "in_progress"},
			wantErr: false,
		},
		{
			name:    "valid status - done",
			req:     models.UpdateTaskRequest{Status: "done"},
			wantErr: false,
		},
		{
			name:    "invalid status",
			req:     models.UpdateTaskRequest{Status: "completed"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateTaskRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateTaskRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdateAgentStatusRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     models.UpdateAgentStatusRequest
		wantErr bool
	}{
		{
			name:    "valid status - active",
			req:     models.UpdateAgentStatusRequest{Status: "active"},
			wantErr: false,
		},
		{
			name:    "valid status - idle",
			req:     models.UpdateAgentStatusRequest{Status: "idle"},
			wantErr: false,
		},
		{
			name:    "valid status - offline",
			req:     models.UpdateAgentStatusRequest{Status: "offline"},
			wantErr: false,
		},
		{
			name:    "invalid status",
			req:     models.UpdateAgentStatusRequest{Status: "busy"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateAgentStatusRequest(&tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateAgentStatusRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
