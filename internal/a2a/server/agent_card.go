package server

import (
	"encoding/json"
	"net/http"

	"github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

// AgentCardHandler handles requests for the A2A agent card
type AgentCardHandler struct {
	version string
	baseURL string
	// authRequired mirrors the state of the API-key middleware. The card is a
	// discovery document: a client reads it to learn what credential it must
	// send, so it must not claim "none" on a deployment that returns 401.
	authRequired bool
}

// NewAgentCardHandler creates a new AgentCardHandler
func NewAgentCardHandler(version, baseURL string, authRequired bool) *AgentCardHandler {
	return &AgentCardHandler{
		version:      version,
		baseURL:      baseURL,
		authRequired: authRequired,
	}
}

// authSchemes describes the credentials a client must present.
//
// This used to advertise `none` unconditionally, which was accurate while every
// surface was open and became a lie the moment API-key auth landed: a client
// that trusted the card would send nothing and get a 401 on every call. The
// scheme is now derived from the actual middleware state.
func (h *AgentCardHandler) authSchemes() []models.AuthScheme {
	if !h.authRequired {
		return []models.AuthScheme{
			{
				Scheme:      "none",
				Description: "Public endpoints require no authentication (suitable for development and testing)",
			},
		}
	}
	return []models.AuthScheme{
		{
			Scheme:      "apiKey",
			Description: "Send the key in the X-API-Key header, or as an Authorization: Bearer <key> token.",
		},
	}
}

// ServeHTTP handles GET requests for /.well-known/agent-card.json
func (h *AgentCardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	card := h.generateAgentCard(resolvePublicBaseURL(h.baseURL, r))

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(card); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// generateAgentCard creates the agent card following the official A2A schema v1.0.
//
// baseURL is the origin to advertise, already resolved by
// resolvePublicBaseURL for the request being served. It may be empty, in which
// case the URLs are root-relative and still resolve correctly for a client that
// reached the server on the same origin.
func (h *AgentCardHandler) generateAgentCard(baseURL string) models.AgentCard {
	return models.AgentCard{
		// Required fields (official schema v1.0)
		SchemaVersion:   "1.0",
		HumanReadableID: "techbuzzz/agent-shaker",
		AgentVersion:    h.version,
		Name:            "Agent Shaker",
		Description:     "MCP-compatible context management server with A2A support for AI agent coordination, task management, and real-time collaboration",
		URL:             baseURL + "/a2a/v1",

		Provider: models.Provider{
			Name:           "techbuzzz",
			URL:            "https://github.com/techbuzzz/agent-shaker",
			SupportContact: "https://github.com/techbuzzz/agent-shaker/issues",
		},

		Capabilities: models.Capabilities{
			A2AVersion:                "1.0",
			MCPVersion:                "0.6",
			SupportedMessageParts:     []string{"text", "file", "data"},
			SupportsPushNotifications: true,
		},

		AuthSchemes: h.authSchemes(),

		// Optional fields
		Skills: []models.Skill{
			{
				ID:          "task_execution",
				Name:        "Asynchronous Task Execution",
				Description: "Execute tasks asynchronously with status tracking and result retrieval",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"message": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"content": map[string]string{"type": "string"},
								"format":  map[string]string{"type": "string", "enum": "[\"text\", \"markdown\"]"},
							},
							"required": []string{"content"},
						},
					},
					"required": []string{"message"},
				},
			},
			{
				ID:          "sse_streaming",
				Name:        "Server-Sent Events Streaming",
				Description: "Real-time task updates via Server-Sent Events for long-running operations",
			},
			{
				ID:          "artifact_sharing",
				Name:        "Artifact Sharing",
				Description: "Share markdown contexts as A2A artifacts for cross-agent knowledge transfer",
			},
			{
				ID:          "mcp_integration",
				Name:        "MCP Protocol Support",
				Description: "Model Context Protocol support for tool integration with AI assistants like GitHub Copilot",
			},
		},

		Tags: []string{
			"agent-coordination",
			"task-management",
			"mcp",
			"a2a",
			"context-sharing",
			"real-time",
			"streaming",
		},

		PrivacyPolicyURL:  "https://github.com/techbuzzz/agent-shaker/blob/main/docs/PRIVACY.md",
		TermsOfServiceURL: "https://github.com/techbuzzz/agent-shaker/blob/main/LICENSE",
		// /favicon.ico rather than the old /images/icon.png: the SPA is what
		// serves the public origin, and it ships exactly this file. The old path
		// resolved to nothing anywhere in the topology.
		IconURL: baseURL + "/favicon.ico",

		// Legacy fields for backward compatibility
		Version: h.version,
		Endpoints: []models.Endpoint{
			{
				Path:        "/a2a/v1/message",
				Method:      "POST",
				Description: "Send a new task to the agent",
				Protocol:    "A2A",
			},
			{
				Path:        "/a2a/v1/message:stream",
				Method:      "POST",
				Description: "Send a task and receive streaming updates via SSE",
				Protocol:    "A2A",
			},
			{
				Path:        "/a2a/v1/tasks",
				Method:      "GET",
				Description: "List all tasks with optional status filtering",
				Protocol:    "A2A",
				Params: map[string]string{
					"status": "Filter by task status (pending, running, completed, failed)",
					"limit":  "Maximum number of tasks to return",
					"offset": "Offset for pagination",
				},
			},
			{
				Path:        "/a2a/v1/tasks/{taskId}",
				Method:      "GET",
				Description: "Get details of a specific task",
				Protocol:    "A2A",
			},
			{
				Path:        "/a2a/v1/artifacts",
				Method:      "GET",
				Description: "List all available artifacts (contexts)",
				Protocol:    "A2A",
			},
			{
				Path:        "/a2a/v1/artifacts/{artifactId}",
				Method:      "GET",
				Description: "Get details and content of a specific artifact",
				Protocol:    "A2A",
			},
			{
				Path:        "/mcp",
				Method:      "POST",
				Description: "MCP JSON-RPC endpoint for tool invocation",
				Protocol:    "MCP",
			},
			{
				Path:        "/ws",
				Method:      "GET",
				Description: "WebSocket endpoint for real-time notifications",
				Protocol:    "WebSocket",
			},
		},
		Metadata: map[string]any{
			"supported_protocols":  []string{"A2A", "MCP", "WebSocket"},
			"websocket_available":  true,
			"sse_available":        true,
			"documentation_url":    "https://github.com/techbuzzz/agent-shaker",
			"contact":              "https://github.com/techbuzzz/agent-shaker/issues",
			"max_concurrent_tasks": 100,
		},
	}
}
