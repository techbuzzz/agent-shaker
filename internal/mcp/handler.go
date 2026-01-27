package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	a2aClient "github.com/techbuzzz/agent-shaker/internal/a2a/client"
	a2aModels "github.com/techbuzzz/agent-shaker/internal/a2a/models"
	"github.com/techbuzzz/agent-shaker/internal/database"
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
}

func NewMCPHandler(db *database.DB, hub *websocket.Hub) *MCPHandler {
	return &MCPHandler{
		db:  db,
		hub: hub,
	}
}

// extractContext extracts project_id and agent_id from URL params or headers
func (h *MCPHandler) extractContext(r *http.Request) MCPContext {
	ctx := MCPContext{}

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

	// Extract context from URL/headers
	ctx := h.extractContext(r)
	if ctx.ProjectID != "" || ctx.AgentID != "" {
		log.Printf("MCP Context: project_id=%s, agent_id=%s", ctx.ProjectID, ctx.AgentID)
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
		"protocolVersion": "2024-11-05",
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

	log.Printf("MCP SSE connection established: %s (project=%s, agent=%s)", sessionID, ctx.ProjectID, ctx.AgentID)

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
			log.Printf("MCP SSE connection closed: %s", sessionID)
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

	log.Printf("MCP Request: method=%s, id=%v, project=%s, agent=%s", req.Method, req.ID, ctx.ProjectID, ctx.AgentID)

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
		json.Unmarshal(params, &clientParams)
	}

	log.Printf("MCP Initialize - Client: %v, Protocol: %s, Project: %s, Agent: %s",
		clientParams.ClientInfo, clientParams.ProtocolVersion, ctx.ProjectID, ctx.AgentID)

	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
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
			Description: "List all documentation and contexts shared by agents in the project. Content is in markdown format for easy reading.",
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

	log.Printf("MCP Tool Call: %s with args %v (project=%s, agent=%s)", callParams.Name, callParams.Arguments, ctx.ProjectID, ctx.AgentID)

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
	default:
		return nil, &JSONRPCError{
			Code:    -32601,
			Message: "Unknown tool",
			Data:    fmt.Sprintf("Tool not found: %s", callParams.Name),
		}
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

	switch readParams.URI {
	case "agent-shaker://projects":
		content, isError = h.executeListProjects()
	case "agent-shaker://agents":
		content, isError = h.executeListAgents(nil)
	case "agent-shaker://tasks":
		content, isError = h.executeListTasks(nil)
	case "agent-shaker://dashboard":
		content, isError = h.executeGetDashboard()
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
				MimeType: "application/json",
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

	query := `SELECT id, project_id, title, description, status, priority, assigned_to, created_at FROM tasks WHERE 1=1`
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
			argNum++
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
		var description, assignedTo *string
		var createdAt interface{}
		if err := rows.Scan(&id, &projectID, &title, &description, &status, &priority, &assignedTo, &createdAt); err != nil {
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
	query := `INSERT INTO tasks (id, project_id, title, description, status, priority, created_by, assigned_to) 
	          VALUES ($1, $2, $3, $4, 'pending', $5, $6, $7) RETURNING id, created_at`

	var createdID string
	var createdAt interface{}
	var assignedToPtr *string
	if assignedTo != "" {
		assignedToPtr = &assignedTo
	}

	err := h.db.QueryRow(query, id, projectID, title, description, priority, createdBy, assignedToPtr).Scan(&createdID, &createdAt)
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

	// Get agent name for better feedback
	var agentName string
	h.db.QueryRow(`SELECT name FROM agents WHERE id = $1`, agentID).Scan(&agentName)
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

func (h *MCPHandler) executeGetDashboard() (string, bool) {
	if h.db == nil {
		return `{"error": "Database not connected"}`, true
	}

	var projectCount, agentCount, taskCount, contextCount int
	var pendingTasks, inProgressTasks, doneTasks, blockedTasks int

	h.db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&projectCount)
	h.db.QueryRow("SELECT COUNT(*) FROM agents").Scan(&agentCount)
	h.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&taskCount)
	h.db.QueryRow("SELECT COUNT(*) FROM contexts").Scan(&contextCount)

	h.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'pending'").Scan(&pendingTasks)
	h.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'in_progress'").Scan(&inProgressTasks)
	h.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'done'").Scan(&doneTasks)
	h.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE status = 'blocked'").Scan(&blockedTasks)

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
		} else {
			result["final_status"] = task.Status
			result["task"] = task
		}
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return string(resultJSON), false
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

			if task.Status == a2aModels.TaskStatusCompleted || task.Status == a2aModels.TaskStatusFailed {
				return task, nil
			}
		}
	}
}

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

	if !identity["configured"].(bool) {
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
	var agentCount int
	h.db.QueryRow("SELECT COUNT(*) FROM agents WHERE project_id = $1", ctx.ProjectID).Scan(&agentCount)

	// Get tasks summary
	var pendingTasks, inProgressTasks, doneTasks, blockedTasks int
	h.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND status = 'pending'", ctx.ProjectID).Scan(&pendingTasks)
	h.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND status = 'in_progress'", ctx.ProjectID).Scan(&inProgressTasks)
	h.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND status = 'done'", ctx.ProjectID).Scan(&doneTasks)
	h.db.QueryRow("SELECT COUNT(*) FROM tasks WHERE project_id = $1 AND status = 'blocked'", ctx.ProjectID).Scan(&blockedTasks)

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
