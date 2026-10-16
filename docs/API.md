# API Reference

## Base URL

```
http://localhost:8080/api
```

## Authentication

Currently, no authentication is required. This is intended for internal development environments.

## Response Format

All API responses return JSON. Success responses include the requested data. Error responses include an error message.

## Endpoints

### Projects

#### POST /api/projects

Create a new project.

**Request Body:**
```json
{
  "name": "string (required)",
  "description": "string (optional)"
}
```

**Response:**
```json
{
  "id": "uuid",
  "name": "string",
  "description": "string",
  "status": "active",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

**Status Codes:**
- `200 OK` - Project created successfully
- `400 Bad Request` - Invalid request body
- `500 Internal Server Error` - Server error

---

#### GET /api/projects

List all projects.

**Response:**
```json
[
  {
    "id": "uuid",
    "name": "string",
    "description": "string",
    "status": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

---

#### GET /api/projects/{id}

Get a specific project.

**URL Parameters:**
- `id` (uuid) - Project ID

**Response:**
```json
{
  "id": "uuid",
  "name": "string",
  "description": "string",
  "status": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

**Status Codes:**
- `200 OK` - Project found
- `400 Bad Request` - Invalid project ID
- `404 Not Found` - Project not found

---

### Agents

#### POST /api/agents

Register a new agent.

**Request Body:**
```json
{
  "project_id": "uuid (required)",
  "name": "string (required)",
  "role": "string (optional)",
  "team": "string (optional)",
  "tool_type": "string (optional)"
}
```

`tool_type` is which client this agent runs in, and it decides which
configuration file is generated for it. It is optional and defaults to
`unknown`, so a client that predates the field keeps working. Accepted values:
`claude-code`, `codex`, `opencode`, `copilot-cli`, `copilot-vs`, `cursor`,
`windsurf`, `a2a`, `generic-mcp`, `unknown`. An unrecognised value is a 400
listing the accepted ones.

**Response:**
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "name": "string",
  "role": "string",
  "team": "string",
  "status": "active",
  "tool_type": "claude-code",
  "last_seen": "timestamp",
  "created_at": "timestamp"
}
```

---

#### GET /api/agents/{id}/config

Render the MCP configuration for one agent.

**URL Parameters:**
- `id` (uuid) - Agent ID

**Query Parameters:**
- `format` (string, optional) - `claude`, `codex`, `cursor` or `generic`.
  Omitted, the layout matching the agent's `tool_type` is rendered.

**Response:**
```json
{
  "agent": { "id": "uuid", "tool_type": "claude-code" },
  "format": "claude",
  "files": [
    {
      "format": "claude",
      "filename": ".claude/mcp.json",
      "target": "",
      "description": "Claude Code reads this from the project directory or ~/.claude",
      "content": "{ \"mcpServers\": { ... } }",
      "install": ""
    }
  ],
  "env_var": "AGENT_SHAKER_API_KEY",
  "endpoints": { "mcp": "https://shaker.example.com/mcp?project_id=...&agent_id=..." }
}
```

**No credential appears in the response.** Every rendered file references
`${env:AGENT_SHAKER_API_KEY}` instead, which is what makes a downloaded file safe
to commit. The plaintext lives only in the `POST .../connection` response below.

Requires the `read` scope. An unsupported `format` is a 400 whose message
enumerates the formats this server supports.

---

#### POST /api/agents/{id}/connection

Issue a credential bound to one agent, and render its configuration.

**URL Parameters:**
- `id` (uuid) - Agent ID

**Request Body:**
```json
{ "format": "claude" }
```

Both the body and the `format` field are optional.

**Response (201):**
```json
{
  "key": {
    "id": "uuid",
    "key_prefix": "ask_abcdefgh",
    "principal_id": "uuid",
    "agent_id": "uuid",
    "scopes": ["read", "write"],
    "secret": "ask_..."
  },
  "agent": { "id": "uuid" },
  "format": "claude",
  "files": [ { "filename": ".claude/mcp.json", "content": "..." } ]
}
```

`key.secret` is the **only** copy of the plaintext that will ever exist: the
server keeps a SHA-256 hash and cannot return the secret again. The response is
sent with `Cache-Control: no-store`.

Properties worth knowing before calling it:

- The credential is bound to this agent. On the MCP surface the binding is
  authoritative - a connection URL carrying a *different* `agent_id` is rejected
  with HTTP 403 rather than believed. See
  [AUTHENTICATION.md](./AUTHENTICATION.md).
- Scopes are fixed at `read` + `write`. The `admin` scope mints and revokes
  credentials, and is never granted to an agent - otherwise an agent could mint a
  key for any other agent and impersonate it, which is what the binding prevents.
- The key's project is taken from the agent row, not from the request.
- **Not idempotent.** One POST is one credential. A retry after a network timeout
  produces a second key; revoke the unused one with `DELETE /api/keys/{id}`.
- Requires the `write` scope.

---

#### GET /api/agents

List agents in a project.

**Query Parameters:**
- `project_id` (uuid, required) - Project ID

**Response:**
```json
[
  {
    "id": "uuid",
    "project_id": "uuid",
    "name": "string",
    "role": "string",
    "team": "string",
    "status": "string",
    "tool_type": "string",
    "last_seen": "timestamp",
    "created_at": "timestamp"
  }
]
```

---

#### PUT /api/agents/{id}/status

Update agent status.

**URL Parameters:**
- `id` (uuid) - Agent ID

**Request Body:**
```json
{
  "status": "string (required)" // "active", "idle", "offline"
}
```

**Response:**
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "name": "string",
  "role": "string",
  "team": "string",
  "status": "string",
  "last_seen": "timestamp",
  "created_at": "timestamp"
}
```

---

### Tasks

#### POST /api/tasks

Create a new task.

**Request Body:**
```json
{
  "project_id": "uuid (required)",
  "title": "string (required)",
  "description": "string (optional)",
  "priority": "string (optional)", // "low", "medium", "high"
  "created_by": "uuid (required)", // Agent ID
  "assigned_to": "uuid (optional)" // Agent ID
}
```

**Response:**
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "title": "string",
  "description": "string",
  "status": "pending",
  "priority": "string",
  "created_by": "uuid",
  "assigned_to": "uuid",
  "output": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### GET /api/tasks

List tasks in a project.

**Query Parameters:**
- `project_id` (uuid, required) - Project ID
- `status` (string, optional) - Filter by status
- `assigned_to` (uuid, optional) - Filter by assigned agent

**Response:**
```json
[
  {
    "id": "uuid",
    "project_id": "uuid",
    "title": "string",
    "description": "string",
    "status": "string",
    "priority": "string",
    "created_by": "uuid",
    "assigned_to": "uuid",
    "output": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

---

#### GET /api/tasks/{id}

Get a specific task.

**URL Parameters:**
- `id` (uuid) - Task ID

**Response:**
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "title": "string",
  "description": "string",
  "status": "string",
  "priority": "string",
  "created_by": "uuid",
  "assigned_to": "uuid",
  "output": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### PUT /api/tasks/{id}

Update a task.

**URL Parameters:**
- `id` (uuid) - Task ID

**Request Body:**
```json
{
  "status": "string (required)", // "pending", "in_progress", "blocked", "done", "cancelled"
  "output": "string (optional)"  // Result or notes
}
```

**Response:**
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "title": "string",
  "description": "string",
  "status": "string",
  "priority": "string",
  "created_by": "uuid",
  "assigned_to": "uuid",
  "output": "string",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

### Contexts (Documentation)

#### POST /api/contexts

Add documentation.

**Request Body:**
```json
{
  "project_id": "uuid (required)",
  "agent_id": "uuid (required)",
  "task_id": "uuid (optional)",
  "title": "string (required)",
  "content": "string (optional)", // Markdown format
  "tags": ["string"] // Array of tags
}
```

**Response:**
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "agent_id": "uuid",
  "task_id": "uuid",
  "title": "string",
  "content": "string",
  "tags": ["string"],
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

#### GET /api/contexts

List documentation in a project.

**Query Parameters:**
- `project_id` (uuid, required) - Project ID
- `tags` (string, optional) - Comma-separated tags to filter by

**Response:**
```json
[
  {
    "id": "uuid",
    "project_id": "uuid",
    "agent_id": "uuid",
    "task_id": "uuid",
    "title": "string",
    "content": "string",
    "tags": ["string"],
    "created_at": "timestamp",
    "updated_at": "timestamp"
  }
]
```

---

#### GET /api/contexts/{id}

Get specific documentation.

**URL Parameters:**
- `id` (uuid) - Context ID

**Response:**
```json
{
  "id": "uuid",
  "project_id": "uuid",
  "agent_id": "uuid",
  "task_id": "uuid",
  "title": "string",
  "content": "string",
  "tags": ["string"],
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

---

### WebSocket

#### WS /ws

Connect to real-time updates.

**Query Parameters:**
- `project_id` (uuid, required) - Project ID to subscribe to

**Message Format:**
```json
{
  "type": "string", // "task_update", "agent_update", "context_added"
  "payload": {}     // Entity data
}
```

**Example:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?project_id=UUID');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Type:', message.type);
  console.log('Payload:', message.payload);
};
```

---

### Health Check

#### GET /health

Check server health.

**Response:**
```
OK
```

**Status Code:** `200 OK`

---

## Status Values

### Task Status
- `pending` - Task is created and waiting to be started
- `in_progress` - Task is being worked on
- `blocked` - Task is blocked by a dependency
- `done` - Task is completed
- `cancelled` - Task was cancelled

### Task Priority
- `low` - Low priority
- `medium` - Medium priority (default)
- `high` - High priority

### Agent Status
- `active` - Agent is actively working
- `idle` - Agent is idle and waiting for tasks
- `offline` - Agent is offline

### Project Status
- `active` - Project is active (default)

## Error Handling

All error responses follow this format:

```json
{
  "error": "Error message description"
}
```

Common HTTP status codes:
- `400 Bad Request` - Invalid input or missing required fields
- `404 Not Found` - Resource not found
- `500 Internal Server Error` - Server-side error

## Rate Limiting

Currently, there are no rate limits. This may change in future versions.

## Versioning

The API is currently unversioned. Breaking changes will be communicated in advance.

---

## Mesh-app additions (Phase 1/2/3)

### Milestones

#### POST /api/milestones

Create a milestone.

Request body: { "project_id": "uuid", "title": "string", "description": "string?",
"status": "planned|active|done|dropped", "target_date": "YYYY-MM-DD?", "created_by": "uuid" }.

Response: 201 with the created Milestone. Status 409 (milestone_open_tasks) is
returned only on transitions to done if any linked task is not in a terminal state.

#### GET /api/milestones?project_id=...

List milestones for a project, newest first.

#### PUT /api/milestones/{id}/status

Body: { "status": "planned|active|done|dropped", "description": "string?" }.
done is rejected with 409 if linked tasks are still open.

#### DELETE /api/milestones/{id}

Tasks linked to the deleted milestone are unlinked via ON DELETE SET NULL.

### Project repositories

#### POST /api/project_repos
#### GET /api/project_repos?project_id=...
#### GET /api/project_repos/{id}
#### PUT /api/project_repos/{id}
#### DELETE /api/project_repos/{id}

Body for create/update: { "project_id": "uuid", "url": "string", "branch": "main?",
"role": "code|infra|docs|design?", "agent_id": "uuid?" }.

Deleting a repo does NOT delete its owning agent row.

### Global contexts (Phase 3)

#### POST /api/global_contexts

Body: { "scope": "global|project", "project_id": "uuid? (required when scope=project)",
"agent_id": "uuid", "title": "string", "content": "string", "tags": ["string"]? }.

Returns 201 on success. Returns 409 (global_context_conflict) when
(scope, title) uniqueness is violated.

#### GET /api/global_contexts?scope=global|project&project_id=...&tag_prefix=...

Filter by scope, project, and tag prefix. Non-PM callers see only their
project-scoped docs unless they explicitly filter by scope=project.

#### GET /api/global_contexts/{id}
#### PUT /api/global_contexts/{id}
#### DELETE /api/global_contexts/{id}

### Tasks — extended

POST /api/tasks and PUT /api/tasks/{id} now accept optional
milestone_id (UUID, FK to milestones.id) and 	ags (string[]).
Tasks tagged eature:<name> roll up under the Features view in the Nuxt UI.

### MCP tools added

| Name | Role | Purpose |
| --- | --- | --- |
| egister_self | any | Create / re-attach an agent row; optionally create its project_repos row |
| create_milestone | pm-only | Create a milestone for the connected project |
| list_milestones | any | List milestones for the connected project |
| ssign_task_to_milestone | pm-only | Link an existing task to a milestone |
| publish_global_context | pm-only | Create a global or project playbook |
| list_global_contexts | any | List server-wide or project docs |
| ead_global_context | any | Read one doc by id, or one global playbook by title |
| esources/read global://<title> | any | Read a global playbook as 	ext/markdown |

PM gating returns JSON-RPC error -32003 (Forbidden) when the caller is
not bound to a pm-role agent.