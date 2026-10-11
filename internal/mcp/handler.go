package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	a2aClient "github.com/techbuzzz/agent-shaker/internal/a2a/client"
	a2aModels "github.com/techbuzzz/agent-shaker/internal/a2a/models"
	"github.com/techbuzzz/agent-shaker/internal/database"
	"github.com/techbuzzz/agent-shaker/internal/database/queries"
	"github.com/techbuzzz/agent-shaker/internal/models"
	"github.com/techbuzzz/agent-shaker/internal/websocket"
)

// JSON-RPC 2.0 structures
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id,omitempty"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Protocol structures
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	InputSchema InputSchema `json:"inputSchema"`
}

type InputSchema struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Required   []string               `json:"required,omitempty"`
}

type ToolsListResult struct {
	Tools []Tool `json:"tools"`
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ResourcesListResult struct {
	Resources []Resource `json:"resources"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

type ResourcesReadResult struct {
	Contents []ResourceContent `json:"contents"`
}

type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type ToolResult struct {
	Content []ToolResultContent `json:"content"`
	IsError bool                `json:"isError,omitempty"`
}

type ToolResultContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCPHandler handles MCP protocol requests
type MCPHandler struct {
	db       *database.DB
	hub      *websocket.Hub
	sessions sync.Map
}

type Session struct {
	ID         string
	CreatedAt  time.Time
	ClientInfo map[string]interface{}
	ProjectID  string
	AgentID    string
}

// MCPContext holds the current request context (project/agent)
type MCPContext struct {
	ProjectID string
	AgentID   string
	// Context is the request's own context. It is carried alongside the
	// identifiers rather than replacing them: the identifiers are product
	// data, while this is what lets a tool call open a span as a child of the
	// HTTP request instead of a new root. It is never nil for contexts built
	// by extractContext; zero-valued MCPContext in tests falls back to
	// context.Background via requestContext.
	Context context.Context
}

// requestContext returns a usable context even for a zero-valued MCPContext, so
// callers never have to nil-check before starting a span.
func (c MCPContext) requestContext() context.Context {
	if c.Context != nil {
		return c.Context
	}
	return context.Background()
}

func NewMCPHandler(db *database.DB, hub *websocket.Hub) *MCPHandler {
	return &MCPHandler{
		db:  db,
		hub: hub,
	}
}

// extractContext extracts project_id and agent_id from URL params or headers
func (h *MCPHandler) extractContext(r *http.Request) MCPContext {
	ctx := MCPContext{Context: r.Context()}

	// Try URL query parameters first
	ctx.ProjectID = r.URL.Query().Get("project_id")
	ctx.AgentID = r.URL.Query().Get("agent_id")

	// Override with headers if present
	if headerProjectID := r.Header.Get("X-Project-ID"); headerProjectID != "" {
		ctx.ProjectID = headerProjectID
	}
	if headerAgentID := r.Header.Get("X-Agent-ID"); headerAgentID != "" {
		ctx.AgentID = headerAgentID
	}

	return ctx
}

// HandleMCP handles the main MCP endpoint with SSE support
func (h *MCPHandler) HandleMCP(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, X-Project-ID, X-Agent-ID")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Short-circuit when the server is running without a database. Every
	// tool handler below dereferences h.db, so without this guard the
	// Recovery middleware would catch a nil-pointer panic per request.
	if h.db == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":{"code":"unavailable","message":"database connection not available"}}`))
		return
	}

	// Extract context from URL/headers
	ctx := h.extractContext(r)
	if ctx.ProjectID != "" || ctx.AgentID != "" {
		slog.InfoContext(r.Context(), "mcp server info requested", "project_id", ctx.ProjectID, "agent_id", ctx.AgentID)
	}

	// Check for SSE request (GET with Accept: text/event-stream)
	if r.Method == "GET" {
		accept := r.Header.Get("Accept")
		if accept == "text/event-stream" {
			h.handleSSE(w, r, ctx)
			return
		}
		// Return server info for plain GET
		h.handleServerInfo(w, r, ctx)
		return
	}

	// Handle POST requests (JSON-RPC)
	if r.Method == "POST" {
		h.handleJSONRPC(w, r, ctx)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

func (h *MCPHandler) handleServerInfo(w http.ResponseWriter, r *http.Request, ctx MCPContext) {
	w.Header().Set("Content-Type", "application/json")

	info := map[string]interface{}{
		"name":            "agent-shaker",
		"version":         "1.0.0",
		"protocolVersion": mcpProtocolVersion,
		"capabilities": map[string]interface{}{
			"tools":     map[string]bool{"listChanged": false},
			"resources": map[string]bool{"subscribe": false, "listChanged": false},
		},
	}

	// Add context info if present
	if ctx.ProjectID != "" || ctx.AgentID != "" {
		info["context"] = map[string]string{
			"project_id": ctx.ProjectID,
			"agent_id":   ctx.AgentID,
		}
	}

	json.NewEncoder(w).Encode(info)
}

func (h *MCPHandler) handleSSE(w http.ResponseWriter, r *http.Request, ctx MCPContext) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Create session with context
	sessionID := uuid.New().String()
	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
		ProjectID: ctx.ProjectID,
		AgentID:   ctx.AgentID,
	}
	h.sessions.Store(sessionID, session)
	defer h.sessions.Delete(sessionID)

	slog.InfoContext(r.Context(), "mcp sse connection established",
		"session_id", sessionID, "project_id", ctx.ProjectID, "agent_id", ctx.AgentID)

	// Send initial endpoint message
	endpointMsg := fmt.Sprintf("event: endpoint\ndata: /mcp/message?sessionId=%s\n\n", sessionID)
	w.Write([]byte(endpointMsg))
	flusher.Flush()

	// Keep connection alive with periodic pings
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	done := r.Context().Done()
	for {
		select {
		case <-done:
			slog.InfoContext(r.Context(), "mcp sse connection closed", "session_id", sessionID)
			return
		case <-ticker.C:
			// Send ping to keep connection alive
			w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		}
	}
}

func (h *MCPHandler) handleJSONRPC(w http.ResponseWriter, r *http.Request, ctx MCPContext) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, nil, -32700, "Parse error", err.Error())
		return
	}

	slog.InfoContext(r.Context(), "mcp request", "method", req.Method, "id", req.ID,
		"project_id", ctx.ProjectID, "agent_id", ctx.AgentID)

	var result interface{}
	var rpcErr *JSONRPCError

	switch req.Method {
	case "initialize":
		result, rpcErr = h.handleInitialize(req.Params, ctx)
	case "initialized":
		// Client notification that initialization is complete
		result = map[string]interface{}{}
	case "tools/list":
		result, rpcErr = h.handleToolsList(ctx)
	case "tools/call":
		result, rpcErr = h.handleToolsCall(req.Params, ctx)
	case "resources/list":
		result, rpcErr = h.handleResourcesList()
	case "resources/read":
		result, rpcErr = h.handleResourcesRead(req.Params)
	case "ping":
		result = map[string]interface{}{}
	default:
		rpcErr = &JSONRPCError{
			Code:    -32601,
			Message: "Method not found",
			Data:    fmt.Sprintf("Unknown method: %s", req.Method),
		}
	}

	h.sendResponse(w, req.ID, result, rpcErr)
}

func (h *MCPHandler) handleInitialize(params json.RawMessage, ctx MCPContext) (interface{}, *JSONRPCError) {
	// Parse client info if provided
	var clientParams struct {
		ProtocolVersion string                 `json:"protocolVersion"`
		Capabilities    map[string]interface{} `json:"capabilities"`
		ClientInfo      map[string]interface{} `json:"clientInfo"`
	}
	if params != nil {
		// Best-effort: clientParams feeds a log line and nothing that changes
		// the handshake result, so a malformed params object must not fail
		// initialize. Say so rather than discarding the error silently.
		if err := json.Unmarshal(params, &clientParams); err != nil {
			slog.WarnContext(ctx.requestContext(), "mcp initialize params unparseable, continuing with defaults", "error", err)
		}
	}

	slog.InfoContext(ctx.requestContext(), "mcp initialize",
		"client", clientParams.ClientInfo,
		"protocol_version", clientParams.ProtocolVersion,
		"project_id", ctx.ProjectID,
		"agent_id", ctx.AgentID,
	)

	result := InitializeResult{
		ProtocolVersion: mcpProtocolVersion,
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false,
			},
			Resources: &ResourcesCapability{
				Subscribe:   false,
				ListChanged: false,
			},
		},
		ServerInfo: ServerInfo{
			Name:    "agent-shaker",
			Version: "1.0.0",
		},
	}

	return result, nil
}

func (h *MCPHandler) handleToolsList(ctx MCPContext) (interface{}, *JSONRPCError) {
	tools := []Tool{
		// Context-aware tools (use configured project/agent automatically)
		{
			Name:        "get_my_identity",
			Description: "Get the current agent's identity and assigned project based on MCP connection configuration",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		{
			Name:        "get_my_project",
			Description: "Get details of the project assigned to this MCP connection (requires project_id in connection URL)",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		{
			Name:        "get_my_tasks",
			Description: "Get tasks assigned to the current agent (requires agent_id in connection URL)",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Optional status filter (pending, in_progress, done, blocked)",
					},
				},
			},
		},
		{
			Name:        "update_my_status",
			Description: "Update the current agent's status (requires agent_id in connection URL)",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"description": "New status: idle, working, blocked, offline",
						"enum":        []string{"idle", "working", "blocked", "offline"},
					},
				},
				Required: []string{"status"},
			},
		},
		{
			Name:        "claim_task",
			Description: "Claim (assign to self) a task from the project (requires agent_id in connection URL)",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to claim",
					},
				},
				Required: []string{"task_id"},
			},
		},
		{
			Name:        "complete_task",
			Description: "Mark a task as done (requires agent_id in connection URL)",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to complete",
					},
				},
				Required: []string{"task_id"},
			},
		},
		{
			Name:        "reassign_task",
			Description: "Reassign a task to another agent",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to reassign",
					},
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the agent to assign the task to",
					},
				},
				Required: []string{"task_id", "agent_id"},
			},
		},
		// General tools (for exploring all data)
		{
			Name:        "list_projects",
			Description: "List all projects in the system",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		{
			Name:        "get_project",
			Description: "Get details of a specific project",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "The project ID (UUID)",
					},
				},
				Required: []string{"project_id"},
			},
		},
		{
			Name:        "list_agents",
			Description: "List all agents, optionally filtered by project",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional project ID to filter agents",
					},
				},
			},
		},
		{
			Name:        "get_agent",
			Description: "Get details of a specific agent",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "The agent ID (UUID)",
					},
				},
				Required: []string{"agent_id"},
			},
		},
		{
			Name:        "list_tasks",
			Description: "List tasks, optionally filtered by project or agent",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional project ID to filter tasks",
					},
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional agent ID to filter tasks",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Optional status filter (pending, in_progress, done, blocked)",
					},
				},
			},
		},
		{
			Name:        "create_task",
			Description: "Create a new task in a project. If connected with project_id and agent_id in URL, those will be used automatically.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "The project ID (optional if project_id in MCP connection URL)",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Task title",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Task description",
					},
					"priority": map[string]interface{}{
						"type":        "string",
						"description": "Priority: low, medium, high",
						"enum":        []string{"low", "medium", "high"},
					},
					"created_by": map[string]interface{}{
						"type":        "string",
						"description": "Agent ID who creates the task (optional, will use agent_id from URL or first agent)",
					},
					"assigned_to": map[string]interface{}{
						"type":        "string",
						"description": "Agent ID to assign the task to",
					},
					"milestone_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional milestone id to link this task to",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Optional tags. Use 'feature:<name>' to roll up into the Features view.",
					},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "update_task_status",
			Description: "Update the status of a task",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"description": "New status: pending, in_progress, done, blocked",
						"enum":        []string{"pending", "in_progress", "done", "blocked"},
					},
				},
				Required: []string{"task_id", "status"},
			},
		},
		{
			Name:        "list_contexts",
			Description: "List all documentation and contexts shared by agents in the project. Content is in markdown format for easy reading. Prefer search_contexts when you know what you are looking for: this returns everything in the project.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional project ID to filter contexts (uses connection URL context if not provided)",
					},
				},
			},
		},
		{
			Name: "search_contexts",
			Description: "Full-text search over a project's contexts, most relevant first. " +
				"Use this instead of list_contexts when you are looking for specific knowledge: " +
				"it returns only matching notes rather than the whole project. Matching is plain " +
				"word matching with no stemming, so search for the word as it was written. " +
				"Scoped to one project by design; pass the project explicitly if the connection " +
				"does not already carry it.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Words to look for in context titles and bodies. Multiple words all have to match.",
					},
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "Project to search within. Required: a context belongs to a project and search never crosses that boundary.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results. Defaults to 20 and is capped at 100.",
						"minimum":     1,
						"maximum":     100,
					},
				},
				Required: []string{"query", "project_id"},
			},
		},
		{
			Name:        "add_context",
			Description: "Add documentation or context to share with other agents in the project. Supports full markdown formatting for better readability. If connected with project_id and agent_id in URL, those will be used automatically. Other agents can read this context to understand your work.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "The project ID (optional if project_id in MCP connection URL)",
					},
					"agent_id": map[string]interface{}{
						"type":        "string",
						"description": "Agent ID who creates the context (optional, will use agent_id from URL or first agent)",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Context title - make it descriptive so other agents can find it",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Context content in markdown format. Use headings (# ## ###), code blocks (```), lists (- item), bold (**text**), italic (*text*), links ([text](url)), etc. This will be rendered beautifully for other agents to read.",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"description": "Tags for categorization",
						"items":       map[string]string{"type": "string"},
					},
				},
				Required: []string{"title", "content"},
			},
		},
		{
			Name:        "get_dashboard",
			Description: "Get dashboard statistics and overview",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		// A2A Integration tools
		{
			Name:        "discover_a2a_agent",
			Description: "Discover an external A2A agent by fetching its agent card. Returns the agent's capabilities, endpoints, and metadata.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_url": map[string]interface{}{
						"type":        "string",
						"description": "The base URL of the A2A agent to discover (e.g., https://agent.example.com)",
					},
				},
				Required: []string{"agent_url"},
			},
		},
		{
			Name:        "delegate_to_a2a_agent",
			Description: "Delegate a task to an external A2A agent. The agent will process the message and return a task ID for tracking.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_url": map[string]interface{}{
						"type":        "string",
						"description": "The base URL of the A2A agent",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "The message/task content to send to the agent",
					},
					"wait_for_completion": map[string]interface{}{
						"type":        "boolean",
						"description": "If true, wait for the task to complete before returning (default: false)",
					},
					"timeout_seconds": map[string]interface{}{
						"type":        "integer",
						"description": "Timeout in seconds when waiting for completion (default: 60)",
					},
				},
				Required: []string{"agent_url", "message"},
			},
		},
		{
			Name:        "get_a2a_task_status",
			Description: "Get the status of a task from an external A2A agent",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"agent_url": map[string]interface{}{
						"type":        "string",
						"description": "The base URL of the A2A agent",
					},
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "The task ID to check",
					},
				},
				Required: []string{"agent_url", "task_id"},
			},
		},

		// -----------------------------------------------------------------
		// Mesh app additions — Phase 1/2/3/4 (agent-shaker #mesh-plan)
		// -----------------------------------------------------------------
		// Agent self-registration. Use this the first time an agent joins
		// the mesh: the project_id/agent_id URL params then carry forward
		// for every subsequent MCP call.
		{
			Name:        "register_self",
			Description: "Register this agent against a project. Use this when the agent is connecting for the first time and you want the server to create an `agents` row (and optionally a `project_repos` row) on your behalf.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "Project ID (optional if project_id is in the MCP connection URL)",
					},
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Agent name (unique within the project)",
					},
					"role": map[string]interface{}{
						"type":        "string",
						"description": "Agent role: pm | backend | frontend (other strings accepted as free-text)",
						"enum":        []string{"pm", "backend", "frontend"},
					},
					"team": map[string]interface{}{
						"type":        "string",
						"description": "Optional team label (e.g. 'auth', 'growth')",
					},
					"repo_url": map[string]interface{}{
						"type":        "string",
						"description": "Optional git URL — when provided, a project_repos row is created and linked to this agent",
					},
					"repo_branch": map[string]interface{}{
						"type":        "string",
						"description": "Branch for the repo (default 'main')",
					},
				},
				Required: []string{"name", "role"},
			},
		},

		// Milestones — PM-side tool. Returns 403 for non-PM agents.
		{
			Name:        "create_milestone",
			Description: "PM-only: create a milestone for the connected project. Milestones group tasks and have status planned | active | done | dropped.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "Project ID (optional if project_id is in the MCP connection URL)",
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "Milestone title (max 255 chars)",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Long-form description (markdown OK)",
					},
					"status": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"planned", "active", "done", "dropped"},
						"description": "Initial status (default 'planned')",
					},
					"target_date": map[string]interface{}{
						"type":        "string",
						"description": "Optional target date (ISO 8601 YYYY-MM-DD)",
					},
				},
				Required: []string{"title"},
			},
		},
		{
			Name:        "list_milestones",
			Description: "List milestones for the connected project.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "Project ID (optional if project_id is in the MCP connection URL)",
					},
				},
			},
		},
		{
			Name:        "assign_task_to_milestone",
			Description: "PM-only: link an existing task to a milestone.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"task_id":      map[string]interface{}{"type": "string"},
					"milestone_id": map[string]interface{}{"type": "string"},
				},
				Required: []string{"task_id", "milestone_id"},
			},
		},

		// Global context (server-wide playbook) — PM write side.
		{
			Name:        "publish_global_context",
			Description: "PM-only: publish a markdown playbook or note visible to every agent on this server (scope='global') or only to the connected project (scope='project').",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"scope": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"global", "project"},
						"description": "Either 'global' (server-wide) or 'project' (this project only)",
					},
					"project_id": map[string]interface{}{
						"type":        "string",
						"description": "Required when scope='project'; ignored otherwise",
					},
					"title": map[string]interface{}{"type": "string"},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "Markdown content",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Optional tags for grouping",
					},
				},
				Required: []string{"scope", "title", "content"},
			},
		},
		{
			Name:        "list_global_contexts",
			Description: "List server-wide (scope='global') and/or project-scoped docs. Useful before reading one via read_global_context.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"scope":      map[string]interface{}{"type": "string", "enum": []string{"global", "project"}},
					"project_id": map[string]interface{}{"type": "string"},
					"tag_prefix": map[string]interface{}{"type": "string"},
				},
			},
		},
		{
			Name:        "read_global_context",
			Description: "Read one global or project-scoped doc by id, or one global playbook by title (use title='On-call playbook' style).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"id":    map[string]interface{}{"type": "string", "description": "Doc id (UUID) — preferred"},
					"title": map[string]interface{}{"type": "string", "description": "Title of a scope='global' doc"},
				},
			},
		},
	}

	return ToolsListResult{Tools: tools}, nil
}

func (h *MCPHandler) handleToolsCall(params json.RawMessage, ctx MCPContext) (interface{}, *JSONRPCError) {
	var callParams ToolCallParams
	if err := json.Unmarshal(params, &callParams); err != nil {
		return nil, &JSONRPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	// One span per tool call, parented to the HTTP request so a tool call
	// shows up inside the request that carried it instead of as a separate
	// trace root.
	//
	// Attribute names carry the shaker. prefix on purpose. The gen_ai.* keys
	// are all still at Development stability, so names borrowed from there
	// would be free to change under us; db.system is the one borrowed name,
	// and it is stable.
	spanAttrs := []attribute.KeyValue{
		attribute.String("shaker.mcp.tool", callParams.Name),
		attribute.String("shaker.mcp.protocol_version", mcpProtocolVersion),
	}
	if ctx.ProjectID != "" {
		spanAttrs = append(spanAttrs, attribute.String("shaker.project.id", ctx.ProjectID))
	}
	if ctx.AgentID != "" {
		spanAttrs = append(spanAttrs, attribute.String("shaker.agent.id", ctx.AgentID))
	}
	spanCtx, span := mcpTracer.Start(ctx.requestContext(), "mcp.tools.call", trace.WithAttributes(spanAttrs...))
	defer span.End()

	// Log the shape of the call, not its payload. Tool arguments carry task
	// titles, context bodies and standup text; writing them to the log stream
	// duplicates customer content into wherever logs are shipped.
	slog.InfoContext(spanCtx, "mcp tool call",
		"tool", callParams.Name,
		"project_id", ctx.ProjectID,
		"agent_id", ctx.AgentID,
		"argument_keys", argumentKeys(callParams.Arguments),
	)

	var resultText string
	var isError bool

	switch callParams.Name {
	// Context-aware tools
	case "get_my_identity":
		resultText, isError = h.executeGetMyIdentity(ctx)
	case "get_my_project":
		resultText, isError = h.executeGetMyProject(ctx)
	case "get_my_tasks":
		resultText, isError = h.executeGetMyTasks(callParams.Arguments, ctx)
	case "update_my_status":
		resultText, isError = h.executeUpdateMyStatus(callParams.Arguments, ctx)
	case "claim_task":
		resultText, isError = h.executeClaimTask(callParams.Arguments, ctx)
	case "complete_task":
		resultText, isError = h.executeCompleteTask(callParams.Arguments, ctx)
	case "reassign_task":
		resultText, isError = h.executeReassignTask(callParams.Arguments)
	// General tools
	case "list_projects":
		resultText, isError = h.executeListProjects()
	case "get_project":
		resultText, isError = h.executeGetProject(callParams.Arguments)
	case "list_agents":
		resultText, isError = h.executeListAgents(callParams.Arguments)
	case "get_agent":
		resultText, isError = h.executeGetAgent(callParams.Arguments)
	case "list_tasks":
		resultText, isError = h.executeListTasks(callParams.Arguments)
	case "create_task":
		resultText, isError = h.executeCreateTask(callParams.Arguments, ctx)
	case "update_task_status":
		resultText, isError = h.executeUpdateTaskStatus(callParams.Arguments)
	case "list_contexts":
		resultText, isError = h.executeListContexts(callParams.Arguments)
	case "search_contexts":
		resultText, isError = h.executeSearchContexts(callParams.Arguments)
	case "add_context":
		resultText, isError = h.executeAddContext(callParams.Arguments, ctx)
	case "get_dashboard":
		resultText, isError = h.executeGetDashboard()
	// A2A Integration tools
	case "discover_a2a_agent":
		resultText, isError = h.executeDiscoverA2AAgent(callParams.Arguments)
	case "delegate_to_a2a_agent":
		resultText, isError = h.executeDelegateToA2AAgent(callParams.Arguments)
	case "get_a2a_task_status":
		resultText, isError = h.executeGetA2ATaskStatus(callParams.Arguments)
	// Mesh app additions — Phase 1/2/3/4
	case "register_self":
		resultText, isError = h.executeRegisterSelf(callParams.Arguments, ctx)
	case "create_milestone":
		resultText, isError = h.executeCreateMilestone(callParams.Arguments, ctx)
	case "list_milestones":
		resultText, isError = h.executeListMilestones(callParams.Arguments, ctx)
	case "assign_task_to_milestone":
		resultText, isError = h.executeAssignTaskToMilestone(callParams.Arguments, ctx)
	case "publish_global_context":
		resultText, isError = h.executePublishGlobalContext(callParams.Arguments, ctx)
	case "list_global_contexts":
		resultText, isError = h.executeListGlobalContexts(callParams.Arguments, ctx)
	case "read_global_context":
		resultText, isError = h.executeReadGlobalContext(callParams.Arguments)
	default:
		span.RecordError(fmt.Errorf("unknown tool: %s", callParams.Name))
		span.SetStatus(codes.Error, "unknown tool")
		return nil, &JSONRPCError{
			Code:    -32601,
			Message: "Unknown tool",
			Data:    fmt.Sprintf("Tool not found: %s", callParams.Name),
		}
	}

	// A tool that returns IsError is a failed call even though the JSON-RPC
	// envelope is well formed, so the span has to say so. Otherwise a trace
	// reads as a clean run while the tool was refusing the work.
	span.SetAttributes(attribute.Bool("shaker.mcp.tool_error", isError))
	if isError {
		span.SetStatus(codes.Error, "tool reported an error")
	}

	return ToolResult{
		Content: []ToolResultContent{
			{Type: "text", Text: resultText},
		},
		IsError: isError,
	}, nil
}

func (h *MCPHandler) handleResourcesList() (interface{}, *JSONRPCError) {
	resources := []Resource{
		{
			URI:         "agent-shaker://projects",
			Name:        "Projects",
			Description: "List of all projects",
			MimeType:    "application/json",
		},
		{
			URI:         "agent-shaker://agents",
			Name:        "Agents",
			Description: "List of all agents",
			MimeType:    "application/json",
		},
		{
			URI:         "agent-shaker://tasks",
			Name:        "Tasks",
			Description: "List of all tasks",
			MimeType:    "application/json",
		},
		{
			URI:         "agent-shaker://dashboard",
			Name:        "Dashboard",
			Description: "Dashboard statistics",
			MimeType:    "application/json",
		},
	}

	// Expose every global playbook as `global://<title>`. This is what
	// lets any agent on any project read server-wide playbooks via the MCP
	// `resources/read` path.
	if h.db != nil {
		rows, err := h.db.Query(`SELECT title FROM global_contexts WHERE scope = 'global' ORDER BY title`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var title string
				if err := rows.Scan(&title); err != nil {
					continue
				}
				resources = append(resources, Resource{
					URI:         "global://" + title,
					Name:        title,
					Description: "Server-wide playbook (global_contexts row)",
					MimeType:    "text/markdown",
				})
			}
		}
	}

	return ResourcesListResult{Resources: resources}, nil
}

func (h *MCPHandler) handleResourcesRead(params json.RawMessage) (interface{}, *JSONRPCError) {
	var readParams struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(params, &readParams); err != nil {
		return nil, &JSONRPCError{
			Code:    -32602,
			Message: "Invalid params",
			Data:    err.Error(),
		}
	}

	var content string
	var isError bool
	var mimeType = "application/json"

	switch {
	case readParams.URI == "agent-shaker://projects":
		content, isError = h.executeListProjects()
	case readParams.URI == "agent-shaker://agents":
		content, isError = h.executeListAgents(nil)
	case readParams.URI == "agent-shaker://tasks":
		content, isError = h.executeListTasks(nil)
	case readParams.URI == "agent-shaker://dashboard":
		content, isError = h.executeGetDashboard()
	case strings.HasPrefix(readParams.URI, "global://"):
		mimeType = "text/markdown"
		title := strings.TrimPrefix(readParams.URI, "global://")
		content, isError = h.executeReadGlobalContextByTitle(title)
	default:
		return nil, &JSONRPCError{
			Code:    -32602,
			Message: "Unknown resource",
			Data:    fmt.Sprintf("Resource not found: %s", readParams.URI),
		}
	}

	if isError {
		return nil, &JSONRPCError{
			Code:    -32000,
			Message: "Resource read failed",
			Data:    content,
		}
	}

	return ResourcesReadResult{
		Contents: []ResourceContent{
			{
				URI:      readParams.URI,
				MimeType: mimeType,
				Text:     content,
			},
		},
	}, nil
}

// Tool execution methods
func (h *MCPHandler) executeListProjects() (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	rows, err := h.db.Query(`
		SELECT id, name, description, status, created_at, updated_at 
		FROM projects ORDER BY created_at DESC
	`)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	defer rows.Close()

	var projects []map[string]interface{}
	for rows.Next() {
		var id, name, description, status string
		var createdAt, updatedAt interface{}
		if err := rows.Scan(&id, &name, &description, &status, &createdAt, &updatedAt); err != nil {
			continue
		}
		projects = append(projects, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": description,
			"status":      status,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
	}

	result, _ := json.MarshalIndent(projects, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeGetProject(args map[string]interface{}) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	projectID, ok := args["project_id"].(string)
	if !ok {
		return `{"error": "project_id is required"}`, true
	}

	var id, name, description, status string
	var createdAt, updatedAt interface{}
	err := h.db.QueryRow(`
		SELECT id, name, description, status, created_at, updated_at 
		FROM projects WHERE id = $1
	`, projectID).Scan(&id, &name, &description, &status, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"id":          id,
		"name":        name,
		"description": description,
		"status":      status,
		"created_at":  createdAt,
		"updated_at":  updatedAt,
	}, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeListAgents(args map[string]interface{}) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	query := `SELECT id, project_id, name, role, status, team, created_at FROM agents`
	var queryArgs []interface{}

	if args != nil {
		if projectID, ok := args["project_id"].(string); ok && projectID != "" {
			query += " WHERE project_id = $1"
			queryArgs = append(queryArgs, projectID)
		}
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.db.Query(query, queryArgs...)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	defer rows.Close()

	var agents []map[string]interface{}
	for rows.Next() {
		var id, projectID, name, role, status string
		var team *string
		var createdAt interface{}
		if err := rows.Scan(&id, &projectID, &name, &role, &status, &team, &createdAt); err != nil {
			continue
		}
		agent := map[string]interface{}{
			"id":         id,
			"project_id": projectID,
			"name":       name,
			"role":       role,
			"status":     status,
			"created_at": createdAt,
		}
		if team != nil {
			agent["team"] = *team
		}
		agents = append(agents, agent)
	}

	result, _ := json.MarshalIndent(agents, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeGetAgent(args map[string]interface{}) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	agentID, ok := args["agent_id"].(string)
	if !ok {
		return `{"error": "agent_id is required"}`, true
	}

	var id, projectID, name, role, status string
	var team *string
	var createdAt interface{}
	err := h.db.QueryRow(`
		SELECT id, project_id, name, role, status, team, created_at 
		FROM agents WHERE id = $1
	`, agentID).Scan(&id, &projectID, &name, &role, &status, &team, &createdAt)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	agent := map[string]interface{}{
		"id":         id,
		"project_id": projectID,
		"name":       name,
		"role":       role,
		"status":     status,
		"created_at": createdAt,
	}
	if team != nil {
		agent["team"] = *team
	}

	result, _ := json.MarshalIndent(agent, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeListTasks(args map[string]interface{}) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	query := `SELECT id, project_id, title, description, status, priority, assigned_to, milestone_id, created_at FROM tasks WHERE 1=1`
	var queryArgs []interface{}
	argNum := 1

	if args != nil {
		if projectID, ok := args["project_id"].(string); ok && projectID != "" {
			query += fmt.Sprintf(" AND project_id = $%d", argNum)
			queryArgs = append(queryArgs, projectID)
			argNum++
		}
		if agentID, ok := args["agent_id"].(string); ok && agentID != "" {
			query += fmt.Sprintf(" AND assigned_to = $%d", argNum)
			queryArgs = append(queryArgs, agentID)
			argNum++
		}
		if status, ok := args["status"].(string); ok && status != "" {
			query += fmt.Sprintf(" AND status = $%d", argNum)
			queryArgs = append(queryArgs, status)
			// No argNum++ here: this is the last optional filter, so the next
			// placeholder number is never computed. The increment only matters
			// for filters that follow it.
		}
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.db.Query(query, queryArgs...)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	defer rows.Close()

	var tasks []map[string]interface{}
	for rows.Next() {
		var id, projectID, title, status, priority string
		var description, assignedTo, milestoneID *string
		var createdAt interface{}
		if err := rows.Scan(&id, &projectID, &title, &description, &status, &priority, &assignedTo, &milestoneID, &createdAt); err != nil {
			continue
		}
		task := map[string]interface{}{
			"id":         id,
			"project_id": projectID,
			"title":      title,
			"status":     status,
			"priority":   priority,
			"created_at": createdAt,
		}
		if description != nil {
			task["description"] = *description
		}
		if assignedTo != nil {
			task["assigned_to"] = *assignedTo
		}
		if milestoneID != nil {
			task["milestone_id"] = *milestoneID
		}
		tasks = append(tasks, task)
	}

	result, _ := json.MarshalIndent(tasks, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeCreateTask(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	projectID, ok := args["project_id"].(string)
	if !ok || projectID == "" {
		// Use project_id from context if not provided in args
		if ctx.ProjectID != "" {
			projectID = ctx.ProjectID
		} else {
			return `{"error": "project_id is required"}`, true
		}
	}

	title, ok := args["title"].(string)
	if !ok {
		return `{"error": "title is required"}`, true
	}

	description, _ := args["description"].(string)
	priority, _ := args["priority"].(string)
	if priority == "" {
		priority = "medium"
	}
	assignedTo, _ := args["assigned_to"].(string)
	createdBy, _ := args["created_by"].(string)

	// Use agent_id from context if created_by not provided
	if createdBy == "" && ctx.AgentID != "" {
		createdBy = ctx.AgentID
	}

	// If still no created_by, try to use the first agent from the project
	if createdBy == "" {
		err := h.db.QueryRow(`SELECT id FROM agents WHERE project_id = $1 LIMIT 1`, projectID).Scan(&createdBy)
		if err != nil {
			return `{"error": "created_by is required or no agents found in project"}`, true
		}
	}

	// Use agent_id from context if assigned_to not provided (agent assigns task to themselves)
	if assignedTo == "" && ctx.AgentID != "" {
		assignedTo = ctx.AgentID
	}

	id := uuid.New().String()
	milestoneID, _ := args["milestone_id"].(string)
	var milestonePtr *string
	if milestoneID != "" {
		milestonePtr = &milestoneID
	}
	query := `INSERT INTO tasks (id, project_id, title, description, status, priority, created_by, assigned_to, milestone_id)
	          VALUES ($1, $2, $3, $4, 'pending', $5, $6, $7, $8) RETURNING id, created_at`

	var createdID string
	var createdAt interface{}
	var assignedToPtr *string
	if assignedTo != "" {
		assignedToPtr = &assignedTo
	}

	err := h.db.QueryRow(query, id, projectID, title, description, priority, createdBy, assignedToPtr, milestonePtr).Scan(&createdID, &createdAt)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	responseData := map[string]interface{}{
		"success":    true,
		"id":         createdID,
		"title":      title,
		"status":     "pending",
		"priority":   priority,
		"created_by": createdBy,
		"created_at": createdAt,
	}
	if milestonePtr != nil {
		responseData["milestone_id"] = *milestonePtr
	}

	// Include assigned_to in response if it was set
	if assignedTo != "" {
		responseData["assigned_to"] = assignedTo
	}

	result, _ := json.MarshalIndent(responseData, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeUpdateTaskStatus(args map[string]interface{}) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	taskID, ok := args["task_id"].(string)
	if !ok {
		return `{"error": "task_id is required"}`, true
	}
	status, ok := args["status"].(string)
	if !ok {
		return `{"error": "status is required"}`, true
	}

	// Validate status
	validStatuses := map[string]bool{"pending": true, "in_progress": true, "done": true, "blocked": true}
	if !validStatuses[status] {
		return `{"error": "invalid status, must be one of: pending, in_progress, done, blocked"}`, true
	}

	_, err := h.db.Exec(`UPDATE tasks SET status = $1, updated_at = NOW() WHERE id = $2`, status, taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"success": true,
		"task_id": taskID,
		"status":  status,
	}, "", "  ")
	return string(result), false
}

// executeSearchContexts runs a full-text search over a project's contexts.
//
// This exists because list_contexts forces a choice between two bad options
// for an agent that needs one specific note: read every context in the project,
// or guess. The search is always scoped to a single project — a context is a
// project's knowledge, and a cross-project hit would put one team's notes into
// another team's task.
func (h *MCPHandler) executeSearchContexts(args map[string]interface{}) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	rawQuery, _ := args["query"].(string)
	trimmed := strings.TrimSpace(rawQuery)
	if trimmed == "" {
		// An empty query returns no results rather than the whole project.
		// The store enforces the same rule, and both have to: an agent that
		// calls this with no query should get an empty list, not every
		// context it was trying to avoid reading.
		return `{"contexts": [], "count": 0}`, false
	}

	projectID, _ := args["project_id"].(string)
	if projectID == "" {
		return `{"error": "project_id is required: context search is always scoped to one project"}`, true
	}

	limit := queries.DefaultSearchLimit
	if raw, ok := args["limit"].(float64); ok {
		limit = int(raw)
	}
	if limit <= 0 {
		limit = queries.DefaultSearchLimit
	}
	if limit > queries.MaxSearchLimit {
		limit = queries.MaxSearchLimit
	}

	// websearch_to_tsquery never raises on malformed input, which is the right
	// behaviour here: the query text comes from an agent, and a stray quote
	// should cost it results rather than produce an error it cannot parse.
	const searchSQL = `
		SELECT c.id, c.title, c.content, c.tags, c.created_at,
		       a.name as agent_name,
		       ts_rank(c.search_vector, websearch_to_tsquery('simple', $2)) as rank
		FROM contexts c
		LEFT JOIN agents a ON c.agent_id = a.id
		WHERE c.project_id = $1
		  AND c.search_vector @@ websearch_to_tsquery('simple', $2)
		ORDER BY rank DESC, c.created_at DESC
		LIMIT $3`

	rows, err := h.db.Query(searchSQL, projectID, trimmed, limit)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	defer rows.Close()

	contexts := []map[string]interface{}{}
	for rows.Next() {
		var id, title, createdAt string
		var content, agentName *string
		var tags interface{}
		var rank float64
		if err := rows.Scan(&id, &title, &content, &tags, &createdAt, &agentName, &rank); err != nil {
			return fmt.Sprintf(`{"error": "failed to read search result: %s"}`, err.Error()), true
		}
		entry := map[string]interface{}{
			"id":         id,
			"title":      title,
			"created_at": createdAt,
			"rank":       rank,
		}
		// Nullable columns are omitted rather than sent as JSON null: an agent
		// reading this should not have to distinguish "no body" from "the
		// read failed" by looking at a null.
		if content != nil {
			entry["content"] = *content
		}
		if agentName != nil {
			entry["agent_name"] = *agentName
		}
		if tags != nil {
			entry["tags"] = tags
		}
		contexts = append(contexts, entry)
	}
	if err := rows.Err(); err != nil {
		return fmt.Sprintf(`{"error": "failed to read search results: %s"}`, err.Error()), true
	}

	result, err := json.MarshalIndent(map[string]interface{}{
		"contexts": contexts,
		"count":    len(contexts),
		"query":    trimmed,
	}, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to encode results: %s"}`, err.Error()), true
	}
	return string(result), false
}

func (h *MCPHandler) executeListContexts(args map[string]interface{}) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	query := `SELECT c.id, c.project_id, c.agent_id, a.name as agent_name, c.title, c.content, c.tags, c.created_at 
	          FROM contexts c 
	          LEFT JOIN agents a ON c.agent_id = a.id`
	var queryArgs []interface{}

	if args != nil {
		if projectID, ok := args["project_id"].(string); ok && projectID != "" {
			query += " WHERE c.project_id = $1"
			queryArgs = append(queryArgs, projectID)
		}
	}
	query += " ORDER BY c.created_at DESC"

	rows, err := h.db.Query(query, queryArgs...)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	defer rows.Close()

	var contexts []map[string]interface{}
	for rows.Next() {
		var id, projectID, agentID, title, content string
		var agentName *string
		var tags interface{}
		var createdAt interface{}
		if err := rows.Scan(&id, &projectID, &agentID, &agentName, &title, &content, &tags, &createdAt); err != nil {
			continue
		}

		// Create a preview of the content
		preview := content
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}

		agentNameStr := "Unknown"
		if agentName != nil {
			agentNameStr = *agentName
		}

		contexts = append(contexts, map[string]interface{}{
			"id":         id,
			"project_id": projectID,
			"agent_id":   agentID,
			"agent_name": agentNameStr,
			"title":      title,
			"content":    content,
			"preview":    preview,
			"format":     "markdown",
			"tags":       tags,
			"created_at": createdAt,
		})
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"contexts": contexts,
		"count":    len(contexts),
		"note":     "Content is in markdown format - render it for best readability",
	}, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeAddContext(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	projectID, ok := args["project_id"].(string)
	if !ok || projectID == "" {
		// Use project_id from context if not provided in args
		if ctx.ProjectID != "" {
			projectID = ctx.ProjectID
		} else {
			return `{"error": "project_id is required"}`, true
		}
	}

	title, ok := args["title"].(string)
	if !ok {
		return `{"error": "title is required"}`, true
	}
	content, ok := args["content"].(string)
	if !ok {
		return `{"error": "content is required"}`, true
	}

	agentID, _ := args["agent_id"].(string)

	// Use agent_id from context if not provided in args
	if agentID == "" && ctx.AgentID != "" {
		agentID = ctx.AgentID
	}

	// If still no agent_id, try to use the first agent from the project
	if agentID == "" {
		err := h.db.QueryRow(`SELECT id FROM agents WHERE project_id = $1 LIMIT 1`, projectID).Scan(&agentID)
		if err != nil {
			return `{"error": "agent_id is required or no agents found in project"}`, true
		}
	}

	// Convert tags to PostgreSQL array format using pq.Array
	var tags []string
	if tagsInterface, ok := args["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	id := uuid.New().String()
	query := `INSERT INTO contexts (id, project_id, agent_id, title, content, tags) VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`

	var createdAt interface{}
	// Use pq.Array for proper PostgreSQL array handling
	err := h.db.QueryRow(query, id, projectID, agentID, title, content, pq.Array(tags)).Scan(&createdAt)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	// Create a preview of the content (first 200 chars)
	preview := content
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	// Get agent name for better feedback. A miss is expected — the agent may
	// have been deleted, or the row may not exist yet — and the fallback below
	// covers it, so the error itself only needs to be visible in the log.
	var agentName string
	if err := h.db.QueryRow(`SELECT name FROM agents WHERE id = $1`, agentID).Scan(&agentName); err != nil {
		slog.WarnContext(ctx.requestContext(), "could not resolve agent name for context", "agent_id", agentID, "error", err)
	}
	if agentName == "" {
		agentName = "Unknown Agent"
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"success":     true,
		"id":          id,
		"title":       title,
		"agent_id":    agentID,
		"agent_name":  agentName,
		"tags":        tags,
		"preview":     preview,
		"format":      "markdown",
		"created_at":  createdAt,
		"shared_with": "All agents in the project can now read this context",
	}, "", "  ")
	return string(result), false
}

// countRows returns the result of a COUNT(*) query, or 0 when it fails.
//
// This dashboard used to do `h.db.QueryRow(...).Scan(&n)` eight times with the
// error discarded. A failed scan leaves n at its previous value while the rest
// of the response is assembled from the others, so a transient database error
// rendered as a confident, internally inconsistent dashboard — some counters
// fresh, some silently stale — rather than as an error the caller could see.
func countRows(db *database.DB, query string, args ...any) int {
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		// No context to correlate: countRows is a package-level helper with
		// thirteen call sites, and threading one through every dashboard
		// counter to enrich a swallowed error is not worth the signature churn.
		// The queries are static COUNT(*) strings, so logging one leaks nothing.
		slog.Error("mcp dashboard count failed", "error", err, "query", query)
		return 0
	}
	return n
}

func (h *MCPHandler) executeGetDashboard() (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	var projectCount, agentCount, taskCount, contextCount int
	var pendingTasks, inProgressTasks, doneTasks, blockedTasks int

	projectCount = countRows(h.db, "SELECT COUNT(*) FROM projects")
	agentCount = countRows(h.db, "SELECT COUNT(*) FROM agents")
	taskCount = countRows(h.db, "SELECT COUNT(*) FROM tasks")
	contextCount = countRows(h.db, "SELECT COUNT(*) FROM contexts")

	pendingTasks = countRows(h.db, "SELECT COUNT(*) FROM tasks WHERE status = 'pending'")
	inProgressTasks = countRows(h.db, "SELECT COUNT(*) FROM tasks WHERE status = 'in_progress'")
	doneTasks = countRows(h.db, "SELECT COUNT(*) FROM tasks WHERE status = 'done'")
	blockedTasks = countRows(h.db, "SELECT COUNT(*) FROM tasks WHERE status = 'blocked'")

	result, _ := json.MarshalIndent(map[string]interface{}{
		"projects":          projectCount,
		"agents":            agentCount,
		"tasks":             taskCount,
		"contexts":          contextCount,
		"pending_tasks":     pendingTasks,
		"in_progress_tasks": inProgressTasks,
		"done_tasks":        doneTasks,
		"blocked_tasks":     blockedTasks,
	}, "", "  ")
	return string(result), false
}

// A2A Integration tool implementations

func (h *MCPHandler) executeDiscoverA2AAgent(args map[string]interface{}) (string, bool) {
	agentURL, ok := args["agent_url"].(string)
	if !ok || agentURL == "" {
		return `{"error": "agent_url is required"}`, true
	}

	// Create A2A client and discover agent
	client := createA2AClient()
	card, err := client.Discover(context.Background(), agentURL)
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to discover agent: %s"}`, err.Error()), true
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"success":      true,
		"agent_url":    agentURL,
		"name":         card.Name,
		"description":  card.Description,
		"version":      card.Version,
		"capabilities": card.Capabilities,
		"endpoints":    card.Endpoints,
		"metadata":     card.Metadata,
	}, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeDelegateToA2AAgent(args map[string]interface{}) (string, bool) {
	agentURL, ok := args["agent_url"].(string)
	if !ok || agentURL == "" {
		return `{"error": "agent_url is required"}`, true
	}

	message, ok := args["message"].(string)
	if !ok || message == "" {
		return `{"error": "message is required"}`, true
	}

	waitForCompletion := false
	if wait, ok := args["wait_for_completion"].(bool); ok {
		waitForCompletion = wait
	}

	timeoutSeconds := 60
	if timeout, ok := args["timeout_seconds"].(float64); ok {
		timeoutSeconds = int(timeout)
	}

	// Create A2A client
	client := createA2AClient()

	// Send message
	req := &a2aModels.SendMessageRequest{
		Message: a2aModels.Message{
			Content: message,
			Format:  "text",
		},
	}

	resp, err := client.SendMessage(context.Background(), agentURL, req)
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to send message: %s"}`, err.Error()), true
	}

	result := map[string]interface{}{
		"success":    true,
		"agent_url":  agentURL,
		"task_id":    resp.TaskID,
		"status":     resp.Status,
		"created_at": resp.CreatedAt,
	}

	// If waiting for completion, poll until done
	if waitForCompletion {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
		defer cancel()

		task, err := pollTaskUntilComplete(ctx, client, agentURL, resp.TaskID)
		if err != nil {
			result["wait_error"] = err.Error()
			result["final_status"] = "unknown"
			// Phase 5: even on timeout, persist a partial global_contexts row
			// so PMs see it on the global page. Tagged a2a:timeout so it
			// stands out in the GlobalContextCard.
			h.landA2AArtifact(agentURL, resp.TaskID, ctx, "timeout: "+err.Error())
		} else {
			result["final_status"] = task.Status
			result["task"] = task
			// Phase 5: on a completed A2A task, persist the artifact payload
			// into the global_contexts table so any agent can read it back
			// via list_global_contexts / resources/read. Tag format: a2a:<host>
			// so the UI can group / filter them.
			h.landA2AArtifact(agentURL, resp.TaskID, ctx, formatA2ATaskMarkdown(task))
		}
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return string(resultJSON), false
}

// landA2AArtifact persists an A2A delegation result into the
// `global_contexts` table so it is visible to other agents via the same
// flows the PM uses for human-authored playbooks.
//
// Behaviour:
//   - scope = 'project' when ctx.ProjectID is set; 'global' otherwise.
//     We default to 'global' on A2A artifacts because the most common
//     case is "external reviewer shared something everyone should read".
//   - Title is `<a2a-host> — <task-id prefix>` so the dashboard's
//     GlobalContextCard shows a recognisable heading.
//   - Tags always include `a2a:<host>`; failures / timeouts add
//     `a2a:timeout` or `a2a:failed` so PMs can filter.
//   - Errors are silently logged: an A2A artifact write failure must
//     never break the MCP response, because the calling agent still
//     needs the A2A task ID and status to act on.
func (h *MCPHandler) landA2AArtifact(agentURL, taskID string, ctx context.Context, body string) {
	if h.db == nil {
		return
	}
	// Fallback agent_id for the artifact row: the calling MCP agent, or
	// any PM if none is bound. We never want a NULL agent_id (FK requires
	// a real agent).
	agentID := ""
	if mcpc, ok := ctx.Value("mcpCtx").(MCPContext); ok {
		agentID = mcpc.AgentID
	}
	if agentID == "" {
		// Last resort: pick any PM agent so the FK passes.
		_ = h.db.QueryRow(`SELECT id FROM agents WHERE role = 'pm' ORDER BY created_at LIMIT 1`).Scan(&agentID)
	}
	if agentID == "" {
		slog.WarnContext(ctx, "landA2AArtifact: no agent available, skipping write", "task_id", taskID)
		return
	}

	host := agentURL
	if u, err := url.Parse(agentURL); err == nil && u.Host != "" {
		host = u.Host
	}
	title := fmt.Sprintf("%s — %s", host, shortPrefix(taskID, 8))
	tags := []string{"a2a:" + host, "a2a:" + shortPrefix(taskID, 8)}
	if strings.HasPrefix(body, "timeout:") {
		tags = append(tags, "a2a:timeout")
	} else if strings.HasPrefix(body, "failed:") {
		tags = append(tags, "a2a:failed")
	}

	_, err := h.db.Exec(`
		INSERT INTO global_contexts (id, scope, project_id, agent_id, title, content, tags)
		VALUES ($1, 'global', NULL, $2, $3, $4, $5)
	`, uuid.New().String(), agentID, title, body, pq.Array(tags))
	if err != nil {
		slog.ErrorContext(ctx, "landA2AArtifact: insert failed", "task_id", taskID, "error", err)
	}
}

// formatA2ATaskMarkdown renders a *a2aModels.Task as a markdown blob
// suitable for a global_contexts row. We keep the shape stable so
// downstream agents can parse it.
func formatA2ATaskMarkdown(t *a2aModels.Task) string {
	if t == nil {
		return ""
	}
	b, _ := json.MarshalIndent(t, "", "  ")
	return fmt.Sprintf("# A2A task\n\n**Status:** `%s`\n\n```json\n%s\n```\n", t.Status, string(b))
}

// shortPrefix returns the first n characters of s — used to keep the
// a2a-context title short.
func shortPrefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func (h *MCPHandler) executeGetA2ATaskStatus(args map[string]interface{}) (string, bool) {
	agentURL, ok := args["agent_url"].(string)
	if !ok || agentURL == "" {
		return `{"error": "agent_url is required"}`, true
	}

	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return `{"error": "task_id is required"}`, true
	}

	// Create A2A client and get task
	client := createA2AClient()
	task, err := client.GetTask(context.Background(), agentURL, taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to get task: %s"}`, err.Error()), true
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"success":   true,
		"agent_url": agentURL,
		"task":      task,
	}, "", "  ")
	return string(result), false
}

// Helper functions for A2A integration

func createA2AClient() *a2aClient.HTTPClient {
	return a2aClient.NewHTTPClient(a2aClient.WithTimeout(30 * time.Second))
}

func pollTaskUntilComplete(ctx context.Context, client *a2aClient.HTTPClient, agentURL, taskID string) (*a2aModels.Task, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task completion")
		case <-ticker.C:
			task, err := client.GetTask(ctx, agentURL, taskID)
			if err != nil {
				return nil, err
			}

			if task.Status.IsTerminal() {
				return task, nil
			}
		}
	}
}

// -------------------------------------------------------------------
// Mesh app additions — Phase 1/2/3/4 (agent-shaker #mesh-plan)
// -------------------------------------------------------------------

// helper: returns true when ctx.AgentID is bound to an agent whose role
// is 'pm'. Used to gate PM-only MCP tools. Returns false when the agent
// cannot be resolved (no DB / agent not found), which is the safest
// default — closed by default, open only when the role is explicit.
func (h *MCPHandler) isPMAgent(ctx MCPContext) bool {
	if h.db == nil || ctx.AgentID == "" {
		return false
	}
	var role string
	err := h.db.QueryRow(`SELECT role FROM agents WHERE id = $1`, ctx.AgentID).Scan(&role)
	if err != nil {
		return false
	}
	return role == string(models.RolePM)
}

// helper: returns a JSON-RPC -32003 error string for PM-only tools.
func pmForbidden(tool string) string {
	return fmt.Sprintf(`{"error": "Forbidden: tool %q is PM-only. Connect with an agent whose role='pm' or ask the project owner to grant your agent the PM role."}`, tool)
}

func (h *MCPHandler) executeRegisterSelf(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	projectID, _ := args["project_id"].(string)
	if projectID == "" {
		projectID = ctx.ProjectID
	}
	if projectID == "" {
		return `{"error": "project_id is required (pass as argument or set it in the MCP connection URL)"}`, true
	}

	name, ok := args["name"].(string)
	if !ok || name == "" {
		return `{"error": "name is required"}`, true
	}

	role, _ := args["role"].(string)
	if role == "" {
		role = "backend"
	}
	team, _ := args["team"].(string)
	repoURL, _ := args["repo_url"].(string)
	repoBranch, _ := args["repo_branch"].(string)
	if repoBranch == "" {
		repoBranch = "main"
	}

	id := uuid.New().String()
	var teamPtr *string
	if team != "" {
		teamPtr = &team
	}

	// Use a single transaction so an agent row + its optional repo row
	// either both succeed or both roll back.
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	rolledBack := false
	defer func() {
		if rolledBack {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.Exec(`
		INSERT INTO agents (id, project_id, name, role, team, status)
		VALUES ($1, $2, $3, $4, $5, 'idle')
		ON CONFLICT (project_id, name) DO UPDATE SET role = EXCLUDED.role, team = EXCLUDED.team
		RETURNING id
	`, id, projectID, name, role, teamPtr)
	if err != nil {
		rolledBack = true
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	// Re-read the agent id — ON CONFLICT may have rewritten our freshly
	// generated uuid.
	var finalAgentID string
	err = tx.QueryRow(`SELECT id FROM agents WHERE project_id = $1 AND name = $2`, projectID, name).Scan(&finalAgentID)
	if err != nil {
		rolledBack = true
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	var repoRow map[string]interface{}
	if repoURL != "" {
		repoID := uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO project_repos (id, project_id, url, branch, role, agent_id)
			VALUES ($1, $2, $3, $4, 'code', $5)
		`, repoID, projectID, repoURL, repoBranch, finalAgentID)
		if err != nil {
			rolledBack = true
			return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
		}
		repoRow = map[string]interface{}{
			"id":       repoID,
			"url":      repoURL,
			"branch":   repoBranch,
			"agent_id": finalAgentID,
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	resp := map[string]interface{}{
		"success":    true,
		"id":         finalAgentID,
		"project_id": projectID,
		"name":       name,
		"role":       role,
		"note":       "Send this id as ?agent_id= in your MCP connection URL on subsequent connections",
	}
	if team != "" {
		resp["team"] = team
	}
	if repoRow != nil {
		resp["repo"] = repoRow
	}
	out, _ := json.MarshalIndent(resp, "", "  ")
	return string(out), false
}

func (h *MCPHandler) executeCreateMilestone(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}
	if !h.isPMAgent(ctx) {
		return pmForbidden("create_milestone"), true
	}

	projectID, _ := args["project_id"].(string)
	if projectID == "" {
		projectID = ctx.ProjectID
	}
	if projectID == "" {
		return `{"error": "project_id is required"}`, true
	}
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return `{"error": "title is required"}`, true
	}
	description, _ := args["description"].(string)
	status, _ := args["status"].(string)
	if status == "" {
		status = "planned"
	}
	var targetDate *time.Time
	if td, ok := args["target_date"].(string); ok && td != "" {
		// Accept either YYYY-MM-DD or RFC3339.
		var t time.Time
		var err error
		if t, err = time.Parse("2006-01-02", td); err != nil {
			t, err = time.Parse(time.RFC3339, td)
		}
		if err != nil {
			return fmt.Sprintf(`{"error": "invalid target_date %q (use YYYY-MM-DD)"}`, td), true
		}
		targetDate = &t
	}

	id := uuid.New().String()
	createdBy := ctx.AgentID
	if createdBy == "" {
		// Last-resort fallback: use the first PM agent of the project.
		var pmID string
		err := h.db.QueryRow(`SELECT id FROM agents WHERE project_id = $1 AND role = 'pm' ORDER BY created_at LIMIT 1`, projectID).Scan(&pmID)
		if err != nil {
			return `{"error": "created_by is required (no agent_id in URL and no PM in project)"}`, true
		}
		createdBy = pmID
	}

	_, err := h.db.Exec(`
		INSERT INTO milestones (id, project_id, title, description, status, target_date, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, projectID, title, description, status, targetDate, createdBy)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	out, _ := json.MarshalIndent(map[string]interface{}{
		"success":     true,
		"id":          id,
		"project_id":  projectID,
		"title":       title,
		"description": description,
		"status":      status,
		"target_date": targetDate,
		"created_by":  createdBy,
		"note":        "Add tasks to this milestone with assign_task_to_milestone",
	}, "", "  ")
	return string(out), false
}

func (h *MCPHandler) executeListMilestones(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}
	projectID, _ := args["project_id"].(string)
	if projectID == "" {
		projectID = ctx.ProjectID
	}
	if projectID == "" {
		return `{"error": "project_id is required"}`, true
	}
	rows, err := h.db.Query(`
		SELECT id, title, description, status, target_date, created_by, created_at
		FROM milestones WHERE project_id = $1 ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id, title, status, createdBy string
		var description *string
		var targetDate *time.Time
		var createdAt interface{}
		if err := rows.Scan(&id, &title, &description, &status, &targetDate, &createdBy, &createdAt); err != nil {
			continue
		}
		entry := map[string]interface{}{
			"id":         id,
			"title":      title,
			"status":     status,
			"created_by": createdBy,
			"created_at": createdAt,
		}
		if description != nil {
			entry["description"] = *description
		}
		if targetDate != nil {
			entry["target_date"] = targetDate.Format("2006-01-02")
		}
		items = append(items, entry)
	}
	out, _ := json.MarshalIndent(map[string]interface{}{
		"milestones": items,
		"count":      len(items),
	}, "", "  ")
	return string(out), false
}

func (h *MCPHandler) executeAssignTaskToMilestone(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}
	if !h.isPMAgent(ctx) {
		return pmForbidden("assign_task_to_milestone"), true
	}
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return `{"error": "task_id is required"}`, true
	}
	milestoneID, ok := args["milestone_id"].(string)
	if !ok || milestoneID == "" {
		return `{"error": "milestone_id is required"}`, true
	}
	tag, err := h.db.Exec(`UPDATE tasks SET milestone_id = $1, updated_at = NOW() WHERE id = $2`, milestoneID, taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	rows, err := tag.RowsAffected()
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	if rows == 0 {
		return `{"error": "task not found"}`, true
	}
	out, _ := json.MarshalIndent(map[string]interface{}{
		"success":      true,
		"task_id":      taskID,
		"milestone_id": milestoneID,
	}, "", "  ")
	return string(out), false
}

func (h *MCPHandler) executePublishGlobalContext(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}
	if !h.isPMAgent(ctx) {
		return pmForbidden("publish_global_context"), true
	}

	scope, _ := args["scope"].(string)
	if scope != "global" && scope != "project" {
		return `{"error": "scope must be 'global' or 'project'"}`, true
	}
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return `{"error": "title is required"}`, true
	}
	content, _ := args["content"].(string)

	var tags []string
	if t, ok := args["tags"].([]interface{}); ok {
		for _, v := range t {
			if s, ok := v.(string); ok {
				tags = append(tags, s)
			}
		}
	}

	var projectIDPtr *string
	if scope == "project" {
		pid, _ := args["project_id"].(string)
		if pid == "" {
			pid = ctx.ProjectID
		}
		if pid == "" {
			return `{"error": "scope='project' requires project_id"}`, true
		}
		projectIDPtr = &pid
	}

	agentID := ctx.AgentID
	if agentID == "" {
		// Fallback to first PM agent of the project (or any agent if global).
		if projectIDPtr != nil {
			err := h.db.QueryRow(`SELECT id FROM agents WHERE project_id = $1 AND role = 'pm' ORDER BY created_at LIMIT 1`, *projectIDPtr).Scan(&agentID)
			if err != nil {
				return `{"error": "no agent_id in URL and no PM in project"}`, true
			}
		} else {
			err := h.db.QueryRow(`SELECT id FROM agents WHERE role = 'pm' ORDER BY created_at LIMIT 1`).Scan(&agentID)
			if err != nil {
				return `{"error": "no PM agent registered on this server"}`, true
			}
		}
	}

	id := uuid.New().String()
	_, err := h.db.Exec(`
		INSERT INTO global_contexts (id, scope, project_id, agent_id, title, content, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, scope, projectIDPtr, agentID, title, content, pq.Array(tags))
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			return fmt.Sprintf(`{"error": "a global context with title %q already exists; pick a different title"}`, title), true
		}
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	resp := map[string]interface{}{
		"success":  true,
		"id":       id,
		"scope":    scope,
		"title":    title,
		"tags":     tags,
		"agent_id": agentID,
	}
	if projectIDPtr != nil {
		resp["project_id"] = *projectIDPtr
	}
	if scope == "global" {
		resp["uri"] = "global://" + title
		resp["note"] = "any agent can now read this with read_resource('global://" + title + "')"
	}
	out, _ := json.MarshalIndent(resp, "", "  ")
	return string(out), false
}

func (h *MCPHandler) executeListGlobalContexts(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	scope, _ := args["scope"].(string)
	if scope != "" && scope != "global" && scope != "project" {
		return `{"error": "scope must be 'global', 'project', or empty"}`, true
	}

	var projectIDPtr *string
	if pid, _ := args["project_id"].(string); pid != "" {
		projectIDPtr = &pid
	} else if ctx.ProjectID != "" {
		pid := ctx.ProjectID
		projectIDPtr = &pid
	}
	tagPrefix, _ := args["tag_prefix"].(string)

	q := `SELECT id, scope, project_id, agent_id, title, tags, created_at, updated_at
	      FROM global_contexts WHERE 1=1`
	args2 := []interface{}{}
	if scope != "" {
		q += fmt.Sprintf(" AND scope = $%d", len(args2)+1)
		args2 = append(args2, scope)
	}
	if projectIDPtr != nil {
		q += fmt.Sprintf(" AND project_id = $%d", len(args2)+1)
		args2 = append(args2, *projectIDPtr)
	} else if scope == "" {
		// Listing all and no project filter: do not surface global rows
		// to non-PM callers. PMs see everything.
		if !h.isPMAgent(ctx) {
			q += " AND scope = 'project'"
		}
	}
	if tagPrefix != "" {
		q += fmt.Sprintf(" AND EXISTS (SELECT 1 FROM unnest(tags) t WHERE t LIKE $%d)", len(args2)+1)
		args2 = append(args2, tagPrefix+"%")
	}
	q += " ORDER BY updated_at DESC"

	rows, err := h.db.Query(q, args2...)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	defer rows.Close()

	var items []map[string]interface{}
	for rows.Next() {
		var id, scopeStr, agentID, title string
		var projectID *string
		var tagsRaw interface{}
		var createdAt, updatedAt interface{}
		if err := rows.Scan(&id, &scopeStr, &projectID, &agentID, &title, &tagsRaw, &createdAt, &updatedAt); err != nil {
			continue
		}
		entry := map[string]interface{}{
			"id":         id,
			"scope":      scopeStr,
			"agent_id":   agentID,
			"title":      title,
			"tags":       tagsRaw,
			"created_at": createdAt,
			"updated_at": updatedAt,
		}
		if projectID != nil {
			entry["project_id"] = *projectID
		}
		if scopeStr == "global" {
			entry["uri"] = "global://" + title
		}
		items = append(items, entry)
	}
	out, _ := json.MarshalIndent(map[string]interface{}{
		"contexts": items,
		"count":    len(items),
	}, "", "  ")
	return string(out), false
}

func (h *MCPHandler) executeReadGlobalContext(args map[string]interface{}) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}
	id, _ := args["id"].(string)
	title, _ := args["title"].(string)
	if id == "" && title == "" {
		return `{"error": "either id or title is required"}`, true
	}

	var (
		rowID, scopeStr, agentID, rowTitle, content string
		projectID                                   *string
		tagsRaw                                     interface{}
		createdAt, updatedAt                        interface{}
	)
	var err error
	if id != "" {
		err = h.db.QueryRow(`
			SELECT id, scope, project_id, agent_id, title, content, tags, created_at, updated_at
			FROM global_contexts WHERE id = $1
		`, id).Scan(&rowID, &scopeStr, &projectID, &agentID, &rowTitle, &content, &tagsRaw, &createdAt, &updatedAt)
	} else {
		err = h.db.QueryRow(`
			SELECT id, scope, project_id, agent_id, title, content, tags, created_at, updated_at
			FROM global_contexts WHERE scope = 'global' AND title = $1
		`, title).Scan(&rowID, &scopeStr, &projectID, &agentID, &rowTitle, &content, &tagsRaw, &createdAt, &updatedAt)
	}
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	resp := map[string]interface{}{
		"id":         rowID,
		"scope":      scopeStr,
		"agent_id":   agentID,
		"title":      rowTitle,
		"content":    content,
		"tags":       tagsRaw,
		"created_at": createdAt,
		"updated_at": updatedAt,
		"format":     "markdown",
	}
	if projectID != nil {
		resp["project_id"] = *projectID
	}
	if scopeStr == "global" {
		resp["uri"] = "global://" + rowTitle
	}
	out, _ := json.MarshalIndent(resp, "", "  ")
	return string(out), false
}

func (h *MCPHandler) executeReadGlobalContextByTitle(title string) (string, bool) {
	return h.executeReadGlobalContext(map[string]interface{}{"title": title})
}

// -------------------------------------------------------------------

func (h *MCPHandler) sendResponse(w http.ResponseWriter, id interface{}, result interface{}, rpcErr *JSONRPCError) {
	w.Header().Set("Content-Type", "application/json")

	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
	}

	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *MCPHandler) sendError(w http.ResponseWriter, id interface{}, code int, message string, data interface{}) {
	h.sendResponse(w, id, nil, &JSONRPCError{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// Context-aware tool implementations

func (h *MCPHandler) executeGetMyIdentity(ctx MCPContext) (string, bool) {
	identity := map[string]interface{}{
		"configured": ctx.ProjectID != "" || ctx.AgentID != "",
	}

	if ctx.ProjectID != "" {
		identity["project_id"] = ctx.ProjectID
		// Fetch project details
		if h.db != nil {
			var name, description, status string
			err := h.db.QueryRow("SELECT name, description, status FROM projects WHERE id = $1", ctx.ProjectID).
				Scan(&name, &description, &status)
			if err == nil {
				identity["project"] = map[string]string{
					"name":        name,
					"description": description,
					"status":      status,
				}
			}
		}
	}

	if ctx.AgentID != "" {
		identity["agent_id"] = ctx.AgentID
		// Fetch agent details
		if h.db != nil {
			var name, role, status string
			var projectID interface{}
			err := h.db.QueryRow("SELECT name, role, status, project_id FROM agents WHERE id = $1", ctx.AgentID).
				Scan(&name, &role, &status, &projectID)
			if err == nil {
				identity["agent"] = map[string]interface{}{
					"name":       name,
					"role":       role,
					"status":     status,
					"project_id": projectID,
				}
			}
		}
	}

	// `configured` is set on every path that builds `identity`, but the value
	// comes out of a map literal that is assembled in several branches above.
	// An unchecked assertion on it is a panic in the middle of a tool call, so
	// treat "missing or not a bool" the same as "not configured".
	if configured, ok := identity["configured"].(bool); !ok || !configured {
		identity["message"] = "No project_id or agent_id configured in MCP connection URL. Add ?project_id=UUID&agent_id=UUID to the URL."
	}

	result, _ := json.MarshalIndent(identity, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeGetMyProject(ctx MCPContext) (string, bool) {
	if ctx.ProjectID == "" {
		return `{"error": "No project_id configured in MCP connection URL. Add ?project_id=UUID to the URL."}`, true
	}

	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	var id, name, description, status string
	var createdAt, updatedAt interface{}
	err := h.db.QueryRow("SELECT id, name, description, status, created_at, updated_at FROM projects WHERE id = $1", ctx.ProjectID).
		Scan(&id, &name, &description, &status, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Sprintf(`{"error": "Project not found: %s"}`, err.Error()), true
	}

	// Get agents count
	agentCount := countRows(h.db, "SELECT COUNT(*) FROM agents WHERE project_id = $1", ctx.ProjectID)

	// Get tasks summary
	pendingTasks := countRows(h.db, "SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND status = 'pending'", ctx.ProjectID)
	inProgressTasks := countRows(h.db, "SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND status = 'in_progress'", ctx.ProjectID)
	doneTasks := countRows(h.db, "SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND status = 'done'", ctx.ProjectID)
	blockedTasks := countRows(h.db, "SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND status = 'blocked'", ctx.ProjectID)

	result, _ := json.MarshalIndent(map[string]interface{}{
		"id":          id,
		"name":        name,
		"description": description,
		"status":      status,
		"created_at":  createdAt,
		"updated_at":  updatedAt,
		"agents":      agentCount,
		"tasks": map[string]int{
			"pending":     pendingTasks,
			"in_progress": inProgressTasks,
			"done":        doneTasks,
			"blocked":     blockedTasks,
			"total":       pendingTasks + inProgressTasks + doneTasks + blockedTasks,
		},
	}, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeGetMyTasks(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if ctx.AgentID == "" {
		return `{"error": "No agent_id configured in MCP connection URL. Add ?agent_id=UUID to the URL."}`, true
	}

	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	query := "SELECT id, project_id, title, description, status, priority, assigned_to, created_at, updated_at FROM tasks WHERE assigned_to = $1"
	queryArgs := []interface{}{ctx.AgentID}

	if args != nil {
		if status, ok := args["status"].(string); ok && status != "" {
			query += " AND status = $2"
			queryArgs = append(queryArgs, status)
		}
	}
	query += " ORDER BY created_at DESC"

	rows, err := h.db.Query(query, queryArgs...)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}
	defer rows.Close()

	var tasks []map[string]interface{}
	for rows.Next() {
		var id, projectID, title, status, priority string
		var description, assignedTo interface{}
		var createdAt, updatedAt interface{}
		if err := rows.Scan(&id, &projectID, &title, &description, &status, &priority, &assignedTo, &createdAt, &updatedAt); err != nil {
			continue
		}
		tasks = append(tasks, map[string]interface{}{
			"id":          id,
			"project_id":  projectID,
			"title":       title,
			"description": description,
			"status":      status,
			"priority":    priority,
			"assigned_to": assignedTo,
			"created_at":  createdAt,
			"updated_at":  updatedAt,
		})
	}

	result, _ := json.MarshalIndent(map[string]interface{}{
		"agent_id": ctx.AgentID,
		"count":    len(tasks),
		"tasks":    tasks,
	}, "", "  ")
	return string(result), false
}

func (h *MCPHandler) executeUpdateMyStatus(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if ctx.AgentID == "" {
		return `{"error": "No agent_id configured in MCP connection URL. Add ?agent_id=UUID to the URL."}`, true
	}

	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	status, ok := args["status"].(string)
	if !ok {
		return `{"error": "status is required (idle, working, blocked, offline)"}`, true
	}

	validStatuses := map[string]bool{"idle": true, "working": true, "blocked": true, "offline": true}
	if !validStatuses[status] {
		return `{"error": "Invalid status. Must be one of: idle, working, blocked, offline"}`, true
	}

	query := "UPDATE agents SET status = $1, updated_at = NOW() WHERE id = $2"
	result, err := h.db.Exec(query, status, ctx.AgentID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return `{"error": "Agent not found"}`, true
	}

	// Retrieve updated agent information
	var agent models.Agent
	err = h.db.QueryRow(`
		SELECT id, project_id, name, role, team, status, last_seen, created_at
		FROM agents
		WHERE id = $1
	`, ctx.AgentID).Scan(&agent.ID, &agent.ProjectID, &agent.Name, &agent.Role, &agent.Team, &agent.Status, &agent.LastSeen, &agent.CreatedAt)
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to retrieve updated agent: %s"}`, err.Error()), true
	}

	// Broadcast agent update to project subscribers via WebSocket
	if h.hub != nil {
		h.hub.BroadcastToProject(agent.ProjectID, "agent_update", agent)
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success":  true,
		"agent_id": ctx.AgentID,
		"status":   status,
		"message":  "Agent status updated and broadcasted to project",
	}, "", "  ")
	return string(resultJSON), false
}

func (h *MCPHandler) executeClaimTask(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if ctx.AgentID == "" {
		return `{"error": "No agent_id configured in MCP connection URL. Add ?agent_id=UUID to the URL."}`, true
	}

	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	taskID, ok := args["task_id"].(string)
	if !ok {
		return `{"error": "task_id is required"}`, true
	}

	// Verify task exists and get its current assignment
	var currentAssignment interface{}
	var title string
	err := h.db.QueryRow("SELECT title, assigned_to FROM tasks WHERE id = $1", taskID).Scan(&title, &currentAssignment)
	if err != nil {
		return fmt.Sprintf(`{"error": "Task not found: %s"}`, err.Error()), true
	}

	// Update task assignment and set status to in_progress
	query := "UPDATE tasks SET assigned_to = $1, status = 'in_progress', updated_at = NOW() WHERE id = $2"
	_, err = h.db.Exec(query, ctx.AgentID, taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success":  true,
		"task_id":  taskID,
		"title":    title,
		"agent_id": ctx.AgentID,
		"status":   "in_progress",
		"message":  "Task claimed and status set to in_progress",
	}, "", "  ")
	return string(resultJSON), false
}

func (h *MCPHandler) executeCompleteTask(args map[string]interface{}, ctx MCPContext) (string, bool) {
	if ctx.AgentID == "" {
		return `{"error": "No agent_id configured in MCP connection URL. Add ?agent_id=UUID to the URL."}`, true
	}

	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	taskID, ok := args["task_id"].(string)
	if !ok {
		return `{"error": "task_id is required"}`, true
	}

	// Verify task is assigned to this agent
	var assignedTo interface{}
	var title string
	err := h.db.QueryRow("SELECT title, assigned_to FROM tasks WHERE id = $1", taskID).Scan(&title, &assignedTo)
	if err != nil {
		return fmt.Sprintf(`{"error": "Task not found: %s"}`, err.Error()), true
	}

	// Allow completion only if assigned to this agent (or unassigned)
	if assignedTo != nil && assignedTo != ctx.AgentID {
		assignedStr, _ := assignedTo.(string)
		if assignedStr != "" && assignedStr != ctx.AgentID {
			return fmt.Sprintf(`{"error": "Task is assigned to a different agent: %s"}`, assignedStr), true
		}
	}

	// Update task status to done
	query := "UPDATE tasks SET status = 'done', updated_at = NOW() WHERE id = $1"
	_, err = h.db.Exec(query, taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "%s"}`, err.Error()), true
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success":  true,
		"task_id":  taskID,
		"title":    title,
		"agent_id": ctx.AgentID,
		"status":   "done",
		"message":  "Task marked as completed",
	}, "", "  ")
	return string(resultJSON), false
}

func (h *MCPHandler) executeReassignTask(args map[string]interface{}) (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return `{"error": "task_id is required"}`, true
	}

	agentID, ok := args["agent_id"].(string)
	if !ok || agentID == "" {
		return `{"error": "agent_id is required"}`, true
	}

	// Verify the agent exists
	var agentName string
	err := h.db.QueryRow("SELECT name FROM agents WHERE id = $1", agentID).Scan(&agentName)
	if err != nil {
		return fmt.Sprintf(`{"error": "Agent not found: %s"}`, err.Error()), true
	}

	// Verify the task exists
	var taskTitle string
	err = h.db.QueryRow("SELECT title FROM tasks WHERE id = $1", taskID).Scan(&taskTitle)
	if err != nil {
		return fmt.Sprintf(`{"error": "Task not found: %s"}`, err.Error()), true
	}

	// Update the task's assigned_to field
	_, err = h.db.Exec("UPDATE tasks SET assigned_to = $1, updated_at = NOW() WHERE id = $2", agentID, taskID)
	if err != nil {
		return fmt.Sprintf(`{"error": "Failed to reassign task: %s"}`, err.Error()), true
	}

	resultJSON, _ := json.MarshalIndent(map[string]interface{}{
		"success":    true,
		"task_id":    taskID,
		"task_title": taskTitle,
		"agent_id":   agentID,
		"agent_name": agentName,
		"message":    fmt.Sprintf("Task '%s' reassigned to agent '%s'", taskTitle, agentName),
	}, "", "  ")
	return string(resultJSON), false
}
