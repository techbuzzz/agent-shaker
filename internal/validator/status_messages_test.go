package validator

import (
	"errors"
	"strings"
	"testing"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// A validation message is rendered verbatim in the UI — the form banner shows
// exactly this string — and by the MCP surface. "invalid status value" leaves
// the caller with nothing to act on; the vocabulary has to be in the failure.
func TestStatusValidationNamesTheValueAndTheAcceptedSet(t *testing.T) {
	agentReq := &models.UpdateAgentStatusRequest{Status: "levitating"}
	err := ValidateUpdateAgentStatusRequest(agentReq)
	if err == nil {
		t.Fatal("an unknown agent status was accepted")
	}
	msg := err.Error()
	for _, want := range []string{"levitating", "active", "idle", "blocked", "offline"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q does not mention %q", msg, want)
		}
	}
	// The sentinel must survive wrapping so the status maps to 400.
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("error %v no longer wraps ErrInvalidStatus", err)
	}

	taskReq := &models.UpdateTaskRequest{Status: models.TaskStatus("flying")}
	taskErr := ValidateUpdateTaskRequest(taskReq)
	if taskErr == nil {
		t.Fatal("an unknown task status was accepted")
	}
	for _, want := range []string{"flying", "pending", "ready", "in_progress", "done", "cancelled"} {
		if !strings.Contains(taskErr.Error(), want) {
			t.Errorf("task message %q does not mention %q", taskErr.Error(), want)
		}
	}
	if !errors.Is(taskErr, ErrInvalidStatus) {
		t.Errorf("task error %v no longer wraps ErrInvalidStatus", taskErr)
	}
}

// A valid value still passes, and the message change must not have made the
// validator stricter.
func TestStatusValidationAcceptsEveryKnownValue(t *testing.T) {
	for _, status := range models.AllAgentStatuses {
		if err := ValidateUpdateAgentStatusRequest(&models.UpdateAgentStatusRequest{Status: string(status)}); err != nil {
			t.Errorf("agent status %q was rejected: %v", status, err)
		}
	}
	for _, status := range models.AllTaskStatuses {
		if err := ValidateUpdateTaskRequest(&models.UpdateTaskRequest{Status: status}); err != nil {
			t.Errorf("task status %q was rejected: %v", status, err)
		}
	}
}
