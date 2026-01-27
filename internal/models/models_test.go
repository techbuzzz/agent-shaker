package models

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestProjectModel(t *testing.T) {
	project := Project{
		ID:          uuid.New(),
		Name:        "Test Project",
		Description: "Test Description",
		Status:      "active",
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
	}

	if project.Name != "Test Project" {
		t.Errorf("Expected project name to be 'Test Project', got '%s'", project.Name)
	}

	if project.Description != "Test Description" {
		t.Errorf("Expected description to be 'Test Description', got '%s'", project.Description)
	}

	if project.Status != "active" {
		t.Errorf("Expected status to be 'active', got '%s'", project.Status)
	}

	if project.ID == uuid.Nil {
		t.Error("Expected non-nil project ID")
	}
}

func TestAgentModel(t *testing.T) {
	agent := Agent{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		Name:      "Test Agent",
		Role:      "backend",
		Team:      "Backend Team",
		Status:    "active",
		LastSeen:  time.Time{},
		CreatedAt: time.Time{},
	}

	if agent.Name != "Test Agent" {
		t.Errorf("Expected agent name to be 'Test Agent', got '%s'", agent.Name)
	}

	if agent.Role != "backend" {
		t.Errorf("Expected role to be 'backend', got '%s'", agent.Role)
	}

	if agent.Team != "Backend Team" {
		t.Errorf("Expected team to be 'Backend Team', got '%s'", agent.Team)
	}

	if agent.Status != "active" {
		t.Errorf("Expected status to be 'active', got '%s'", agent.Status)
	}

	if agent.ID == uuid.Nil {
		t.Error("Expected non-nil agent ID")
	}

	if agent.ProjectID == uuid.Nil {
		t.Error("Expected non-nil project ID")
	}
}

func TestTaskModel(t *testing.T) {
	assignedTo := uuid.New()
	task := Task{
		ID:          uuid.New(),
		ProjectID:   uuid.New(),
		Title:       "Test Task",
		Description: "Test Description",
		Status:      "pending",
		Priority:    "high",
		CreatedBy:   uuid.New(),
		AssignedTo:  &assignedTo,
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
	}

	if task.Title != "Test Task" {
		t.Errorf("Expected task title to be 'Test Task', got '%s'", task.Title)
	}

	if task.Status != "pending" {
		t.Errorf("Expected status to be 'pending', got '%s'", task.Status)
	}

	if task.Priority != "high" {
		t.Errorf("Expected priority to be 'high', got '%s'", task.Priority)
	}

	if task.Description != "Test Description" {
		t.Errorf("Expected description to be 'Test Description', got '%s'", task.Description)
	}

	if task.ID == uuid.Nil {
		t.Error("Expected non-nil task ID")
	}

	if task.ProjectID == uuid.Nil {
		t.Error("Expected non-nil project ID")
	}

	if task.CreatedBy == uuid.Nil {
		t.Error("Expected non-nil CreatedBy")
	}

	if task.AssignedTo == nil {
		t.Error("Expected AssignedTo to be set")
	} else if *task.AssignedTo != assignedTo {
		t.Errorf("Expected AssignedTo to match assigned UUID, got '%s'", task.AssignedTo.String())
	}
}

func TestContextModel(t *testing.T) {
	taskID := uuid.New()
	ctx := Context{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		AgentID:   uuid.New(),
		TaskID:    &taskID,
		Title:     "Test Context",
		Content:   "Test Content",
		Tags:      []string{"api", "documentation"},
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
	}

	if ctx.Title != "Test Context" {
		t.Errorf("Expected context title to be 'Test Context', got '%s'", ctx.Title)
	}

	if len(ctx.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(ctx.Tags))
	}

	if ctx.Tags[0] != "api" {
		t.Errorf("Expected first tag to be 'api', got '%s'", ctx.Tags[0])
	}

	if ctx.Content != "Test Content" {
		t.Errorf("Expected content to be 'Test Content', got '%s'", ctx.Content)
	}

	if ctx.ID == uuid.Nil {
		t.Error("Expected non-nil context ID")
	}

	if ctx.ProjectID == uuid.Nil {
		t.Error("Expected non-nil project ID")
	}

	if ctx.AgentID == uuid.Nil {
		t.Error("Expected non-nil agent ID")
	}

	if ctx.TaskID == nil {
		t.Error("Expected TaskID to be set")
	} else if *ctx.TaskID != taskID {
		t.Errorf("Expected TaskID to match task UUID, got '%s'", ctx.TaskID.String())
	}
}
