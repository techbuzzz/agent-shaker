package a2a_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
	a2aserver "github.com/techbuzzz/agent-shaker/internal/a2a/server"
	"github.com/techbuzzz/agent-shaker/internal/task"
)

func TestAgentCardEndpoint(t *testing.T) {
	handler := a2aserver.NewAgentCardHandler("1.0.0", "http://localhost:8080")

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
	r := mux.NewRouter()
	r.HandleFunc("/a2a/v1/tasks/{taskId}", handler.GetTask)

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

	r := mux.NewRouter()
	r.HandleFunc("/a2a/v1/tasks/{taskId}", handler.GetTask)

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

func TestListTasksWithStatusFilter(t *testing.T) {
	store := task.NewMemoryStore("")
	manager := task.NewManager(store, nil, "http://localhost:8080")
	handler := a2aserver.NewA2AHandler(manager)

	// Create a task
	ctx := context.Background()
	sendReq := &models.SendMessageRequest{
		Message: models.Message{
			Content: "Test message",
			Format:  "text",
		},
	}
	_, _ = manager.CreateTask(ctx, sendReq)

	// Wait for task to complete
	time.Sleep(200 * time.Millisecond)

	// List completed tasks
	req := httptest.NewRequest(http.MethodGet, "/a2a/v1/tasks?status=completed", nil)
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

	// All returned tasks should be completed
	for _, task := range tasks {
		taskMap := task.(map[string]any)
		if taskMap["status"] != "completed" {
			t.Errorf("Expected status 'completed', got %v", taskMap["status"])
		}
	}
}
