package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// fakeTaskStore records what the handler asked it to write, so a test can tell
// "refused" apart from "refused but changed the data anyway".
type fakeTaskStore struct {
	tasks map[uuid.UUID]*models.Task

	reassignCalls   []reassignCall
	reassignErr     error
	getErr          error
	deleteCalls     []uuid.UUID
	updateStatusErr error
}

type reassignCall struct {
	taskID     uuid.UUID
	assignedTo uuid.UUID
}

func (f *fakeTaskStore) Available() bool { return true }

func (f *fakeTaskStore) CreateTask(context.Context, *models.Task) error { return nil }

func (f *fakeTaskStore) ListTasks(context.Context, *uuid.UUID, string) ([]models.Task, error) {
	return nil, nil
}

func (f *fakeTaskStore) GetTask(_ context.Context, id uuid.UUID) (*models.Task, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	t, ok := f.tasks[id]
	if !ok {
		return nil, errors.New("task not found")
	}
	copyOf := *t
	return &copyOf, nil
}

func (f *fakeTaskStore) UpdateTask(context.Context, *models.Task) error { return nil }

func (f *fakeTaskStore) UpdateTaskStatus(context.Context, uuid.UUID, string) error {
	return f.updateStatusErr
}

func (f *fakeTaskStore) ReassignTask(_ context.Context, id, assignedTo uuid.UUID) error {
	if f.reassignErr != nil {
		return f.reassignErr
	}
	f.reassignCalls = append(f.reassignCalls, reassignCall{taskID: id, assignedTo: assignedTo})
	return nil
}

func (f *fakeTaskStore) DeleteTask(_ context.Context, id uuid.UUID) error {
	f.deleteCalls = append(f.deleteCalls, id)
	return nil
}

// fakeAgentLookup answers with the project each agent belongs to. An agent
// missing from the map reads as "no such agent", which is what a missing row
// looks like to the store.
type fakeAgentLookup struct {
	agents map[uuid.UUID]uuid.UUID // agent id -> project id
}

func (f *fakeAgentLookup) GetAgent(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	project, ok := f.agents[id]
	if !ok {
		return nil, errors.New("agent not found")
	}
	return &models.Agent{ID: id, ProjectID: project}, nil
}

func newReassignRequest(taskID, agentID uuid.UUID) *http.Request {
	req := httptest.NewRequest(http.MethodPut, "/api/tasks/"+taskID.String()+"/reassign",
		strings.NewReader(`{"assigned_to":"`+agentID.String()+`"}`))
	req.SetPathValue("id", taskID.String())
	return req
}

// A task may only be assigned to an agent of its own project.
//
// The foreign key on tasks.assigned_to proves the agent exists and nothing
// else. Writing the id straight through produced tasks owned by agents of
// another project, and the damage appeared far from the write: a blocker on
// such a task answers "agent not found", the dashboard's agent filter hides
// it, and the agent's heartbeat publishes into a project it does not belong to.
func TestReassignTask_AssigneeMustShareTheProject(t *testing.T) {
	projectID := uuid.New()
	otherProjectID := uuid.New()
	taskID := uuid.New()
	localAgent := uuid.New()
	foreignAgent := uuid.New()

	newHandler := func() (*TaskHandler, *fakeTaskStore) {
		store := &fakeTaskStore{tasks: map[uuid.UUID]*models.Task{
			taskID: {ID: taskID, ProjectID: projectID, Status: models.StatusTaskPending},
		}}
		agents := &fakeAgentLookup{agents: map[uuid.UUID]uuid.UUID{
			localAgent:   projectID,
			foreignAgent: otherProjectID,
		}}
		return &TaskHandler{store: store, agents: agents}, store
	}

	t.Run("an agent of this project is accepted", func(t *testing.T) {
		h, store := newHandler()
		rec := httptest.NewRecorder()

		h.ReassignTask(rec, newReassignRequest(taskID, localAgent))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %s)", rec.Code, rec.Body.String())
		}
		if len(store.reassignCalls) != 1 || store.reassignCalls[0].assignedTo != localAgent {
			t.Fatalf("store saw %v, want exactly one write of %s", store.reassignCalls, localAgent)
		}
	})

	t.Run("an agent of another project is refused", func(t *testing.T) {
		h, store := newHandler()
		rec := httptest.NewRecorder()

		h.ReassignTask(rec, newReassignRequest(taskID, foreignAgent))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 (body %s)", rec.Code, rec.Body.String())
		}
		// The refusal must happen before the write, not roll it back.
		if len(store.reassignCalls) != 0 {
			t.Fatalf("a refused reassignment still wrote: %v", store.reassignCalls)
		}
	})

	t.Run("an agent that does not exist is refused", func(t *testing.T) {
		h, store := newHandler()
		rec := httptest.NewRecorder()

		h.ReassignTask(rec, newReassignRequest(taskID, uuid.New()))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rec.Code)
		}
		if len(store.reassignCalls) != 0 {
			t.Fatalf("a refused reassignment still wrote: %v", store.reassignCalls)
		}
	})

	t.Run("a missing task is 404 and never reaches the store write", func(t *testing.T) {
		store := &fakeTaskStore{tasks: map[uuid.UUID]*models.Task{}}
		h := &TaskHandler{store: store, agents: &fakeAgentLookup{}}
		rec := httptest.NewRecorder()

		h.ReassignTask(rec, newReassignRequest(uuid.New(), uuid.New()))

		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 (body %s)", rec.Code, rec.Body.String())
		}
		if len(store.reassignCalls) != 0 {
			t.Fatal("reached the write without a task")
		}
	})

	t.Run("a malformed id is 400, not 500", func(t *testing.T) {
		h, _ := newHandler()
		req := httptest.NewRequest(http.MethodPut, "/api/tasks/not-a-uuid",
			strings.NewReader(`{"assigned_to":"`+localAgent.String()+`"}`))
		req.SetPathValue("id", "not-a-uuid")
		rec := httptest.NewRecorder()

		h.ReassignTask(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (body %s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("the refusal names the agent and the project", func(t *testing.T) {
		h, _ := newHandler()
		rec := httptest.NewRecorder()

		h.ReassignTask(rec, newReassignRequest(taskID, foreignAgent))

		var envelope struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("response is not the error envelope: %v (%s)", err, rec.Body.String())
		}
		if envelope.Error.Message == "" {
			t.Fatal("empty message; the UI renders exactly this field")
		}
		if envelope.Error.Code != "not_found" {
			t.Errorf("code = %q, want not_found", envelope.Error.Code)
		}
		for _, want := range []string{foreignAgent.String(), projectID.String()} {
			if !strings.Contains(envelope.Error.Message, want) {
				t.Errorf("message %q does not mention %s", envelope.Error.Message, want)
			}
		}
	})
}
