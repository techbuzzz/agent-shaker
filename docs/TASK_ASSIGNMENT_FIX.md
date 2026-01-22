# Task Assignment Fix

## Issue
When an agent called `create_task`, the agent ID was detected and logged (e.g., "Assigned to: c72ea0e5-13e1-4a9b-a557-ce7a8bbdfdf0"), but the task was not actually assigned to that agent in the database. The `assigned_to` field remained NULL.

## Root Cause
The `executeCreateTask` function had two problems:

1. **Missing context-aware assignment logic**: The function used `ctx.AgentID` for `created_by` when not provided, but didn't do the same for `assigned_to`. This meant that when an agent created a task, it would set themselves as the creator but not automatically assign the task to themselves.

2. **Response didn't include assigned_to**: The success response JSON didn't include the `assigned_to` field, making it difficult to verify whether the assignment worked.

## Solution

### 1. Added Context-Aware Assignment Logic

```go
// Use agent_id from context if assigned_to not provided (agent assigns task to themselves)
if assignedTo == "" && ctx.AgentID != "" {
	assignedTo = ctx.AgentID
}
```

This ensures that when an agent creates a task through the MCP connection (with `agent_id` in the URL), the task is automatically assigned to that agent unless explicitly specified otherwise.

### 2. Enhanced Response to Include assigned_to

```go
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
```

The response now includes `assigned_to` when a task is assigned, making it clear who the task is assigned to.

## Behavior

### When Connected with Agent Context

If you connect to the MCP server with:
```
http://localhost:8080?project_id=PROJECT_UUID&agent_id=AGENT_UUID
```

And create a task:
```json
{
  "method": "tools/call",
  "params": {
    "name": "create_task",
    "arguments": {
      "title": "Fix login bug"
    }
  }
}
```

The task will be:
- **Created by**: AGENT_UUID (from URL context)
- **Assigned to**: AGENT_UUID (from URL context - agent assigns to themselves)

### Explicit Assignment Override

You can still explicitly assign to a different agent:
```json
{
  "method": "tools/call",
  "params": {
    "name": "create_task",
    "arguments": {
      "title": "Fix login bug",
      "assigned_to": "DIFFERENT_AGENT_UUID"
    }
  }
}
```

This will:
- **Created by**: Your agent ID (from URL context)
- **Assigned to**: DIFFERENT_AGENT_UUID (from arguments - explicit override)

## Priority Chain

For `assigned_to` field:
1. **Explicit argument** - If provided in `assigned_to`, use it
2. **URL context** - If not provided and `agent_id` in URL, assign to that agent (self-assignment)
3. **Unassigned** - If neither provided, task remains unassigned (NULL)

For `created_by` field:
1. **Explicit argument** - If provided in `created_by`, use it
2. **URL context** - If not provided and `agent_id` in URL, use that agent
3. **First agent fallback** - Query the first agent in the project

## Testing

### Test 1: Self-Assignment via Context
```bash
curl "http://localhost:8080?project_id=PROJECT_ID&agent_id=AGENT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "create_task",
      "arguments": {
        "title": "Test task"
      }
    }
  }'
```

**Expected response:**
```json
{
  "success": true,
  "id": "NEW_TASK_UUID",
  "title": "Test task",
  "status": "pending",
  "priority": "medium",
  "created_by": "AGENT_ID",
  "assigned_to": "AGENT_ID",
  "created_at": "2026-01-22T..."
}
```

### Test 2: Assign to Different Agent
```bash
curl "http://localhost:8080?project_id=PROJECT_ID&agent_id=AGENT_1_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "create_task",
      "arguments": {
        "title": "Task for agent 2",
        "assigned_to": "AGENT_2_ID"
      }
    }
  }'
```

**Expected response:**
```json
{
  "success": true,
  "id": "NEW_TASK_UUID",
  "title": "Task for agent 2",
  "status": "pending",
  "priority": "medium",
  "created_by": "AGENT_1_ID",
  "assigned_to": "AGENT_2_ID",
  "created_at": "2026-01-22T..."
}
```

### Test 3: Unassigned Task
```bash
curl "http://localhost:8080?project_id=PROJECT_ID&agent_id=AGENT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "create_task",
      "arguments": {
        "title": "Unassigned task",
        "assigned_to": ""
      }
    }
  }'
```

**Expected response:**
```json
{
  "success": true,
  "id": "NEW_TASK_UUID",
  "title": "Unassigned task",
  "status": "pending",
  "priority": "medium",
  "created_by": "AGENT_ID",
  "created_at": "2026-01-22T..."
}
```

Note: `assigned_to` is not in the response (task is unassigned).

## Database Schema

The `tasks` table structure:
```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id),
    title TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    priority TEXT DEFAULT 'medium',
    created_by UUID NOT NULL REFERENCES agents(id),  -- Required: who created it
    assigned_to UUID REFERENCES agents(id),          -- Optional: who it's assigned to
    output TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

## Benefits

1. **Natural workflow**: When an agent creates a task, it's automatically assigned to them (self-assignment is common)
2. **Explicit control**: Can still assign to other agents when needed
3. **Clear responses**: The response now shows both who created and who it's assigned to
4. **Context-aware**: Leverages the MCP connection context for convenience
5. **Backward compatible**: Explicit arguments still work as before

## Files Modified

- `internal/mcp/handler.go` - `executeCreateTask()` function
  - Added context-aware logic for `assigned_to`
  - Enhanced response to include `assigned_to` field

## Related Documentation

- [MCP Context-Aware Endpoints](./MCP_CONTEXT_AWARE_ENDPOINTS.md) - Overview of context-aware functionality
- [MCP Endpoints Fix](./MCP_ENDPOINTS_FIX.md) - Previous database constraint fixes
