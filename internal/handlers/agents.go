package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

type AgentHandler struct {
	db  *database.DB
	hub *websocket.Hub
}

func NewAgentHandler(db *database.DB, hub *websocket.Hub) *AgentHandler {
	return &AgentHandler{db: db, hub: hub}
}

func (h *AgentHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validator.ValidateCreateAgentRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	_, err := h.db.Exec(`
		INSERT INTO agents (id, project_id, name, role, team, status, last_seen, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, agent.ID, agent.ProjectID, agent.Name, agent.Role, agent.Team, agent.Status, agent.LastSeen, agent.CreatedAt)
	if err != nil {
		http.Error(w, "Failed to create agent", http.StatusInternalServerError)
		return
	}

	// Broadcast agent creation
	h.hub.BroadcastToProject(agent.ProjectID, "agent_update", agent)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(agent)
}

func (h *AgentHandler) ListAgents(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		http.Error(w, "project_id query parameter is required", http.StatusBadRequest)
		return
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		http.Error(w, "Invalid project_id format", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(`
		SELECT id, project_id, name, role, team, status, last_seen, created_at
		FROM agents
		WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		http.Error(w, "Failed to retrieve agents", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Name, &a.Role, &a.Team, &a.Status, &a.LastSeen, &a.CreatedAt); err != nil {
			http.Error(w, "Failed to scan agent", http.StatusInternalServerError)
			return
		}
		agents = append(agents, a)
	}

	// Return empty array instead of null
	if agents == nil {
		agents = []models.Agent{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agents)
}

func (h *AgentHandler) UpdateAgentStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID format", http.StatusBadRequest)
		return
	}

	var req models.UpdateAgentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validator.ValidateUpdateAgentStatusRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(`
		UPDATE agents
		SET status = $1, last_seen = $2
		WHERE id = $3
	`, req.Status, time.Now(), id)
	if err != nil {
		http.Error(w, "Failed to update agent status", http.StatusInternalServerError)
		return
	}

	// Get updated agent
	var agent models.Agent
	err = h.db.QueryRow(`
		SELECT id, project_id, name, role, team, status, last_seen, created_at
		FROM agents
		WHERE id = $1
	`, id).Scan(&agent.ID, &agent.ProjectID, &agent.Name, &agent.Role, &agent.Team, &agent.Status, &agent.LastSeen, &agent.CreatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve agent", http.StatusInternalServerError)
		return
	}

	// Broadcast agent update
	h.hub.BroadcastToProject(agent.ProjectID, "agent_update", agent)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(agent)
}
