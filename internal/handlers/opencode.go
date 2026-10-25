package handlers

import (
	"net/http"

	"github.com/techbuzzz/agent-shaker/internal/configgen"
	"github.com/techbuzzz/agent-shaker/internal/httpx"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/publicurl"
)

// OpenCodeManifestHandler serves the organisation-wide OpenCode configuration.
//
// OpenCode fetches `/.well-known/opencode` by itself, before any credential
// exists, and merges what it finds underneath every other configuration
// source. It is the one place an organisation can hand a whole fleet of
// agents the same coordination server and the same plugin, without touching
// one machine.
//
// The document is rendered by the same generator that renders a per-agent
// bundle, so there is one implementation of the OpenCode shape rather than
// two that drift.
type OpenCodeManifestHandler struct {
	// baseURL is the operator's BASE_URL, read once at startup. Empty means
	// "derive the origin from each request", which is correct behind a proxy
	// that sets X-Forwarded-* and wrong for a deployment exposed directly.
	baseURL string
}

// NewOpenCodeManifestHandler returns a manifest handler.
func NewOpenCodeManifestHandler(baseURL string) *OpenCodeManifestHandler {
	return &OpenCodeManifestHandler{baseURL: baseURL}
}

// Manifest writes the OpenCode configuration an organisation publishes.
//
// GET /.well-known/opencode
//
// Unauthenticated on purpose: OpenCode fetches this before any credential
// exists, and the document carries no secret. Every credential in it is the
// {env:AGENT_SHAKER_API_KEY} placeholder, which is what makes it safe to
// serve from a public path.
//
// The connection it publishes is unbound — no project, no agent — because an
// organisation does not know which agent is about to connect. The agent
// registers itself with register_self, or the operator issues it a key.
func (h *OpenCodeManifestHandler) Manifest(w http.ResponseWriter, r *http.Request) {
	origin := publicurl.Resolve(h.baseURL, r)

	files, err := configgen.Render(
		configgen.Input{Origin: origin},
		models.ConfigFormatOpenCodeV2,
	)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// The origin can be derived from the request's host when BASE_URL is
	// unset, so a cached copy could be served to a different host.
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(files[0].Content))
}
