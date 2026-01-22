# Enhanced MCP JSON Configuration for VS Code

## Overview

The enhanced `mcp.json` configuration provides comprehensive metadata for VS Code's Model Context Protocol (MCP) integration, enabling better project detection, agent identification, and tool definitions.

The Agent Shaker MCP server supports context-aware connections that allow multiple developers to work on the same project with different agent identities. Each developer can configure their VS Code to connect as a specific agent on a specific project.

## Configuration Structure

### 1. **MCP Servers Section**

```json
{
  "mcpServers": {
    "agent-shaker": {
      "url": "http://localhost:8080/api",
      "type": "http",
      ...
    }
  }
}
```

The main server configuration that VS Code will connect to.

### 2. **Metadata**

```json
"metadata": {
  "name": "Agent Shaker MCP Server",
  "version": "1.0.0",
  "description": "Multi-agent coordination platform for collaborative development",
  "capabilities": [
    "resources",
    "tools",
    "prompts",
    "context-sharing"
  ]
}
```

Describes the server's capabilities and purpose.

### 3. **Project Configuration**

```json
"project": {
  "id": "uuid",
  "name": "Project Name",
  "description": "Project description",
  "status": "active",
  "type": "multi-agent",
  "root": "${workspaceFolder}",
  "detect": {
    "patterns": ["**/*.go", "**/*.vue", "**/*.js", "go.mod", "package.json"],
    "excludePatterns": ["node_modules", "vendor", ".git", "dist", "build"]
  }
}
```

**Features:**
- Project identification (ID, name, status)
- File pattern detection for project type recognition
- Exclude patterns to ignore certain directories
- Workspace folder reference

### 4. **Agent Configuration**

```json
"agent": {
  "id": "agent-uuid",
  "name": "Agent Name",
  "role": "frontend|backend",
  "team": "team-name",
  "status": "active",
  "type": "ai-developer",
  "capabilities": [
    "ui-development",
    "task-management",
    "context-sharing",
    "documentation"
  ],
  "context": {
    "projectId": "uuid",
    "projectName": "Project Name",
    "agentRole": "frontend",
    "apiBaseUrl": "http://localhost:8080/api"
  },
  "behavior": {
    "autoReconnect": true,
    "maxRetries": 3,
    "timeout": 30000,
    "healthCheckInterval": 60000
  }
}
```

**Features:**
- Complete agent identity (ID, name, role, team)
- Agent capabilities based on role
- Context information for agent operations
- Behavior settings (reconnection, timeouts, health checks)

### 5. **Resources**

```json
"resources": {
  "baseUrl": "http://localhost:8080/api",
  "websocket": "ws://localhost:8080/ws",
  "endpoints": {
    "health": "/health",
    "projects": "/projects",
    "agents": "/agents",
    "tasks": "/tasks",
    "contexts": "/contexts",
    "myTasks": "/agents/{agent_id}/tasks",
    "myAgent": "/agents/{agent_id}",
    "projectAgents": "/projects/{project_id}/agents",
    "projectTasks": "/projects/{project_id}/tasks",
    "projectContexts": "/projects/{project_id}/contexts"
  }
}
```

**Features:**
- Complete API endpoint mapping
- WebSocket URL for real-time updates
- Agent-specific and project-specific endpoints

### 6. **Tools**

```json
"tools": [
  {
    "name": "get_my_tasks",
    "description": "Get tasks assigned to this agent",
    "category": "task-management",
    "endpoint": "/agents/{agent_id}/tasks",
    "method": "GET"
  },
  {
    "name": "update_task_status",
    "description": "Update the status of a task",
    "category": "task-management",
    "endpoint": "/tasks/{task_id}/status",
    "method": "PUT",
    "parameters": {
      "task_id": "string",
      "status": "enum[pending,in_progress,done,blocked]"
    }
  },
  ...
]
```

**Available Tools:**
- `get_my_tasks` - Get tasks assigned to agent
- `update_task_status` - Update task status
- `create_task` - Create new tasks
- `get_project_contexts` - Get documentation/contexts
- `add_context` - Add documentation
- `get_project_agents` - Get all agents in project
- `get_dashboard_stats` - Get project statistics

### 7. **Security**

```json
"security": {
  "authentication": "none",
  "cors": {
    "enabled": true,
    "allowOrigins": ["http://localhost:5173", "http://localhost:3000"]
  }
}
```

Security configuration for API access.

### 8. **Monitoring**

```json
"monitoring": {
  "healthCheck": {
    "enabled": true,
    "interval": 30000,
    "endpoint": "/health"
  },
  "logging": {
    "level": "info",
    "format": "json",
    "includeTimestamp": true
  }
}
```

Health monitoring and logging configuration.

### 9. **Development Settings**

```json
"development": {
  "hotReload": true,
  "debugMode": false,
  "mockData": false
}
```

Development-specific settings.

### 10. **Global Settings**

```json
"globalSettings": {
  "autoStart": true,
  "autoReconnect": true,
  "connectionTimeout": 5000,
  "defaultPort": 8080
}
```

Global MCP client settings.

### 11. **VS Code Integration**

```json
"vscodeIntegration": {
  "terminal": {
    "env": {
      "MCP_AGENT_NAME": "agent-name",
      "MCP_AGENT_ID": "agent-id",
      "MCP_PROJECT_ID": "project-id",
      "MCP_PROJECT_NAME": "project-name",
      "MCP_API_URL": "http://localhost:8080/api"
    }
  },
  "tasks": {
    "autoDetect": true,
    "problemMatcher": ["$eslint-stylish", "$tsc"]
  }
}
```

VS Code-specific integration settings for terminal environment and task detection.

## Multi-Developer Scenario

### Example: E-Commerce Platform Team

Suppose you have a project "E-Commerce Platform" with two developers:
- **Alice** - Frontend Developer (React Frontend Agent)
- **Bob** - Backend Developer (Node Backend Agent)

Each developer configures their VS Code with different agent/project context:

**Alice's `.vscode/mcp.json`:**
```json
{
  "servers": {
    "agent-shaker": {
      "type": "http",
      "url": "http://localhost:8080?project_id=550e8400-e29b-41d4-a716-446655440001&agent_id=660e8400-e29b-41d4-a716-446655440001"
    }
  }
}
```

**Bob's `.vscode/mcp.json`:**
```json
{
  "servers": {
    "agent-shaker": {
      "type": "http",
      "url": "http://localhost:8080?project_id=550e8400-e29b-41d4-a716-446655440001&agent_id=660e8400-e29b-41d4-a716-446655440002"
    }
  }
}
```

Now when Alice asks VS Code Copilot about her tasks, it returns only frontend tasks assigned to her agent. Bob gets only backend tasks.

## URL Parameters

| Parameter | Description |
|-----------|-------------|
| `project_id` | UUID of the project to work on |
| `agent_id` | UUID of the agent identity for this connection |

## Alternative: Using Headers

You can also pass context via HTTP headers:
- `X-Project-ID`: Project UUID
- `X-Agent-ID`: Agent UUID

## Available MCP Tools

### Context-Aware Tools (use configured project/agent automatically)

| Tool | Description | Requires |
|------|-------------|----------|
| `get_my_identity` | Get current agent identity and project info | project_id or agent_id |
| `get_my_project` | Get assigned project details with task summary | project_id |
| `get_my_tasks` | Get tasks assigned to this agent | agent_id |
| `update_my_status` | Update agent status (idle, working, blocked, offline) | agent_id |
| `claim_task` | Claim a task and start working on it | agent_id |
| `complete_task` | Mark a task as done | agent_id |

### General Tools (work without context)

| Tool | Description |
|------|-------------|
| `list_projects` | List all projects in the system |
| `get_project` | Get details of a specific project |
| `list_agents` | List all agents, optionally filtered by project |
| `get_agent` | Get details of a specific agent |
| `list_tasks` | List tasks, optionally filtered by project/agent/status |
| `create_task` | Create a new task in a project |
| `update_task_status` | Update the status of a task |
| `list_contexts` | List documentation/contexts for a project |
| `add_context` | Add documentation or context to a project |
| `get_dashboard` | Get dashboard statistics and overview |

## Getting Project and Agent IDs

### From the Web UI
1. Go to the Agent Shaker dashboard
2. Navigate to your project
3. Click on an agent to view MCP setup instructions
4. Download the generated `mcp.json` file

### From the API
```bash
# List all projects
curl http://localhost:8080/api/projects

# List agents for a project
curl http://localhost:8080/api/projects/{project_id}/agents
```

### From MCP Tools
```bash
# Using MCP protocol
curl -X POST http://localhost:8080 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_projects","arguments":{}}}'
```

## Example Workflows

### Workflow 1: Agent Checks Identity
```
User: "Who am I and what project am I working on?"
AI calls: get_my_identity
Response: Shows agent name, role, project name, and status
```

### Workflow 2: Agent Gets Tasks
```
User: "What tasks are assigned to me?"
AI calls: get_my_tasks
Response: Lists all tasks assigned to the configured agent
```

### Workflow 3: Agent Claims a Task
```
User: "I want to work on task 770e8400-e29b-41d4-a716-446655440001"
AI calls: claim_task with task_id
Response: Task claimed and status set to in_progress
```

### Workflow 4: Agent Completes Task
```
User: "I finished the product listing page task"
AI calls: complete_task with task_id
Response: Task marked as done
```

### Workflow 5: Agent Updates Status
```
User: "I'm blocked waiting for API documentation"
AI calls: update_my_status with status="blocked"
Response: Agent status updated
```

## Usage

### 1. **Download the Configuration**

In the Agent Shaker web UI:
1. Navigate to a project
2. Click on an agent to open MCP Setup
3. Download the `.vscode/mcp.json` file (marked with "Enhanced" badge)

### 2. **Install in Your Project**

```bash
# Create .vscode directory if it doesn't exist
mkdir -p .vscode

# Copy the downloaded mcp.json
cp ~/Downloads/mcp.json .vscode/mcp.json
```

### 3. **Restart VS Code**

After placing the file, restart VS Code to load the configuration.

### 4. **Verify Connection**

The MCP server should automatically connect. Check the VS Code output panel for MCP-related logs.

## Quick Setup

1. **Start the MCP server:**
   ```bash
   docker-compose up -d
   ./server.exe
   ```

2. **Create/copy mcp.json to your project:**
   ```bash
   mkdir -p .vscode
   # Download from Agent Shaker UI or create manually
   ```

3. **Reload VS Code** to pick up the MCP configuration

4. **Test the connection:**
   - Open VS Code Chat (Copilot)
   - Ask: "What's my agent identity?"
   - You should see your configured agent and project info

## Troubleshooting

### Configuration Not Loading
- Ensure the file is named `mcp.json` and placed in `.vscode/` directory
- Check for JSON syntax errors
- Restart VS Code after making changes

### Connection Failures
- Verify the server URL is correct
- Check that the MCP server is running
- Review the `globalSettings.connectionTimeout` value

### Tool Execution Issues
- Verify endpoint URLs are correct
- Check authentication settings if enabled
- Review tool parameters match the API requirements

### "No project_id or agent_id configured"
Your mcp.json URL doesn't include the context parameters. Add `?project_id=UUID&agent_id=UUID` to the URL.

### "405 Method Not Allowed"
Make sure you're connecting to the MCP endpoint (port 8080 root) not the REST API endpoint.

### "Database not connected"
The MCP server needs a PostgreSQL connection. Check that docker-compose is running and DATABASE_URL is set correctly.

## Benefits

1. **Automatic Project Detection**: VS Code can identify your project type based on patterns
2. **Agent Identity**: Full agent information available to MCP clients
3. **Tool Definitions**: All available API operations are documented
4. **Resource Mapping**: Complete endpoint mapping for easy discovery
5. **Environment Integration**: Terminal environment variables automatically set
6. **Health Monitoring**: Automatic health checks and reconnection
7. **Development Support**: Hot reload and debug mode settings

## Comparison with Basic Configuration

### Basic `mcp.json`:
```json
{
  "servers": {
    "agent-shaker": {
      "url": "http://localhost:8080",
      "type": "http"
    }
  },
  "inputs": []
}
```

### Enhanced `mcp.json`:
- ✅ Project detection and metadata
- ✅ Complete agent identity and capabilities
- ✅ All API endpoints documented
- ✅ Tool definitions with parameters
- ✅ WebSocket support
- ✅ Health monitoring
- ✅ VS Code terminal integration
- ✅ Security and logging configuration
- ✅ Development settings

## Related Documentation

- [MCP Protocol Specification](https://spec.modelcontextprotocol.io/)
- [Agent Setup Guide](./AGENT_SETUP_GUIDE.md)
- [Copilot MCP Integration](./COPILOT_MCP_INTEGRATION.md)
- [API Documentation](./API.md)

## Future Enhancements

Potential additions to the configuration:
- Authentication tokens
- Custom tool definitions
- Prompt templates
- Resource caching strategies
- Multi-server configurations
- Team-specific settings
