package handlers

import (
	"log/slog"
	"net/http"

	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
)

// DashboardHandler handles dashboard statistics requests.
type DashboardHandler struct {
	store *queries.DashboardStore
}

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(store *queries.DashboardStore) *DashboardHandler {
	return &DashboardHandler{store: store}
}

// DashboardStats is the wire shape returned to clients. Fields mirror
// queries.DashboardStats buckets; keeping a separate handler type keeps the
// JSON envelope independent of the internal aggregate layout.
type DashboardStats struct {
	Projects ProjectStats `json:"projects"`
	Agents   AgentStats   `json:"agents"`
	Tasks    TaskStats    `json:"tasks"`
	Contexts ContextStats `json:"contexts"`
}

// ProjectStats represents project statistics.
type ProjectStats struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Archived int `json:"archived"`
}

// AgentStats represents agent statistics.
type AgentStats struct {
	Total   int `json:"total"`
	Active  int `json:"active"`
	Idle    int `json:"idle"`
	Offline int `json:"offline"`
}

// TaskStats represents task statistics.
type TaskStats struct {
	Total      int `json:"total"`
	Pending    int `json:"pending"`
	InProgress int `json:"in_progress"`
	Done       int `json:"done"`
	Blocked    int `json:"blocked"`
}

// ContextStats represents context statistics.
type ContextStats struct {
	Total int `json:"total"`
}

// GetDashboardStats returns aggregate counts for the dashboard view.
// If the database is unreachable the response is 503 with a clear error;
// if a single bucket's query fails the others are still returned so the UI
// renders a partial view.
func (h *DashboardHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	if !h.store.Available() {
		handleNoStore(w, r)
		return
	}
	internal, err := h.store.Stats(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "dashboard stats partial", "error", err)
		// Even on error we render the partial result so the UI is not blank.
	}
	httpx.WriteJSON(w, http.StatusOK, DashboardStats{
		Projects: ProjectStats{Total: internal.Projects.Total, Active: internal.Projects.Active, Archived: internal.Projects.Archived},
		Agents:   AgentStats{Total: internal.Agents.Total, Active: internal.Agents.Active, Idle: internal.Agents.Idle, Offline: internal.Agents.Offline},
		Tasks:    TaskStats{Total: internal.Tasks.Total, Pending: internal.Tasks.Pending, InProgress: internal.Tasks.InProgress, Done: internal.Tasks.Done, Blocked: internal.Tasks.Blocked},
		Contexts: ContextStats{Total: internal.Contexts.Total},
	})
}
