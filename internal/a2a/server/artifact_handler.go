package server

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

// ContextStorage defines the interface for retrieving contexts
type ContextStorage interface {
	ListContexts() ([]ContextData, error)
	GetContext(id string) (*ContextData, error)
}

// ContextData represents the internal context structure
type ContextData struct {
	ID          string
	Name        string
	Content     string
	Tags        []string
	Description string
	CreatedAt   string
	UpdatedAt   string
}

// ArtifactHandler handles A2A artifact endpoints
type ArtifactHandler struct {
	contextStorage ContextStorage
	baseURL        string
}

// NewArtifactHandler creates a new artifact handler
func NewArtifactHandler(cs ContextStorage, baseURL string) *ArtifactHandler {
	return &ArtifactHandler{
		contextStorage: cs,
		baseURL:        baseURL,
	}
}

// ListArtifacts handles GET /a2a/v1/artifacts
func (h *ArtifactHandler) ListArtifacts(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		h.writeCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	contexts, err := h.contextStorage.ListContexts()
	if err != nil {
		h.writeError(w, "Failed to list artifacts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	artifacts := make([]models.Artifact, len(contexts))
	for i, ctx := range contexts {
		artifacts[i] = h.contextToArtifact(&ctx)
	}

	resp := models.ArtifactListResponse{
		Artifacts:  artifacts,
		TotalCount: len(artifacts),
	}

	h.writeJSON(w, resp, http.StatusOK)
}

// GetArtifact handles GET /a2a/v1/artifacts/{artifactId}
func (h *ArtifactHandler) GetArtifact(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		h.writeCORSHeaders(w)
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodGet {
		h.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vars := mux.Vars(r)
	artifactID := vars["artifactId"]
	if artifactID == "" {
		h.writeError(w, "Artifact ID is required", http.StatusBadRequest)
		return
	}

	ctx, err := h.contextStorage.GetContext(artifactID)
	if err != nil {
		h.writeError(w, "Artifact not found", http.StatusNotFound)
		return
	}

	artifact := h.contextToArtifact(ctx)
	h.writeJSON(w, artifact, http.StatusOK)
}

// contextToArtifact converts a context to an A2A artifact
func (h *ArtifactHandler) contextToArtifact(ctx *ContextData) models.Artifact {
	return models.Artifact{
		ID:          ctx.ID,
		Name:        ctx.Name,
		Type:        "markdown",
		ContentType: "text/markdown",
		Content:     ctx.Content,
		URL:         h.baseURL + "/a2a/v1/artifacts/" + ctx.ID,
		Size:        int64(len(ctx.Content)),
		CreatedAt:   ctx.CreatedAt,
		Metadata: map[string]any{
			"tags":        ctx.Tags,
			"description": ctx.Description,
			"updated_at":  ctx.UpdatedAt,
		},
	}
}

// writeJSON writes a JSON response with the given status code
func (h *ArtifactHandler) writeJSON(w http.ResponseWriter, data any, statusCode int) {
	h.writeCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// writeError writes a JSON error response
func (h *ArtifactHandler) writeError(w http.ResponseWriter, message string, statusCode int) {
	h.writeCORSHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := map[string]string{"error": message}
	json.NewEncoder(w).Encode(errorResp)
}

// writeCORSHeaders writes CORS headers for the response
func (h *ArtifactHandler) writeCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
}
