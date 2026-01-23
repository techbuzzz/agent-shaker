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
}

// NewAgentCardHandler creates a new AgentCardHandler
func NewAgentCardHandler(version, baseURL string) *AgentCardHandler {
	return &AgentCardHandler{
		version: version,
		baseURL: baseURL,
	}
}

// ServeHTTP handles GET requests for /.well-known/agent-card.json
func (h *AgentCardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	card := h.generateAgentCard()

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(card); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// generateAgentCard creates the agent card with current capabilities
func (h *AgentCardHandler) generateAgentCard() models.AgentCard {
	return models.AgentCard{
		Name:        "Agent Shaker",
		Description: "MCP-compatible context management server with A2A support for AI agent coordination and task management",
		Version:     h.version,
		Capabilities: []models.Capability{
			{
				Type:        "task",
				Description: "Asynchronous task execution and management",
			},
			{
				Type:        "streaming",
				Description: "Real-time task updates via Server-Sent Events (SSE)",
			},
			{
				Type:        "artifacts",
				Description: "Markdown context sharing as A2A artifacts",
			},
			{
				Type:        "mcp",
				Description: "Model Context Protocol (MCP) support for tool integration",
			},
		},
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
