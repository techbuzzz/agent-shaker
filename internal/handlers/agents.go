package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// AgentHandler manages CRUD for the agents resource.
type AgentHandler struct {
	store *queries.AgentsStore
	hub   *websocket.Hub
}

// NewAgentHandler returns a handler backed by the supplied store.
func NewAgentHandler(store *queries.AgentsStore, hub *websocket.Hub) *AgentHandler {
	return &AgentHandler{store: store, hub: hub}
}

func (h *AgentHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	var req models.CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateCreateAgentRequest(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
		return
	}

	agent := models.Agent{
		ID:        uuid.New(),
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Role:      req.Role,
		Team:      req.Team,
		Status:    "active",
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
	}
	if err := h.store.CreateAgent(r.Context(), &agent); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(agent.ProjectID, "agent_update", agent)
	httpx.WriteJSON(w, http.StatusCreated, agent)
}

func (h *AgentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	var projectID *uuid.UUID
	if pidStr := r.URL.Query().Get("project_id"); pidStr != "" {
		pid, err := uuid.Parse(pidStr)
		if err != nil {
			httpx.WriteError(w, r, fmt.Errorf("invalid project_id: %w", err))
			return
		}
		projectID = &pid
	}
	agents, err := h.store.ListAgents(r.Context(), projectID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if agents == nil {
		agents = []models.Agent{}
	}
	httpx.WriteJSON(w, http.StatusOK, agents)
}

func (h *AgentHandler) GetAgent(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid agent id: %w", err))
		return
	}
	agent, err := h.store.GetAgent(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("agent not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, agent)
}

func (h *AgentHandler) UpdateAgentStatus(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid agent id: %w", err))
		return
	}
	var req models.UpdateAgentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateUpdateAgentStatusRequest(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
		return
	}
	if err := h.store.UpdateAgentStatus(r.Context(), id, req.Status, time.Now()); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	agent, err := h.store.GetAgent(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("agent not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(agent.ProjectID, "agent_update", agent)
	httpx.WriteJSON(w, http.StatusOK, agent)
}

func (h *AgentHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid agent id: %w", err))
		return
	}

	tx, err := h.store.BeginTx(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	projectID, err := h.store.GetAgentProjectID(r.Context(), tx, id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("agent not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	if _, err := h.store.DeleteAgentCascade(r.Context(), tx, id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(projectID, "agent_deleted", map[string]any{
		"agent_id":   id,
		"project_id": projectID,
		"deleted_at": time.Now(),
	})
	w.WriteHeader(http.StatusNoContent)
}
