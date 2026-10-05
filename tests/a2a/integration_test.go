package a2a_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
	a2aserver "github.com/techbuzzz/agent-shaker/internal/a2a/server"
	"github.com/techbuzzz/agent-shaker/internal/task"
)

func TestAgentCardEndpoint(t *testing.T) {
	handler := a2aserver.NewAgentCardHandler("1.0.0", "http://localhost:8080", true)

	req := httptest.NewRequest(http.MethodGet, "/.well-known/agent-card.json", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	var card map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&card); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate required fields
	if card["name"] != "Agent Shaker" {
		t.Errorf("Expected name 'Agent Shaker', got %v", card["name"])
	}

	if card["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", card["version"])
	}

	// New schema v1.0 has capabilities as an object with fields like a2aVersion
	capabilities, ok := card["capabilities"].(map[string]any)
	if !ok {
		t.Error("Expected capabilities to be an object in new schema")
	} else if a2aVersion, hasA2A := capabilities["a2aVersion"]; !hasA2A || a2aVersion != "1.0" {
		t.Error("Expected capabilities to have a2aVersion field with value 1.0")
	}

	endpoints, ok := card["endpoints"].([]any)
	if !ok || len(endpoints) == 0 {
		t.Error("Expected non-empty endpoints array")
	}
}

func TestSendMessageEndpoint(t *testing.T) {
	store := task.NewMemoryStore("")
	manager := task.NewManager(store, nil, "http://localhost:8080")
	handler := a2aserver.NewA2AHandler(manager)

	body := `{"message": {"content": "Test message", "format": "text"}}`
	req := httptest.NewRequest(http.MethodPost, "/a2a/v1/message", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SendMessage(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("Expected status 202, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["task_id"] == nil {
		t.Error("Expected task_id in response")
	}

	if resp["status"] != "pending" {
		t.Errorf("Expected status 'pending', got %v", resp["status"])
	}
}

func TestSendMessageInvalidRequest(t *testing.T) {
	store := task.NewMemoryStore("")
	manager := task.NewManager(store, nil, "http://localhost:8080")
	handler := a2aserver.NewA2AHandler(manager)

	// Test empty content
	body := `{"message": {"content": ""}}`
	req := httptest.NewRequest(http.MethodPost, "/a2a/v1/message", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.SendMessage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rec.Code)
	}
}

func TestGetTaskEndpoint(t *testing.T) {
	store := task.NewMemoryStore("")
	manager := task.NewManager(store, nil, "http://localhost:8080")
	handler := a2aserver.NewA2AHandler(manager)

	// Create a task first
	ctx := context.Background()
	sendReq := &models.SendMessageRequest{
		Message: models.Message{
			Content: "Test message",
			Format:  "text",
		},
	}
	createdTask, err := manager.CreateTask(ctx, sendReq)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Wait for task to start processing
	time.Sleep(100 * time.Millisecond)

	// Get the task
	r := http.NewServeMux()
	r.HandleFunc("GET /a2a/v1/tasks/{taskId}", handler.GetTask)

	req := httptest.NewRequest(http.MethodGet, "/a2a/v1/tasks/"+createdTask.ID, nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["id"] != createdTask.ID {
		t.Errorf("Expected task ID %s, got %v", createdTask.ID, resp["id"])
	}
}

func TestGetTaskNotFound(t *testing.T) {
	store := task.NewMemoryStore("")
	manager := task.NewManager(store, nil, "http://localhost:8080")
	handler := a2aserver.NewA2AHandler(manager)

	r := http.NewServeMux()
	r.HandleFunc("GET /a2a/v1/tasks/{taskId}", handler.GetTask)

	req := httptest.NewRequest(http.MethodGet, "/a2a/v1/tasks/nonexistent-id", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", rec.Code)
	}
}

func TestListTasksEndpoint(t *testing.T) {
	store := task.NewMemoryStore("")
	manager := task.NewManager(store, nil, "http://localhost:8080")
	handler := a2aserver.NewA2AHandler(manager)

	// Create a few tasks
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		sendReq := &models.SendMessageRequest{
			Message: models.Message{
				Content: "Test message",
				Format:  "text",
			},
		}
		_, _ = manager.CreateTask(ctx, sendReq)
	}

	// List tasks
	req := httptest.NewRequest(http.MethodGet, "/a2a/v1/tasks", nil)
	rec := httptest.NewRecorder()

	handler.ListTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	tasks, ok := resp["tasks"].([]any)
	if !ok {
		t.Fatal("Expected tasks array in response")
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}

	totalCount, ok := resp["total_count"].(float64)
	if !ok || int(totalCount) != 3 {
		t.Errorf("Expected total_count 3, got %v", resp["total_count"])
	}
}

// waitForTaskTerminal polls until a task reaches a terminal state and returns
// it. Execution happens in a goroutine, so the state has to be observed rather
// than waited out with a fixed sleep.
func waitForTaskTerminal(t *testing.T, manager *task.Manager, taskID string) *models.Task {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		stored, err := manager.GetTask(context.Background(), taskID)
		if err != nil {
			t.Fatalf("GetTask(%s): %v", taskID, err)
		}
		if stored.Status.IsTerminal() {
			return stored
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %s stayed in %q and never reached a terminal state", taskID, stored.Status)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// listTasksByStatus drives the handler's list endpoint with a status filter and
// returns the decoded task list.
func listTasksByStatus(t *testing.T, handler *a2aserver.A2AHandler, status string) []any {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/a2a/v1/tasks?status="+status, nil)
	rec := httptest.NewRecorder()
	handler.ListTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	tasks, ok := resp["tasks"].([]any)
	if !ok {
		t.Fatalf("Expected tasks array in response, got %#v", resp["tasks"])
	}
	return tasks
}

// TestListTasksWithStatusFilter pins the status filter against a task whose
// terminal state is observed rather than assumed.
//
// The earlier version of this test slept, listed with ?status=completed and
// then asserted that everything returned was completed. The filter runs
// server-side, so that assertion could not fail: an empty list satisfied it,
// and so did a list of completed tasks. It kept passing unchanged even after
// the manager stopped reporting synthetic success for undeliverable work.
//
// This version creates a task with no executor, so it must terminate as failed.
// That gives the assertions something to contradict, and a filter for
// "completed" must come back empty.
func TestListTasksWithStatusFilter(t *testing.T) {
	store := task.NewMemoryStore("")
	manager := task.NewManager(store, nil, "http://localhost:8080")
	handler := a2aserver.NewA2AHandler(manager)

	created, err := manager.CreateTask(context.Background(), &models.SendMessageRequest{
		Message: models.Message{
			Content: "Test message",
			Format:  "text",
		},
	})
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	stored := waitForTaskTerminal(t, manager, created.ID)
	if stored.Status != models.TaskStatusFailed {
		t.Fatalf("task status = %q, want %q: a deployment with no executor must not report success",
			stored.Status, models.TaskStatusFailed)
	}

	completed := listTasksByStatus(t, handler, string(models.TaskStatusCompleted))
	if len(completed) != 0 {
		t.Errorf("status=completed returned %d task(s), want 0: nothing completed", len(completed))
	}

	failed := listTasksByStatus(t, handler, string(models.TaskStatusFailed))
	if len(failed) != 1 {
		t.Fatalf("status=failed returned %d task(s), want 1", len(failed))
	}
	if got := failed[0].(map[string]any)["id"]; got != created.ID {
		t.Errorf("status=failed returned task %v, want %v", got, created.ID)
	}
}
