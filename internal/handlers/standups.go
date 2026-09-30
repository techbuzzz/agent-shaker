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
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

type StandupHandler struct {
	store *queries.StandupsStore
	hub   *websocket.Hub
}

func NewStandupHandler(store *queries.StandupsStore, hub *websocket.Hub) *StandupHandler {
	return &StandupHandler{store: store, hub: hub}
}

func (h *StandupHandler) CreateStandup(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	var req models.CreateStandupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	standupDate, err := time.Parse("2006-01-02", req.StandupDate)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid standup_date; expected YYYY-MM-DD: %w", err))
		return
	}
	st := models.DailyStandup{
		ID:             uuid.New(),
		AgentID:        req.AgentID,
		ProjectID:      req.ProjectID,
		StandupDate:    standupDate,
		Did:            req.Did,
		Doing:          req.Doing,
		Done:           req.Done,
		Blockers:       req.Blockers,
		Challenges:     req.Challenges,
		ReferenceLinks: req.ReferenceLinks,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := h.store.CreateStandup(r.Context(), &st); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, st)
}

func (h *StandupHandler) ListStandups(w http.ResponseWriter, r *http.Request) {
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
	st, err := h.store.ListStandups(r.Context(), projectID)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if st == nil {
		st = []models.DailyStandup{}
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *StandupHandler) GetStandup(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid standup id: %w", err))
		return
	}
	st, err := h.store.GetStandup(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("standup not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *StandupHandler) UpdateStandup(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid standup id: %w", err))
		return
	}
	var req models.UpdateStandupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	st, err := h.store.GetStandup(r.Context(), id)
	if err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("standup not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	st.Did = req.Did
	st.Doing = req.Doing
	st.Done = req.Done
	st.Blockers = req.Blockers
	st.Challenges = req.Challenges
	st.ReferenceLinks = req.ReferenceLinks
	st.UpdatedAt = time.Now()
	if err := h.store.UpdateStandup(r.Context(), st); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, st)
}

func (h *StandupHandler) DeleteStandup(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid standup id: %w", err))
		return
	}
	if err := h.store.DeleteStandup(r.Context(), id); err != nil {
		if isNotFound(err) {
			httpx.WriteError(w, r, fmt.Errorf("standup not found: %w", err))
			return
		}
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *StandupHandler) RecordHeartbeat(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	var req models.CreateHeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	hb := models.AgentHeartbeat{
		ID:            uuid.New(),
		AgentID:       req.AgentID,
		HeartbeatTime: time.Now(),
		Status:        req.Status,
		Metadata:      req.Metadata,
	}
	if err := h.store.RecordHeartbeat(r.Context(), &hb); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, hb)
}

func (h *StandupHandler) GetAgentHeartbeats(w http.ResponseWriter, r *http.Request) {
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
	since := time.Now().Add(-24 * time.Hour)
	hbs, err := h.store.GetAgentHeartbeats(r.Context(), id, since, 100)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if hbs == nil {
		hbs = []models.AgentHeartbeat{}
	}
	httpx.WriteJSON(w, http.StatusOK, hbs)
}
