package mcp

import (
	"fmt"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// The tool registry.
//
// The advertised schemas and the dispatch that serves them live here, in one
// file, and a test holds them in step: every name in the list has an executor,
// and every executor has a schema. They were two lists in a file of three
// thousand lines, and a tool could be advertised without being callable — or
// callable without being advertised — with nothing noticing until an agent
// called it and got "unknown tool".
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

// toolExecutor runs one MCP tool.
//
// The arguments and the connection context are always passed, and a tool that
// needs neither ignores them. One signature for every tool is what lets the
// registry be a list rather than three shapes.
type toolExecutor func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool)

// toolDefinitions returns every tool this server advertises.
func toolDefinitions() []Tool {
	return []Tool{
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
						"description": "New agent status: " + models.JoinValues(models.AllAgentStatuses),
						"enum":        models.StringSlice(models.AllAgentStatuses),
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
					"idempotency_key": map[string]interface{}{
						"type": "string",
						"description": "Optional key that makes a retry safe. " +
							"Sending the same key again returns the first answer " +
							"instead of claiming the task a second time.",
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
					"idempotency_key": map[string]interface{}{
						"type": "string",
						"description": "Optional key that makes a retry safe. " +
							"Sending the same key again returns the first answer " +
							"instead of completing the task a second time.",
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
						"description": "New task status: " + models.JoinValues(models.AllTaskStatuses),
						"enum":        models.StringSlice(models.AllTaskStatuses),
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
						"type": "string",
						// "pm" is deliberately absent from this enum. It used to be
						// listed, which told a client the server would accept it and
						// then refused at runtime — the advertisement and the
						// behaviour disagreed, and the error arrived after the agent
						// had already committed to the call. PM is granted by a
						// human; see the guard in executeRegisterSelf.
						"description": "Agent role: backend | frontend (free-text accepted). NOT 'pm' — the PM role carries approval authority and is granted by a human, so a self-service request for it is refused.",
						"enum":        []string{"backend", "frontend"},
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
		// Coordination Plane tools
		{
			Name: "send_heartbeat",
			Description: "Report that you are alive and renew your work lease. Send it every 30 seconds while you are " +
				"working, and after every meaningful state change. An agent whose lease expires is marked offline " +
				"and its tasks are taken back, so a missed heartbeat ends your assignment. The agent reported is " +
				"always the one this MCP connection is bound to; you cannot report for another agent.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Your current presence: " + models.JoinValues(models.AllHeartbeatStatuses),
						"enum":        models.StringSlice(models.AllHeartbeatStatuses),
					},
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the task you are working on. Shown to humans as 'who is on what'.",
					},
					"progress_pct": map[string]interface{}{
						"type":        "integer",
						"description": "How far along the current task you are, 0-100.",
						"minimum":     0,
						"maximum":     100,
					},
					"eta": map[string]interface{}{
						"type":        "string",
						"description": "When you expect to finish, RFC3339 (e.g. 2026-01-31T15:04:05Z). Advisory only.",
					},
					"metadata": map[string]interface{}{
						"type":        "object",
						"description": "Anything else worth recording with this beat: branch name, current step, what you are waiting on.",
					},
				},
				Required: []string{"status"},
			},
		},
		{
			Name: "announce_capability",
			Description: "Tell the server what this client can do. Call it once, at setup, before the first " +
				"heartbeat: the server uses the answer to decide whether this agent can accept a pushed context, " +
				"author a compaction summary, or enforce a policy gate. Known capabilities: " +
				models.JoinValues(models.AllAgentCapabilities) + ".",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"capabilities": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "The capabilities this client supports: " + models.JoinValues(models.AllAgentCapabilities) + ".",
					},
					"host": map[string]interface{}{
						"type":        "string",
						"description": "The client this agent runs in, e.g. opencode.",
					},
					"host_version": map[string]interface{}{
						"type":        "string",
						"description": "The client's own version.",
					},
					"plugin_version": map[string]interface{}{
						"type":        "string",
						"description": "The version of the coordination plugin that carries these capabilities.",
					},
				},
				Required: []string{"capabilities"},
			},
		},
		{
			Name: "complete_task_with_handoff",
			Description: "Close a task you own and, in the same call, create the follow-up task it implies — linked by a " +
				"dependency edge and carrying your handoff context. Use this instead of complete_task whenever finishing " +
				"the work reveals the next step, so there is never a window where the work is done and nobody knows " +
				"what comes next. The successor starts blocked if something else is outstanding, and every other task " +
				"waiting on the completed one is unblocked.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the task to complete. You must be the assignee.",
					},
					"output": map[string]interface{}{
						"type":        "string",
						"description": "What you produced: the answer, the file changed, the decision taken.",
					},
					"successor_title": map[string]interface{}{
						"type":        "string",
						"description": "Title of the follow-up task this handoff creates.",
					},
					"successor_description": map[string]interface{}{
						"type":        "string",
						"description": "What the successor task has to achieve.",
					},
					"successor_assigned_to": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the agent that should pick up the successor. Leave unset to leave it unassigned.",
					},
					"handoff_context": map[string]interface{}{
						"type":        "string",
						"description": "What the next agent needs and cannot re-derive: the API shape, the branch, the gotcha, what you already tried. Required — a successor without it makes the next agent repeat your work.",
					},
				},
				Required: []string{"task_id", "successor_title", "handoff_context"},
			},
		},
		{
			Name: "add_task_dependency",
			Description: "Record that a task cannot start until another one is done, and block the waiting task until " +
				"then. Use it when you discover the ordering yourself — the blocked task is promoted to ready " +
				"automatically when its last predecessor completes. A cycle is refused.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the task that has to wait.",
					},
					"depends_on_task_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the task that must be done first.",
					},
				},
				Required: []string{"task_id", "depends_on_task_id"},
			},
		},
		{
			Name: "get_task_dependencies",
			Description: "List both directions of a task's dependency edges: what it waits for, and what it unblocks. " +
				"Use it before starting work to see whether a task is claimable, and before completing one to see what " +
				"your finish will release.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the task to inspect.",
					},
				},
				Required: []string{"task_id"},
			},
		},
		{
			Name: "raise_blocker",
			Description: "Report that you cannot proceed until a human decides or provides something, and block your " +
				"task and yourself. Use it for the work that will not move without a person: an approval, a design " +
				"decision, a missing credential, access to a system. Do not use it for ordinary dependencies between " +
				"tasks — use add_task_dependency for those. One blocker per distinct ask; several on one task are " +
				"legitimate, and none of them is replaced by another.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the task that is stuck.",
					},
					"blocker_type": map[string]interface{}{
						"type":        "string",
						"description": "What kind of help you need: " + models.JoinValues(models.AllBlockerTypes),
						"enum":        models.StringSlice(models.AllBlockerTypes),
					},
					"title": map[string]interface{}{
						"type":        "string",
						"description": "One line naming what you need, 500 characters or fewer.",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "What you tried, what you already ruled out, and what a good answer would let you do next.",
					},
				},
				Required: []string{"task_id", "blocker_type", "title"},
			},
		},
		{
			Name: "check_blocker_status",
			Description: "Check whether a blocker you raised has been answered, and read the answer. Use it instead " +
				"of re-raising the blocker when polling. Only the agent that raised it can read it.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"blocker_id": map[string]interface{}{
						"type":        "string",
						"description": "UUID of the blocker to check, as returned by raise_blocker.",
					},
				},
				Required: []string{"blocker_id"},
			},
		},
		{
			Name: "get_my_blockers",
			Description: "List the blockers you raised, newest first, optionally filtered by status. Use it after " +
				"reconnecting to catch up on answers you missed while offline.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"status": map[string]interface{}{
						"type":        "string",
						"description": "Optional blocker status filter: " + models.JoinValues(models.AllBlockerStatuses),
						"enum":        models.StringSlice(models.AllBlockerStatuses),
					},
				},
			},
		},
	}
}

// toolExecutors maps a tool name to its implementation.
//
// A map rather than a switch, so the set of callable tools is enumerable. A
// switch's case labels are not enumerable at runtime, and the test that keeps
// the advertised tools and the callable tools in step would have to parse Go
// source to do it.
var toolExecutors = map[string]toolExecutor{
	"get_my_identity": func(h *MCPHandler, _ map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeGetMyIdentity(ctx)
	},
	"get_my_project": func(h *MCPHandler, _ map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeGetMyProject(ctx)
	},
	"get_my_tasks": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeGetMyTasks(args, ctx)
	},
	"update_my_status": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeUpdateMyStatus(args, ctx)
	},
	"claim_task": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeClaimTask(args, ctx)
	},
	"complete_task": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeCompleteTask(args, ctx)
	},
	"reassign_task": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeReassignTask(args)
	},
	"list_projects": func(h *MCPHandler, _ map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeListProjects()
	},
	"get_project": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeGetProject(args)
	},
	"list_agents": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeListAgents(args)
	},
	"get_agent": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeGetAgent(args)
	},
	"list_tasks": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeListTasks(args)
	},
	"create_task": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeCreateTask(args, ctx)
	},
	"update_task_status": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeUpdateTaskStatus(args)
	},
	"list_contexts": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeListContexts(args)
	},
	"search_contexts": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeSearchContexts(args)
	},
	"add_context": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeAddContext(args, ctx)
	},
	"get_dashboard": func(h *MCPHandler, _ map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeGetDashboard()
	},
	// A2A Integration tools
	"discover_a2a_agent": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeDiscoverA2AAgent(args)
	},
	"delegate_to_a2a_agent": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeDelegateToA2AAgent(args)
	},
	"get_a2a_task_status": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeGetA2ATaskStatus(args)
	},
	"register_self": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeRegisterSelf(args, ctx)
	},
	"create_milestone": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeCreateMilestone(args, ctx)
	},
	"list_milestones": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeListMilestones(args, ctx)
	},
	"assign_task_to_milestone": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeAssignTaskToMilestone(args, ctx)
	},
	"publish_global_context": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executePublishGlobalContext(args, ctx)
	},
	"list_global_contexts": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeListGlobalContexts(args, ctx)
	},
	"read_global_context": func(h *MCPHandler, args map[string]interface{}, _ MCPContext) (string, bool) {
		return h.executeReadGlobalContext(args)
	},
	"send_heartbeat": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeSendHeartbeat(args, ctx)
	},
	"announce_capability": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeAnnounceCapability(args, ctx)
	},
	"complete_task_with_handoff": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeCompleteTaskWithHandoff(args, ctx)
	},
	"add_task_dependency": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeAddTaskDependency(args, ctx)
	},
	"get_task_dependencies": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeGetTaskDependencies(args, ctx)
	},
	"raise_blocker": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeRaiseBlocker(args, ctx)
	},
	"check_blocker_status": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeCheckBlockerStatus(args, ctx)
	},
	"get_my_blockers": func(h *MCPHandler, args map[string]interface{}, ctx MCPContext) (string, bool) {
		return h.executeGetMyBlockers(args, ctx)
	},
}

// dispatchTool runs one tool by name.
//
// An unknown name is a JSON-RPC error rather than a tool error: the client
// asked for something this server does not have.
func (h *MCPHandler) dispatchTool(name string, args map[string]interface{}, ctx MCPContext) (string, bool, *JSONRPCError) {
	executor, ok := toolExecutors[name]
	if !ok {
		return "", false, &JSONRPCError{
			Code:    -32601,
			Message: "Unknown tool",
			Data:    fmt.Sprintf("Tool not found: %s", name),
		}
	}

	resultText, isError := executor(h, args, ctx)
	return resultText, isError, nil
}
