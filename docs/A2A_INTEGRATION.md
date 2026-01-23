# A2A Protocol Integration Guide

## Overview

Agent Shaker now supports the **Agent-to-Agent (A2A) Protocol**, enabling seamless communication between AI agents. This implementation allows Agent Shaker to:

- **Be discovered** by other A2A-compatible agents via the standard agent card endpoint
- **Receive and execute tasks** from external agents
- **Stream real-time updates** via Server-Sent Events (SSE)
- **Share contexts as artifacts** for cross-agent knowledge sharing
- **Communicate with external A2A agents** as a client
- **Integrate A2A capabilities** with existing MCP tools

## Quick Start

### Discovering Agent Shaker

Fetch the agent card to discover Agent Shaker's capabilities:

```bash
curl http://localhost:8080/.well-known/agent-card.json
```

Response:
```json
{
  "name": "Agent Shaker",
  "description": "MCP-compatible context management server with A2A support",
  "version": "1.0.0",
  "capabilities": [
    {"type": "task", "description": "Asynchronous task execution"},
    {"type": "streaming", "description": "Real-time updates via SSE"},
    {"type": "artifacts", "description": "Markdown context sharing"},
    {"type": "mcp", "description": "MCP protocol support"}
  ],
  "endpoints": [
    {"path": "/a2a/v1/message", "method": "POST", "protocol": "A2A"},
    {"path": "/a2a/v1/tasks", "method": "GET", "protocol": "A2A"},
    ...
  ]
}
```

### Sending a Task

Submit a task to Agent Shaker:

```bash
curl -X POST http://localhost:8080/a2a/v1/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "content": "Analyze this document and provide a summary",
      "format": "text"
    },
    "metadata": {
      "priority": "high"
    }
  }'
```

Response:
```json
{
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "created_at": "2026-01-23T10:00:00Z"
}
```

### Checking Task Status

Get the status and result of a task:

```bash
curl http://localhost:8080/a2a/v1/tasks/550e8400-e29b-41d4-a716-446655440000
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "message": {"content": "Analyze this document..."},
  "result": {
    "content": "Analysis complete",
    "format": "text"
  },
  "created_at": "2026-01-23T10:00:00Z",
  "completed_at": "2026-01-23T10:00:05Z"
}
```

### Streaming Task Updates

For long-running tasks, use SSE streaming:

```bash
curl -X POST http://localhost:8080/a2a/v1/message:stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "message": {"content": "Process this data"}
  }'
```

SSE Response:
```
event: task_created
data: {"task_id": "...", "status": "pending"}

event: status
data: {"task_id": "...", "status": "running"}

event: completed
data: {"task_id": "...", "status": "completed", "result": {...}}
```

## API Reference

### Agent Discovery

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/.well-known/agent-card.json` | GET | Get agent capabilities and metadata |

### Task Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/a2a/v1/message` | POST | Create a new task |
| `/a2a/v1/message:stream` | POST | Create task with SSE streaming |
| `/a2a/v1/tasks` | GET | List all tasks (supports filtering) |
| `/a2a/v1/tasks/{taskId}` | GET | Get specific task details |
| `/a2a/v1/tasks/{taskId}` | DELETE | Cancel a task |

### Artifacts

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/a2a/v1/artifacts` | GET | List all artifacts (contexts) |
| `/a2a/v1/artifacts/{artifactId}` | GET | Get specific artifact |

### Query Parameters

For `GET /a2a/v1/tasks`:
- `status` - Filter by status: `pending`, `running`, `completed`, `failed`
- `limit` - Maximum number of results (default: 100)
- `offset` - Pagination offset

## MCP Integration

Agent Shaker includes MCP tools for interacting with external A2A agents:

### discover_a2a_agent

Discover an external A2A agent:

```json
{
  "name": "discover_a2a_agent",
  "arguments": {
    "agent_url": "https://external-agent.example.com"
  }
}
```

### delegate_to_a2a_agent

Delegate a task to an external agent:

```json
{
  "name": "delegate_to_a2a_agent",
  "arguments": {
    "agent_url": "https://external-agent.example.com",
    "message": "Perform this analysis",
    "wait_for_completion": true,
    "timeout_seconds": 120
  }
}
```

### get_a2a_task_status

Check the status of a delegated task:

```json
{
  "name": "get_a2a_task_status",
  "arguments": {
    "agent_url": "https://external-agent.example.com",
    "task_id": "550e8400-e29b-41d4-a716-446655440000"
  }
}
```

## Data Models

### AgentCard

The agent card describes the agent's capabilities and available endpoints. Agent Shaker supports **both standard and alternative capability formats** for maximum compatibility:

**Standard Format (Array):**
```json
{
  "name": "Agent Shaker",
  "version": "1.0.0",
  "capabilities": [
    {"type": "task", "description": "Asynchronous task execution"},
    {"type": "streaming", "description": "Real-time updates via SSE"}
  ],
  "endpoints": [...]
}
```

**Alternative Format (Object) - Also Supported:**
```json
{
  "name": "Some Agent",
  "version": "1.0.0",
  "capabilities": {
    "task": "Asynchronous task execution",
    "streaming": "Real-time updates via SSE"
  },
  "endpoints": [...]
}
```

> **Note:** Agent Shaker automatically converts the object format to the standard array format during discovery. This ensures compatibility with various A2A implementations that may use different formats.

### SendMessageRequest

```json
{
  "message": {
    "content": "string (required)",
    "context": {"key": "value"},
    "format": "text | markdown"
  },
  "metadata": {
    "priority": "low | medium | high",
    "timeout": 60,
    "callback_url": "https://..."
  }
}
```

### Task

```json
{
  "id": "uuid",
  "status": "pending | running | completed | failed",
  "message": {...},
  "result": {
    "content": "string",
    "format": "text | markdown",
    "data": {...}
  },
  "artifacts": [...],
  "created_at": "ISO8601",
  "updated_at": "ISO8601",
  "completed_at": "ISO8601 | null"
}
```

### Artifact

```json
{
  "id": "uuid",
  "name": "string",
  "type": "markdown | json | binary",
  "content_type": "text/markdown",
  "content": "string",
  "url": "string",
  "size": 1024,
  "created_at": "ISO8601",
  "metadata": {...}
}
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Agent Shaker Platform                    │
│                                                             │
│  ┌──────────────┐       ┌──────────────┐                   │
│  │ MCP Server   │       │ A2A Server   │                   │
│  │ (Existing)   │       │ (New)        │                   │
│  └──────────────┘       └──────────────┘                   │
│         │                       │                           │
│         └───────┬───────────────┘                           │
│                 │                                           │
│         ┌───────▼────────┐                                 │
│         │  Task Manager  │                                 │
│         │  (Unified)     │                                 │
│         └───────┬────────┘                                 │
│                 │                                           │
│    ┌────────────┼────────────┐                             │
│    │            │            │                             │
│ ┌──▼───┐   ┌───▼────┐  ┌───▼──────┐                       │
│ │Context│   │WebSocket│ │A2A Client│                       │
│ │Storage│   │  Hub    │ │(Outbound)│                       │
│ └───────┘   └────────┘  └──────────┘                       │
└─────────────────────────────────────────────────────────────┘
```

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `BASE_URL` | Public base URL for artifact URLs | `http://localhost:8080` |
| `TASKS_DIR` | Directory for task persistence | `./data/tasks` |
| `DATABASE_URL` | PostgreSQL connection string | (see docs) |

## Code Examples

### Go Client Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/techbuzzz/agent-shaker/internal/a2a/client"
    "github.com/techbuzzz/agent-shaker/internal/a2a/models"
)

func main() {
    // Create client
    c := client.NewHTTPClient()
    
    // Discover agent
    card, err := c.Discover(context.Background(), "http://localhost:8080")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Agent: %s v%s\n", card.Name, card.Version)
    
    // Send task
    req := &models.SendMessageRequest{
        Message: models.Message{
            Content: "Hello, Agent!",
            Format:  "text",
        },
    }
    
    resp, err := c.SendMessage(context.Background(), "http://localhost:8080", req)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Task ID: %s\n", resp.TaskID)
}
```

### JavaScript/TypeScript Client Example

```typescript
// Discover agent
const response = await fetch('http://localhost:8080/.well-known/agent-card.json');
const agentCard = await response.json();
console.log(`Agent: ${agentCard.name} v${agentCard.version}`);

// Send task
const taskResponse = await fetch('http://localhost:8080/a2a/v1/message', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    message: { content: 'Hello, Agent!', format: 'text' }
  })
});
const { task_id, status } = await taskResponse.json();

// Stream updates
const eventSource = new EventSource(
  `http://localhost:8080/a2a/v1/message:stream`
);
eventSource.onmessage = (event) => {
  console.log('Update:', JSON.parse(event.data));
};
```

### Using with curl

```bash
# Full workflow example

# 1. Discover agent
curl -s http://localhost:8080/.well-known/agent-card.json | jq .

# 2. Create task
TASK_ID=$(curl -s -X POST http://localhost:8080/a2a/v1/message \
  -H "Content-Type: application/json" \
  -d '{"message":{"content":"Hello"}}' | jq -r .task_id)

# 3. Wait and check status
sleep 1
curl -s http://localhost:8080/a2a/v1/tasks/$TASK_ID | jq .

# 4. List all completed tasks
curl -s "http://localhost:8080/a2a/v1/tasks?status=completed" | jq .

# 5. List artifacts
curl -s http://localhost:8080/a2a/v1/artifacts | jq .
```

## Troubleshooting

### Common Issues

**Task stuck in pending state**
- Check server logs for execution errors
- Verify database connectivity for artifact storage
- Ensure no timeout issues with external dependencies

**SSE connection drops**
- Check for proxy/load balancer timeout settings
- Disable buffering: `X-Accel-Buffering: no` header is set
- Verify keepalive messages are being sent

**Agent discovery fails with "cannot unmarshal object into Go struct field AgentCard.capabilities"**
- This error indicates the external agent is using an alternative format for capabilities
- Agent Shaker now supports both formats:
  - Standard array format: `"capabilities": [{"type": "task", "description": "..."}]`
  - Object format: `"capabilities": {"task": "...", "streaming": "..."}`
- The error should be resolved in version 1.0.0+ automatically
- If the issue persists, check the agent card response format:
  ```bash
  curl https://external-agent/.well-known/agent-card.json | jq .capabilities
  ```

**Agent discovery fails (general)**
- Verify the target URL is correct
- Check network connectivity
- Ensure the agent exposes `/.well-known/agent-card.json`
- Verify the agent card has required fields: `name`, `version`

**MCP tools not showing A2A options**
- Restart the MCP client connection
- Verify server is running with latest code
- Check `tools/list` response includes A2A tools

### Debug Logging

Enable verbose logging by checking server output:

```
2026/01/23 10:00:00 Task abc123 completed with status completed
2026/01/23 10:00:01 MCP Tool Call: delegate_to_a2a_agent with args {...}
```

## Security Considerations

- A2A endpoints support CORS for browser-based clients
- No authentication is currently required (add authentication middleware as needed)
- Consider using HTTPS in production
- Validate and sanitize all incoming message content

## Related Documentation

- [MCP Protocol Documentation](./MCP_QUICKSTART.md)
- [WebSocket Integration](./WEBSOCKET_FIX.md)
- [API Reference](./API.md)
- [Architecture Overview](./ARCHITECTURE.md)

## A2A Protocol Specification

For the full A2A Protocol specification, visit: https://a2a-protocol.org/
