# MCP Task Tracker Integration for GitHub Copilot

## Overview

This guide explains how to integrate GitHub Copilot agents with MCP Task Tracker for coordinated work in microservices architectures.

## Setup

1. **Deploy MCP Task Tracker**
   ```bash
   docker-compose up -d
   ```

2. **Create Project**
   ```bash
   curl -X POST http://localhost:8080/api/projects \
     -H "Content-Type: application/json" \
     -d '{
       "name": "InvoiceAI",
       "description": "AI-powered invoice processing"
     }'
   ```

3. **Note the project_id** from the response

## Agent Registration

Each GitHub Copilot instance should register as an agent:

```bash
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "your-project-id",
    "name": "Backend-Copilot",
    "role": "backend",
    "team": "Backend Team"
  }'
```

Save the `agent_id` from the response.

## Copilot Instructions

Create `.copilot/instructions.md` in your project:

```markdown
# MCP Task Tracker Integration

You are integrated with MCP Task Tracker for coordinated work.

## Configuration
- MCP Server: http://localhost:8080
- Project ID: {your-project-id}
- Agent ID: {your-agent-id}

## Workflow

### On Session Start
1. Check for assigned tasks:
   ```bash
   GET /api/tasks?project_id={project-id}&assigned_to={agent-id}&status=pending
   ```

2. Read relevant documentation:
   ```bash
   GET /api/contexts?project_id={project-id}&tags=api,backend
   ```

### During Work
1. When starting a task:
   ```bash
   PUT /api/tasks/{task-id}
   Body: {"status": "in_progress"}
   ```

2. If blocked by dependency:
   ```bash
   PUT /api/tasks/{task-id}
   Body: {
     "status": "blocked",
     "output": "Waiting for authentication endpoint from frontend"
   }
   ```

3. Create task for other team:
   ```bash
   POST /api/tasks
   Body: {
     "project_id": "{project-id}",
     "title": "Need authentication endpoint",
     "description": "Backend needs POST /api/auth/login",
     "priority": "high",
     "created_by": "{your-agent-id}",
     "assigned_to": "{frontend-agent-id}"
   }
   ```

### On Completion
1. Mark task as done:
   ```bash
   PUT /api/tasks/{task-id}
   Body: {
     "status": "done",
     "output": "API implemented at /api/invoices with CRUD operations"
   }
   ```

2. Add documentation:
   ```bash
   POST /api/contexts
   Body: {
     "project_id": "{project-id}",
     "agent_id": "{agent-id}",
     "task_id": "{task-id}",
     "title": "Invoice API Documentation",
     "content": "# Invoice API\n\n## GET /api/invoices\nReturns list...",
     "tags": ["api", "documentation", "backend"]
   }
   ```

3. Create follow-up tasks if needed

## Best Practices

1. **Always check for tasks** when starting work
2. **Update status** to keep team informed
3. **Document everything** for other agents
4. **Use descriptive outputs** when completing tasks
5. **Tag documentation** properly for easy search
6. **Create tasks** instead of making assumptions
7. **Use blocked status** when dependencies exist

## Communication Examples

### Backend → Frontend
```bash
POST /api/tasks
{
  "title": "Implement invoice list UI",
  "description": "API ready at GET /api/invoices. See documentation for schema.",
  "priority": "high",
  "created_by": "{backend-agent-id}",
  "assigned_to": "{frontend-agent-id}"
}
```

### Frontend → Backend
```bash
POST /api/tasks
{
  "title": "Add pagination to invoice API",
  "description": "Need page and limit params. Response should include total count.",
  "priority": "medium",
  "created_by": "{frontend-agent-id}",
  "assigned_to": "{backend-agent-id}"
}
```

## Monitoring

Watch for updates in real-time:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?project_id={project-id}');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.type === 'task_update') {
    console.log('Task updated:', data.payload);
  }
};
```
```

## Example Workflow

### Complete Feature Implementation

1. **Backend Copilot** gets task to create invoice API
2. Backend checks existing documentation
3. Backend implements API endpoints
4. Backend adds comprehensive API documentation
5. Backend creates task for Frontend with API details
6. **Frontend Copilot** gets notification
7. Frontend reads API documentation
8. Frontend implements UI components
9. Frontend adds component documentation
10. Both mark tasks as done

### Handling Changes

1. **Frontend** discovers API needs additional field
2. Frontend creates task for Backend with description
3. Backend gets notification, prioritizes task
4. Backend updates API and documentation
5. Backend marks task as done
6. Frontend gets notification, updates components

## Troubleshooting

### Task not showing up
- Check project_id is correct
- Verify agent_id assignment
- Check task filters (status, assigned_to)

### WebSocket disconnects
- System auto-reconnects after 3 seconds
- Check network connectivity
- Verify project_id in WebSocket URL

### Documentation not found
- Check tags match your search
- Verify project_id filter
- Ensure documentation was created

## Advanced Usage

### Multi-agent Coordination
```bash
# DevOps creates infrastructure task
POST /api/tasks
{
  "title": "Set up production database",
  "assigned_to": "{devops-agent-id}"
}

# Backend waits for infrastructure
PUT /api/tasks/{backend-task-id}
{
  "status": "blocked",
  "output": "Waiting for production database setup"
}

# DevOps completes and notifies
PUT /api/tasks/{devops-task-id}
{
  "status": "done",
  "output": "DB ready at prod-db.example.com:5432"
}
```

### Documentation Search
```bash
# Find all API documentation
GET /api/contexts?project_id={id}&tags=api

# Find frontend components
GET /api/contexts?project_id={id}&tags=component,frontend

# Find specific documentation
GET /api/contexts?project_id={id}&tags=authentication
```
