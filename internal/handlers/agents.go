package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/coordination"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// AgentHandler manages CRUD for the agents resource.
type AgentHandler struct {
	store      *queries.AgentsStore
	tasks      *queries.TasksStore
	heartbeats *queries.HeartbeatsStore
	hub        *websocket.Hub
}

// NewAgentHandler returns a handler backed by the supplied stores.
//
// The tasks and heartbeats stores are injected rather than looked up because the
// coordination surface writes two tables per ping — the agent's live lease and
// the heartbeat history — and it validates the reported task before doing so.
func NewAgentHandler(store *queries.AgentsStore, tasks *queries.TasksStore, heartbeats *queries.HeartbeatsStore, hub *websocket.Hub) *AgentHandler {
	return &AgentHandler{store: store, tasks: tasks, heartbeats: heartbeats, hub: hub}
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
		httpx.WriteValidationError(w, r, err)
		return
	}

	// Normalised rather than taken verbatim: an empty or differently-cased value
	// becomes the canonical 'unknown', so the column holds one spelling of each
	// tool type and the config endpoint's switch cannot miss on 'Claude-Code'.
	toolType, err := models.ParseToolType(string(req.ToolType))
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("tool_type: %w: %w", httpx.ErrBadRequest, err))
		return
	}

	agent := models.Agent{
		ID:        uuid.New(),
		ProjectID: req.ProjectID,
		Name:      req.Name,
		Role:      req.Role,
		Team:      req.Team,
		Status:    "active",
		ToolType:  toolType,
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
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid agent id: %v", httpx.ErrBadRequest, err))
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
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid agent id: %v", httpx.ErrBadRequest, err))
		return
	}
	var req models.UpdateAgentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateUpdateAgentStatusRequest(&req); err != nil {
		httpx.WriteValidationError(w, r, err)
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
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid agent id: %v", httpx.ErrBadRequest, err))
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

// Heartbeat renews an agent's lease and records what it reported.
//
// The two writes cannot share a transaction: the lease renewal is a single
// statement inside AgentsStore.RecordHeartbeat that binds to the pool, and the
// history append is the one that can be enlisted. The history row commits first
// — a recorded heartbeat whose lease write failed is a gap the reaper resolves
// on its own, whereas a renewed lease with no record is a live agent whose
// status the dashboard cannot explain.
func (h *AgentHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() || !h.heartbeats.Available() || !h.tasks.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	agentIDRaw := r.PathValue("agent_id")
	if agentIDRaw == "" {
		agentIDRaw = vars["id"]
	}
	agentID, err := uuid.Parse(agentIDRaw)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid agent id: %v", httpx.ErrBadRequest, err))
		return
	}
	projectIDRaw := projectPathID(r)
	projectID, err := uuid.Parse(projectIDRaw)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid project_id: %v", httpx.ErrBadRequest, err))
		return
	}

	var req models.SendHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w: %w", httpx.ErrBadRequest, err))
		return
	}
	if !models.IsValidHeartbeatStatus(req.Status) {
		httpx.WriteError(w, r, fmt.Errorf("%w: invalid status %q; must be one of: %s",
			httpx.ErrBadRequest, req.Status, models.JoinValues(models.AllHeartbeatStatuses)))
		return
	}
	if req.ProgressPct != nil && (*req.ProgressPct < 0 || *req.ProgressPct > 100) {
		httpx.WriteError(w, r, fmt.Errorf("%w: progress_pct must be between 0 and 100", httpx.ErrBadRequest))
		return
	}

	agent, err := h.store.GetAgent(r.Context(), agentID)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("agent not found: %w", httpx.ErrNotFound))
			return
		}
		httpx.WriteError(w, r, fmt.Errorf("get agent: %w", err))
		return
	}
	if agent.ProjectID != projectID {
		// Same reason as GetBlocker: a project-scoped URL must not confirm or
		// renew an agent that belongs to another project.
		httpx.WriteError(w, r, fmt.Errorf("agent not found: %w", httpx.ErrNotFound))
		return
	}

	if req.TaskID != nil {
		task, err := h.tasks.GetTask(r.Context(), *req.TaskID)
		if err != nil {
			if isNotFound(err) {
				httpx.WriteError(w, r, fmt.Errorf("task not found: %w", httpx.ErrNotFound))
				return
			}
			httpx.WriteError(w, r, fmt.Errorf("get task: %w", err))
			return
		}
		if task.ProjectID != projectID {
			httpx.WriteError(w, r, fmt.Errorf("task not found: %w", httpx.ErrNotFound))
			return
		}
	}

	ttl := coordination.DefaultConfig().LeaseTTL

	// Metadata is stored as a JSONB object on a column the standup reader
	// decodes as a map, so a heartbeat carrying a non-object would write a row
	// nothing can read back. Reject it here instead.
	var metadata map[string]interface{}
	if len(req.Metadata) > 0 {
		if err := json.Unmarshal(req.Metadata, &metadata); err != nil {
			httpx.WriteError(w, r, fmt.Errorf("%w: metadata must be a JSON object: %v", httpx.ErrBadRequest, err))
			return
		}
	}

	tx, err := h.store.BeginTx(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	defer tx.Rollback()

	if err := h.heartbeats.RecordHeartbeat(r.Context(), tx, &models.AgentHeartbeat{
		AgentID:     agentID,
		ProjectID:   &projectID,
		TaskID:      req.TaskID,
		ProgressPct: req.ProgressPct,
		ETA:         req.ETA,
		Status:      req.Status,
		Metadata:    metadata,
	}); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := tx.Commit(); err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	// The agent row stores the fleet's view (active/idle/blocked/offline); the
	// reported activity is what the heartbeat history and the event carry. See
	// models.HeartbeatStatus for why the two vocabularies are separate.
	agentStatus := models.HeartbeatStatus(req.Status).ToAgentStatus()
	renewed, err := h.store.RecordHeartbeat(r.Context(), agentID, agentStatus, req.TaskID, req.ProgressPct, req.Metadata, ttl)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("agent not found: %w", httpx.ErrNotFound))
			return
		}
		httpx.WriteError(w, r, fmt.Errorf("record heartbeat: %w", err))
		return
	}

	nextHeartbeatBefore := time.Now().Add(ttl)
	h.hub.BroadcastToProject(projectID, "agent_heartbeat", map[string]any{
		"agent_id":              agentID,
		"project_id":            projectID,
		"agent_name":            renewed.Name,
		"status":                req.Status,
		"task_id":               req.TaskID,
		"progress_pct":          req.ProgressPct,
		"eta":                   req.ETA,
		"next_heartbeat_before": nextHeartbeatBefore,
	})
	httpx.WriteJSON(w, http.StatusOK, models.HeartbeatResponse{
		LeaseTTLSeconds:     int(ttl.Seconds()),
		NextHeartbeatBefore: nextHeartbeatBefore,
	})
}
