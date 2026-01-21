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

type ProjectHandler struct {
	db  *database.DB
	hub *websocket.Hub
}

func NewProjectHandler(db *database.DB, hub *websocket.Hub) *ProjectHandler {
	return &ProjectHandler{db: db, hub: hub}
}

func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req models.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validator.ValidateCreateProjectRequest(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	project := models.Project{
		ID:          uuid.New(),
		Name:        req.Name,
		Description: req.Description,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := h.db.Exec(`
		INSERT INTO projects (id, name, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, project.ID, project.Name, project.Description, project.Status, project.CreatedAt, project.UpdatedAt)
	if err != nil {
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT id, name, description, status, created_at, updated_at
		FROM projects
		ORDER BY created_at DESC
	`)
	if err != nil {
		http.Error(w, "Failed to retrieve projects", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			http.Error(w, "Failed to scan project", http.StatusInternalServerError)
			return
		}
		projects = append(projects, p)
	}

	// Return empty array instead of null
	if projects == nil {
		projects = []models.Project{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid project ID format", http.StatusBadRequest)
		return
	}

	var project models.Project
	err = h.db.QueryRow(`
		SELECT id, name, description, status, created_at, updated_at
		FROM projects
		WHERE id = $1
	`, id).Scan(&project.ID, &project.Name, &project.Description, &project.Status, &project.CreatedAt, &project.UpdatedAt)
	if err == sql.ErrNoRows {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "Failed to retrieve project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}
