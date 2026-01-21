# 🚀 MCP Task Tracker

> AI Agent Task Coordination System for Microservices Architecture

MCP Task Tracker is a real-time task coordination system designed for AI agents (like GitHub Copilot) working in microservices architectures. It enables backend and frontend teams to synchronize work, exchange tasks, and share documentation in real-time.

## Features

- ✅ **Project Management** - Create and manage multiple projects
- 🤖 **Agent Registration** - Register AI agents (backend, frontend, devops, etc.)
- 📋 **Task Coordination** - Create, assign, and track tasks across teams
- 📚 **Documentation Hub** - Centralized markdown documentation with tags
- 🔄 **Real-time Updates** - WebSocket-based live notifications
- 🎯 **Cross-team Communication** - Backend ↔ Frontend task handoff
- 🔍 **Advanced Filtering** - Filter by status, priority, tags, agents

## Architecture

### Components

1. **Go REST API Server** - Core backend with HTTP handlers
2. **PostgreSQL Database** - Persistent storage for all entities
3. **WebSocket Hub** - Real-time notification system
4. **Web UI** - Management dashboard (HTML/JS)

### Data Model

- **Project** - Container for agents and tasks
- **Agent** - AI agent (Copilot) with role and team
- **Task** - Work item with status tracking
- **Context** - Documentation with markdown and tags

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/techbuzzz/agent-shaker.git
cd agent-shaker

# Start all services
docker-compose up -d

# Check health
curl http://localhost:8080/health
```

The application will be available at:
- Web UI: http://localhost:8080
- API: http://localhost:8080/api
- WebSocket: ws://localhost:8080/ws

### Local Development

#### Prerequisites

- Go 1.21+
- PostgreSQL 15+

#### Setup

```bash
# Install dependencies
go mod download

# Set up environment
cp .env.example .env
# Edit .env with your database credentials

# Start PostgreSQL
# Create database: mcp_tracker

# Run the server
go run cmd/server/main.go
```

## API Documentation

### Projects

#### Create Project
```bash
POST /api/projects
Content-Type: application/json

{
  "name": "InvoiceAI",
  "description": "AI-powered invoice processing"
}
```

#### List Projects
```bash
GET /api/projects
```

#### Get Project
```bash
GET /api/projects/{id}
```

### Agents

#### Register Agent
```bash
POST /api/agents
Content-Type: application/json

{
  "project_id": "uuid",
  "name": "Backend-Copilot",
  "role": "backend",
  "team": "Backend Team"
}
```

#### List Agents
```bash
GET /api/agents?project_id={uuid}
```

#### Update Agent Status
```bash
PUT /api/agents/{id}/status
Content-Type: application/json

{
  "status": "active"
}
```

### Tasks

#### Create Task
```bash
POST /api/tasks
Content-Type: application/json

{
  "project_id": "uuid",
  "title": "Implement invoice API",
  "description": "Create REST endpoint",
  "priority": "high",
  "created_by": "agent-uuid",
  "assigned_to": "agent-uuid"
}
```

#### List Tasks
```bash
GET /api/tasks?project_id={uuid}&status=pending&assigned_to={agent-uuid}
```

#### Get Task
```bash
GET /api/tasks/{id}
```

#### Update Task
```bash
PUT /api/tasks/{id}
Content-Type: application/json

{
  "status": "done",
  "output": "API implemented at /api/invoices"
}
```

### Documentation (Contexts)

#### Add Documentation
```bash
POST /api/contexts
Content-Type: application/json

{
  "project_id": "uuid",
  "agent_id": "uuid",
  "task_id": "uuid",
  "title": "Invoice API Documentation",
  "content": "# Invoice API\n\n## Endpoints...",
  "tags": ["api", "documentation"]
}
```

#### List Documentation
```bash
GET /api/contexts?project_id={uuid}&tags=api,documentation
```

#### Get Documentation
```bash
GET /api/contexts/{id}
```

### WebSocket

#### Connect to Project Updates
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?project_id={uuid}');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Update:', data.type, data.payload);
};
```

Event types:
- `task_update` - Task created or updated
- `agent_update` - Agent registered or status changed
- `context_added` - New documentation added

## Usage Scenarios

### Scenario 1: New Feature Implementation

1. **Backend Team** creates a task for API implementation
2. Backend implements API and adds documentation
3. Backend creates task for Frontend with API details
4. **Frontend Team** reads API documentation
5. Frontend implements UI and completes task

### Scenario 2: API Change Request

1. **Frontend** discovers missing API data
2. Frontend creates task for Backend describing the need
3. **Backend** receives notification, updates API
4. Backend updates documentation
5. Frontend gets notified and updates components

### Scenario 3: Blocked Task

1. **Agent** starts work and discovers dependency
2. Agent changes status to `blocked`, adds reason
3. Agent creates task for dependency team
4. Dependency team gets priority notification
5. After resolution, original agent continues

## Task Statuses

- `pending` - Waiting to start
- `in_progress` - Currently being worked on
- `blocked` - Waiting for dependency
- `done` - Completed
- `cancelled` - Cancelled

## Agent Statuses

- `active` - Currently working
- `idle` - Waiting for tasks
- `offline` - Disconnected

## GitHub Copilot Integration

Create `.copilot/instructions.md` in your project:

```markdown
# MCP Task Tracker Integration

## On Start
1. Register as agent: POST /api/agents
2. Get tasks: GET /api/tasks?project_id=X&assigned_to=Y
3. Read docs: GET /api/contexts?project_id=X

## During Work
1. Update status: PUT /api/tasks/{id} {"status": "in_progress"}
2. If blocked: PUT /api/tasks/{id} {"status": "blocked", "output": "reason"}

## On Complete
1. Mark done: PUT /api/tasks/{id} {"status": "done", "output": "result"}
2. Add docs: POST /api/contexts
3. Create follow-up tasks: POST /api/tasks
```

## Database Schema

```sql
-- Projects
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agents
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(100),
    team VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tasks
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    priority VARCHAR(50) DEFAULT 'medium',
    created_by UUID REFERENCES agents(id),
    assigned_to UUID REFERENCES agents(id),
    output TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Contexts (Documentation)
CREATE TABLE contexts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    agent_id UUID REFERENCES agents(id),
    task_id UUID REFERENCES tasks(id),
    title VARCHAR(255) NOT NULL,
    content TEXT,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Development

### Build

```bash
go build -o mcp-server cmd/server/main.go
```

### Run Tests

```bash
go test ./...
```

### Build Docker Image

```bash
docker build -t mcp-task-tracker .
```

## Configuration

Environment variables:

- `DATABASE_URL` - PostgreSQL connection string (default: `postgres://mcp:secret@localhost:5432/mcp_tracker?sslmode=disable`)
- `PORT` - Server port (default: `8080`)

## License

MIT License - see [LICENSE](LICENSE) file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions, please open an issue on GitHub.
