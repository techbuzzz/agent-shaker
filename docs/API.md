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
  "team": "string (optional)"
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
  "status": "active",
  "last_seen": "timestamp",
  "created_at": "timestamp"
}
```

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
