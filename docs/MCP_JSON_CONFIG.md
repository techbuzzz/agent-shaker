# Enhanced MCP JSON Configuration for VS Code

## Overview

The enhanced `mcp.json` configuration provides comprehensive metadata for VS Code's Model Context Protocol (MCP) integration, enabling better project detection, agent identification, and tool definitions.

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

## Example Use Cases

### 1. **Copilot Agent Identity**
The agent information allows GitHub Copilot to understand its role and capabilities in the project.

### 2. **Automated Task Management**
Tools like `get_my_tasks` and `update_task_status` enable automated task workflows.

### 3. **Project Context Sharing**
The `add_context` and `get_project_contexts` tools enable documentation sharing between agents.

### 4. **Real-time Collaboration**
WebSocket endpoint enables real-time updates for task and agent status changes.

### 5. **Multi-Agent Coordination**
The `get_project_agents` tool helps agents discover and coordinate with each other.

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
