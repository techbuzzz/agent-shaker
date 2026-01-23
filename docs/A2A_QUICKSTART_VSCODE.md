# Quick Start: Using A2A Protocol in VS Code - 5 Minutes

This guide shows you how to use Agent Shaker's A2A Protocol features in VS Code with GitHub Copilot in just 5 minutes.

## Prerequisites

- VS Code with GitHub Copilot installed
- Agent Shaker running locally: `go run cmd/server/main.go`
- PowerShell or terminal access

## Step 1: Verify Agent Shaker is A2A-Ready (30 seconds)

Open PowerShell and test the agent card endpoint:

```powershell
# Check if Agent Shaker is running and A2A-enabled
Invoke-RestMethod http://localhost:8080/.well-known/agent-card.json | ConvertTo-Json -Depth 10
```

**Expected output:**
```json
{
  "name": "Agent Shaker",
  "version": "1.0.0",
  "capabilities": [
    {
      "type": "task",
      "description": "Asynchronous task execution and management"
    },
    {
      "type": "streaming",
      "description": "Real-time task updates via Server-Sent Events (SSE)"
    },
    {
      "type": "artifacts",
      "description": "Markdown context sharing as A2A artifacts"
    }
  ],
  "endpoints": [...]
}
```

✅ If you see this, Agent Shaker is A2A-ready!

## Step 2: Create a Test Project in VS Code (1 minute)

Open VS Code, then open Copilot Chat and type:

```
@workspace I'm using the agent-shaker MCP server. Create a new project called "A2A Demo" for testing A2A protocol features.
```

**Copilot will:**
1. Connect to Agent Shaker via MCP
2. Create the project using the `create_project` tool
3. Return the project ID

**Example Copilot response:**
> "I've created the 'A2A Demo' project. Project ID: `550e8400-e29b-41d4-a716-446655440000`. You can now register agents and start using A2A features."

## Step 3: Simulate an External A2A Agent (1 minute)

For testing, we'll simulate an external A2A agent by sending a task directly to Agent Shaker. In PowerShell:

```powershell
# Send a task to Agent Shaker via A2A protocol
$body = @{
    message = @{
        content = "Analyze the authentication patterns in this codebase"
        format = "text"
    }
    metadata = @{
        priority = "high"
    }
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri http://localhost:8080/a2a/v1/message `
    -Method Post `
    -ContentType "application/json" `
    -Body $body

# Display the task ID
Write-Host "Task created with ID: $($response.task_id)"
Write-Host "Status: $($response.status)"
$taskId = $response.task_id
```

**Expected output:**
```
Task created with ID: 8bc63d36-b2be-4ee8-8c8c-931448246fd8
Status: pending
```

## Step 4: Check Task Status via A2A (30 seconds)

```powershell
# Check the task status
$task = Invoke-RestMethod "http://localhost:8080/a2a/v1/tasks/$taskId"
$task | ConvertTo-Json -Depth 10
```

**Expected output:**
```json
{
  "id": "8bc63d36-b2be-4ee8-8c8c-931448246fd8",
  "status": "completed",
  "message": {
    "content": "Analyze the authentication patterns in this codebase",
    "format": "text"
  },
  "result": {
    "content": "Task received: Analyze the authentication patterns...",
    "format": "text",
    "data": {
      "original_message": "Analyze the authentication patterns...",
      "processed_at": "2026-01-23T10:30:00Z"
    }
  },
  "created_at": "2026-01-23T10:29:55Z",
  "completed_at": "2026-01-23T10:29:56Z"
}
```

## Step 5: Use MCP Tools to Discover External Agents (1 minute)

Back in VS Code Copilot Chat:

```
@workspace Using agent-shaker, discover an A2A agent at http://localhost:8080 (discovering itself as a test)
```

**Copilot executes the `discover_a2a_agent` MCP tool and responds:**
> "I've discovered Agent Shaker at http://localhost:8080:
> 
> **Name:** Agent Shaker  
> **Version:** 1.0.0  
> **Capabilities:**
> - ✅ Task execution and management
> - ✅ Real-time streaming updates (SSE)
> - ✅ Artifact sharing (markdown contexts)
> - ✅ MCP protocol support
> 
> **Available Endpoints:**
> - POST /a2a/v1/message - Create tasks
> - POST /a2a/v1/message:stream - Create tasks with streaming
> - GET /a2a/v1/tasks - List all tasks
> - GET /a2a/v1/artifacts - List shared contexts
> 
> The agent is ready to receive tasks via A2A protocol."

## Step 6: Delegate a Task from VS Code (1 minute)

In Copilot Chat:

```
@workspace Using agent-shaker, delegate this task to http://localhost:8080: "Generate a summary of the current project structure and suggest improvements"
```

**Copilot executes the `delegate_to_a2a_agent` MCP tool.**

**Copilot response:**
> "I've delegated the task to Agent Shaker:
> 
> **Task ID:** `321d5cd0-db4d-425e-8d0c-06d0774666f1`  
> **Status:** pending → running → completed ✅
> 
> **Result:**
> Task received: Generate a summary of the current project structure and suggest improvements
> 
> The task completed successfully. You can check the full details at:
> http://localhost:8080/a2a/v1/tasks/321d5cd0-db4d-425e-8d0c-06d0774666f1"

## Step 7: Share Context as A2A Artifact (30 seconds)

In Copilot Chat:

```
@workspace Using agent-shaker, add a context with title "A2A Testing Notes" describing what we just tested. Include markdown formatting.
```

**Copilot creates a context/artifact:**

```markdown
# A2A Testing Notes

## What We Tested
✅ Agent discovery via /.well-known/agent-card.json
✅ Task creation via POST /a2a/v1/message
✅ Task status checking via GET /a2a/v1/tasks/{id}
✅ MCP tool integration (discover_a2a_agent, delegate_to_a2a_agent)

## Results
All A2A protocol features working correctly!

## Next Steps
- Test streaming with /a2a/v1/message:stream
- Try artifact retrieval via /a2a/v1/artifacts
- Connect real external A2A agents
```

## Step 8: View All A2A Artifacts (30 seconds)

In PowerShell:

```powershell
# List all artifacts (contexts) via A2A
$artifacts = Invoke-RestMethod http://localhost:8080/a2a/v1/artifacts
$artifacts.artifacts | ForEach-Object {
    Write-Host "`n📄 $($_.name)"
    Write-Host "   Type: $($_.type)"
    Write-Host "   Size: $($_.size) bytes"
    Write-Host "   URL: $($_.url)"
    Write-Host "   Tags: $($_.metadata.tags -join ', ')"
}
```

**Expected output:**
```
📄 A2A Testing Notes
   Type: markdown
   Size: 456 bytes
   URL: http://localhost:8080/a2a/v1/artifacts/ctx-12345
   Tags: testing, a2a, documentation
```

## Bonus: Test Streaming (Optional)

For real-time updates, test SSE streaming. In PowerShell:

```powershell
# Create a task with streaming
$body = @{
    message = @{
        content = "Process large dataset"
        format = "text"
    }
} | ConvertTo-Json

# Note: PowerShell doesn't natively support SSE well, 
# but you can see it working in the browser or with curl
Start-Process "http://localhost:8080/a2a/v1/message:stream"
```

Or use `curl` if available:

```bash
curl -X POST http://localhost:8080/a2a/v1/message:stream \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"message": {"content": "Long running task"}}'
```

**You'll see real-time events:**
```
event: task_created
data: {"task_id":"abc123","status":"pending"}

event: status
data: {"task_id":"abc123","status":"running"}

event: completed
data: {"task_id":"abc123","status":"completed","result":{...}}
```

## What You Just Accomplished

In 5 minutes, you:

✅ Verified Agent Shaker's A2A capabilities  
✅ Created an A2A-compatible project  
✅ Sent tasks via A2A protocol  
✅ Used MCP tools in VS Code to discover and delegate to A2A agents  
✅ Shared documentation as A2A artifacts  
✅ Listed all artifacts via A2A API  

## Common Commands Reference

### PowerShell Quick Commands

```powershell
# Discover agent
Invoke-RestMethod http://localhost:8080/.well-known/agent-card.json

# Create task
$task = @{message=@{content="Task text"}} | ConvertTo-Json
Invoke-RestMethod -Uri http://localhost:8080/a2a/v1/message -Method Post -Body $task -ContentType "application/json"

# List tasks
Invoke-RestMethod http://localhost:8080/a2a/v1/tasks

# Get specific task
Invoke-RestMethod http://localhost:8080/a2a/v1/tasks/{task-id}

# List artifacts
Invoke-RestMethod http://localhost:8080/a2a/v1/artifacts

# Get specific artifact
Invoke-RestMethod http://localhost:8080/a2a/v1/artifacts/{artifact-id}
```

### VS Code Copilot Commands

```
@workspace Using agent-shaker, discover the A2A agent at {url}

@workspace Using agent-shaker, delegate this task to {url}: "{message}"

@workspace Using agent-shaker, check the status of A2A task {task-id} from agent {url}

@workspace Using agent-shaker, add a context titled "{title}" with this content: {markdown}

@workspace Using agent-shaker, list all contexts/artifacts
```

## Next Steps

1. **Connect Real External A2A Agents** - Configure actual external agents
2. **Set Up Continuous Monitoring** - Build agents that watch for new artifacts
3. **Create Agent Pipelines** - Chain multiple A2A agents together
4. **Explore Streaming** - Use SSE for long-running tasks
5. **Read the Full Guide** - Check out [A2A_VSCODE_USE_CASE.md](./A2A_VSCODE_USE_CASE.md)

## Troubleshooting

**Agent Shaker not responding?**
```powershell
# Check if it's running
Test-NetConnection -ComputerName localhost -Port 8080
```

**Can't see MCP tools in Copilot?**
- Restart VS Code
- Check MCP configuration in settings.json
- Verify mcp-bridge.js is running

**Tasks stay in pending?**
- Check Agent Shaker logs in terminal
- Verify database connection (if using PostgreSQL)

**Discovery error: "cannot unmarshal object into Go struct field AgentCard.capabilities"?**
- This error is fixed in Agent Shaker v1.0.0+
- Agent Shaker now supports both array and object formats for capabilities
- If you see this error, ensure you're running the latest version
- The external agent you're discovering uses an alternative format that is now automatically handled

## Resources

- [Full A2A Integration Guide](./A2A_INTEGRATION.md)
- [Real-World Use Case](./A2A_VSCODE_USE_CASE.md)
- [A2A Protocol Specification](https://a2a-protocol.org/)
- [MCP Quickstart](./MCP_QUICKSTART.md)
