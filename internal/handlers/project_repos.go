package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/validator"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// ProjectRepoHandler manages CRUD for project_repos.
type ProjectRepoHandler struct {
	store *queries.ProjectReposStore
	hub   *websocket.Hub
}

func NewProjectRepoHandler(store *queries.ProjectReposStore, hub *websocket.Hub) *ProjectRepoHandler {
	return &ProjectRepoHandler{store: store, hub: hub}
}

func (h *ProjectRepoHandler) CreateRepo(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProjectRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateCreateProjectRepoRequest(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
		return
	}
	if req.Branch == "" {
		req.Branch = "main"
	}
	if req.Role == "" {
		req.Role = "code"
	}
	repo := models.ProjectRepo{
		ID:        uuid.New(),
		ProjectID: req.ProjectID,
		URL:       strings.TrimSpace(req.URL),
		Branch:    req.Branch,
		Role:      req.Role,
		AgentID:   req.AgentID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := h.store.Create(r.Context(), &repo); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(repo.ProjectID, "project_repo_added", repo)
	httpx.WriteJSON(w, http.StatusCreated, repo)
}

func (h *ProjectRepoHandler) ListRepos(w http.ResponseWriter, r *http.Request) {
	pidStr := r.URL.Query().Get("project_id")
	if pidStr == "" {
		httpx.WriteError(w, r, fmt.Errorf("%w: project_id query parameter is required", httpx.ErrBadRequest))
		return
	}
	pid, err := uuid.Parse(pidStr)
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid project_id: %w", err))
		return
	}
	out, err := h.store.ListByProject(r.Context(), pid)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if out == nil {
		out = []models.ProjectRepo{}
	}
	httpx.WriteJSON(w, http.StatusOK, out)
}

func (h *ProjectRepoHandler) GetRepo(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid repo id: %w", err))
		return
	}
	repo, err := h.store.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, repo)
}

func (h *ProjectRepoHandler) UpdateRepo(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid repo id: %w", err))
		return
	}
	var req models.CreateProjectRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid request body: %w", err))
		return
	}
	if err := validator.ValidateCreateProjectRepoRequest(&req); err != nil {
		httpx.WriteError(w, r, fmt.Errorf("validation: %w", err))
		return
	}
	current, err := h.store.Get(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	current.URL = strings.TrimSpace(req.URL)
	if req.Branch != "" {
		current.Branch = req.Branch
	}
	if req.Role != "" {
		current.Role = req.Role
	}
	current.AgentID = req.AgentID
	current.UpdatedAt = time.Now()
	if err := h.store.Update(r.Context(), current); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(current.ProjectID, "project_repo_updated", current)
	httpx.WriteJSON(w, http.StatusOK, current)
}

func (h *ProjectRepoHandler) DeleteRepo(w http.ResponseWriter, r *http.Request) {
	vars := muxVars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		httpx.WriteError(w, r, fmt.Errorf("invalid repo id: %w", err))
		return
	}
	pid, err := h.store.GetProjectID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	h.hub.BroadcastToProject(pid, "project_repo_deleted", map[string]uuid.UUID{"id": id})
	w.WriteHeader(http.StatusNoContent)
}
