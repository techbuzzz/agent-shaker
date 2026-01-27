package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

type StandupHandler struct {
	db  *database.DB
	hub *websocket.Hub
}

func NewStandupHandler(db *database.DB, hub *websocket.Hub) *StandupHandler {
	return &StandupHandler{db: db, hub: hub}
}

// CreateStandup creates or updates a daily standup entry
func (h *StandupHandler) CreateStandup(w http.ResponseWriter, r *http.Request) {
	var req models.CreateStandupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.AgentID == uuid.Nil {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}
	if req.ProjectID == uuid.Nil {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	if req.Did == "" || req.Doing == "" || req.Done == "" {
		http.Error(w, "did, doing, and done fields are required", http.StatusBadRequest)
		return
	}

	// Parse standup date
	var standupDate time.Time
	var err error
	if req.StandupDate == "" {
		standupDate = time.Now().UTC().Truncate(24 * time.Hour)
	} else {
		standupDate, err = time.Parse("2006-01-02", req.StandupDate)
		if err != nil {
			http.Error(w, "Invalid standup_date format, use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
	}

	standup := models.DailyStandup{
		ID:          uuid.New(),
		AgentID:     req.AgentID,
		ProjectID:   req.ProjectID,
		StandupDate: standupDate,
		Did:         req.Did,
		Doing:       req.Doing,
		Done:        req.Done,
		Blockers:    req.Blockers,
		Challenges:  req.Challenges,
		References:  req.References,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Insert or update (upsert) using ON CONFLICT
	row := h.db.QueryRow(`
		INSERT INTO daily_standups (id, agent_id, project_id, standup_date, did, doing, done, blockers, challenges, references, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (agent_id, standup_date)
		DO UPDATE SET
			did = EXCLUDED.did,
			doing = EXCLUDED.doing,
			done = EXCLUDED.done,
			blockers = EXCLUDED.blockers,
			challenges = EXCLUDED.challenges,
			references = EXCLUDED.references,
			updated_at = EXCLUDED.updated_at
		RETURNING id, created_at, updated_at
	`, standup.ID, standup.AgentID, standup.ProjectID, standup.StandupDate,
		standup.Did, standup.Doing, standup.Done, standup.Blockers,
		standup.Challenges, standup.References, standup.CreatedAt, standup.UpdatedAt)

	if err := row.Scan(&standup.ID, &standup.CreatedAt, &standup.UpdatedAt); err != nil {
		http.Error(w, "Failed to create standup: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast standup update via WebSocket
	h.hub.BroadcastToProject(standup.ProjectID, "standup_update", standup)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(standup)
}

// ListStandups retrieves standups with optional filters
func (h *StandupHandler) ListStandups(w http.ResponseWriter, r *http.Request) {
	projectIDStr := r.URL.Query().Get("project_id")
	agentIDStr := r.URL.Query().Get("agent_id")
	dateStr := r.URL.Query().Get("date")

	query := `
		SELECT s.id, s.agent_id, s.project_id, s.standup_date, s.did, s.doing, s.done, 
		       s.blockers, s.challenges, s.references, s.created_at, s.updated_at,
		       a.name as agent_name, a.role as agent_role, a.team as agent_team
		FROM daily_standups s
		INNER JOIN agents a ON s.agent_id = a.id
		WHERE 1=1
	`
	args := []interface{}{}

	if projectIDStr != "" {
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			http.Error(w, "Invalid project_id format", http.StatusBadRequest)
			return
		}
		query += fmt.Sprintf(" AND s.project_id = $%d", len(args)+1)
		args = append(args, projectID)
	}

	if agentIDStr != "" {
		agentID, err := uuid.Parse(agentIDStr)
		if err != nil {
			http.Error(w, "Invalid agent_id format", http.StatusBadRequest)
			return
		}
		query += fmt.Sprintf(" AND s.agent_id = $%d", len(args)+1)
		args = append(args, agentID)
	}

	if dateStr != "" {
		standupDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, "Invalid date format, use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		query += fmt.Sprintf(" AND s.standup_date = $%d", len(args)+1)
		args = append(args, standupDate)
	}

	query += " ORDER BY s.standup_date DESC, s.created_at DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, "Failed to retrieve standups: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	standups := []models.StandupWithAgent{}
	for rows.Next() {
		var s models.StandupWithAgent
		err := rows.Scan(
			&s.ID, &s.AgentID, &s.ProjectID, &s.StandupDate, &s.Did, &s.Doing, &s.Done,
			&s.Blockers, &s.Challenges, &s.References, &s.CreatedAt, &s.UpdatedAt,
			&s.AgentName, &s.AgentRole, &s.AgentTeam,
		)
		if err != nil {
			http.Error(w, "Failed to scan standup: "+err.Error(), http.StatusInternalServerError)
			return
		}
		standups = append(standups, s)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to retrieve standups: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(standups)
}

// GetStandup retrieves a specific standup by ID
func (h *StandupHandler) GetStandup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid standup ID", http.StatusBadRequest)
		return
	}

	var s models.StandupWithAgent
	err = h.db.QueryRow(`
		SELECT s.id, s.agent_id, s.project_id, s.standup_date, s.did, s.doing, s.done,
		       s.blockers, s.challenges, s.references, s.created_at, s.updated_at,
		       a.name as agent_name, a.role as agent_role, a.team as agent_team
		FROM daily_standups s
		INNER JOIN agents a ON s.agent_id = a.id
		WHERE s.id = $1
	`, id).Scan(
		&s.ID, &s.AgentID, &s.ProjectID, &s.StandupDate, &s.Did, &s.Doing, &s.Done,
		&s.Blockers, &s.Challenges, &s.References, &s.CreatedAt, &s.UpdatedAt,
		&s.AgentName, &s.AgentRole, &s.AgentTeam,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "Standup not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve standup", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}

// UpdateStandup updates a standup entry
func (h *StandupHandler) UpdateStandup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid standup ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateStandupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(`
		UPDATE daily_standups
		SET did = $1, doing = $2, done = $3, blockers = $4, challenges = $5, references = $6, updated_at = $7
		WHERE id = $8
	`, req.Did, req.Doing, req.Done, req.Blockers, req.Challenges, req.References, time.Now(), id)

	if err != nil {
		http.Error(w, "Failed to update standup", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Standup updated successfully"})
}

// DeleteStandup deletes a standup entry
func (h *StandupHandler) DeleteStandup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid standup ID", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec("DELETE FROM daily_standups WHERE id = $1", id)
	if err != nil {
		http.Error(w, "Failed to delete standup", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Standup deleted successfully"})
}

// RecordHeartbeat records an agent heartbeat
func (h *StandupHandler) RecordHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req models.CreateHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.AgentID == uuid.Nil {
		http.Error(w, "agent_id is required", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		req.Status = "active"
	}

	heartbeat := models.AgentHeartbeat{
		ID:            uuid.New(),
		AgentID:       req.AgentID,
		HeartbeatTime: time.Now(),
		Status:        req.Status,
		Metadata:      req.Metadata,
	}

	metadataJSON, err := json.Marshal(req.Metadata)
	if err != nil {
		http.Error(w, "Failed to serialize metadata", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(`
		INSERT INTO agent_heartbeats (id, agent_id, heartbeat_time, status, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, heartbeat.ID, heartbeat.AgentID, heartbeat.HeartbeatTime, heartbeat.Status, metadataJSON)

	if err != nil {
		http.Error(w, "Failed to record heartbeat: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update agent's last_seen timestamp
	_, _ = h.db.Exec("UPDATE agents SET last_seen = $1 WHERE id = $2", heartbeat.HeartbeatTime, heartbeat.AgentID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(heartbeat)
}

// GetAgentHeartbeats retrieves heartbeats for a specific agent
func (h *StandupHandler) GetAgentHeartbeats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	agentID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid agent ID", http.StatusBadRequest)
		return
	}

	limitParam := r.URL.Query().Get("limit")
	if limitParam == "" {
		limitParam = "50"
	}

	limit, err := strconv.Atoi(limitParam)
	if err != nil || limit <= 0 {
		http.Error(w, "limit must be a positive integer", http.StatusBadRequest)
		return
	}

	rows, err := h.db.Query(`
		SELECT id, agent_id, heartbeat_time, status, metadata
		FROM agent_heartbeats
		WHERE agent_id = $1
		ORDER BY heartbeat_time DESC
		LIMIT $2
	`, agentID, limit)

	if err != nil {
		http.Error(w, "Failed to retrieve heartbeats", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	heartbeats := []models.AgentHeartbeat{}
	for rows.Next() {
		var h models.AgentHeartbeat
		var metadataJSON []byte
		err := rows.Scan(&h.ID, &h.AgentID, &h.HeartbeatTime, &h.Status, &metadataJSON)
		if err != nil {
			http.Error(w, "Failed to scan heartbeat", http.StatusInternalServerError)
			return
		}
		if len(metadataJSON) > 0 {
			_ = json.Unmarshal(metadataJSON, &h.Metadata)
		}
		heartbeats = append(heartbeats, h)
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Failed to retrieve heartbeats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(heartbeats)
}
