# Agents Page Implementation

## Backend Changes

### Updated: `internal/handlers/agents.go`

The `ListAgents` endpoint has been enhanced to support listing all agents globally:

**Previous Behavior:**
- Required `project_id` query parameter
- Only returned agents for a specific project

**New Behavior:**
- `project_id` is now **optional**
- If `project_id` is provided: returns agents for that specific project
- If `project_id` is **not** provided: returns **all agents** across all projects

### API Endpoint

```
GET /api/agents
GET /api/agents?project_id={uuid}
```

#### Response Format
```json
[
  {
    "id": "uuid",
    "project_id": "uuid",
    "name": "Frontend Agent",
    "role": "frontend",
    "team": "Development",
    "status": "active",
    "last_seen": "2026-01-21T14:30:00Z",
    "created_at": "2026-01-20T10:00:00Z"
  }
]
```

## Frontend Implementation

### Agents Page (`web/src/views/Agents.vue`)

**Features:**
- ✅ Modern card-based layout
- ✅ Responsive grid (1-3 columns)
- ✅ Role badges (frontend/backend with colors)
- ✅ Status indicators (active/inactive)
- ✅ Loading and error states
- ✅ Empty state message
- ✅ Formatted dates

**Styling:**
- Uses Tailwind CSS for modern design
- Color-coded badges:
  - **Frontend Role**: Blue background
  - **Backend Role**: Pink background
  - **Active Status**: Green background
  - **Inactive Status**: Red background

### Store (`web/src/stores/agentStore.js`)

**Methods:**
- `fetchAgents()` - Fetches all agents (no project filter)
- `fetchProjectAgents(projectId)` - Fetches agents for specific project
- `createAgent(data)` - Creates new agent
- `updateAgentStatus(id, status)` - Updates agent status

### API Service (`web/src/services/api.js`)

**Endpoints:**
- `getAgents()` - GET /api/agents (all agents)
- `getProjectAgents(projectId)` - GET /api/agents?project_id={id}
- `createAgent(data)` - POST /api/agents
- `updateAgentStatus(id, status)` - PUT /api/agents/{id}/status

## Testing the Agents Page

### 1. Rebuild the Backend

```bash
# From project root
go build -o mcp-tracker ./cmd/server
```

### 2. Restart the Backend Server

```bash
# Stop current server (Ctrl+C)
# Start new server
./mcp-tracker
```

### 3. The Frontend is Already Running

The dev server should already be running at http://localhost:3000

### 4. Navigate to Agents Page

Click on "Agents" in the navigation or go to:
```
http://localhost:3000/agents
```

## Creating Sample Agents

### Option 1: Via API (curl)

```bash
# Create a project first
curl -X POST http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Project",
    "description": "A test project"
  }'

# Create agents (replace PROJECT_ID with actual UUID)
curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "PROJECT_ID",
    "name": "Frontend Agent",
    "role": "frontend",
    "team": "Development"
  }'

curl -X POST http://localhost:8080/api/agents \
  -H "Content-Type: application/json" \
  -d '{
    "project_id": "PROJECT_ID",
    "name": "Backend Agent",
    "role": "backend",
    "team": "Development"
  }'
```

### Option 2: Via PowerShell

```powershell
# Create a project
$project = Invoke-RestMethod -Uri "http://localhost:8080/api/projects" `
  -Method Post `
  -ContentType "application/json" `
  -Body '{"name":"Test Project","description":"A test project"}'

$projectId = $project.id

# Create frontend agent
Invoke-RestMethod -Uri "http://localhost:8080/api/agents" `
  -Method Post `
  -ContentType "application/json" `
  -Body "{`"project_id`":`"$projectId`",`"name`":`"Frontend Agent`",`"role`":`"frontend`",`"team`":`"Development`"}"

# Create backend agent
Invoke-RestMethod -Uri "http://localhost:8080/api/agents" `
  -Method Post `
  -ContentType "application/json" `
  -Body "{`"project_id`":`"$projectId`",`"name`":`"Backend Agent`",`"role`":`"backend`",`"team`":`"Development`"}"
```

### Option 3: Via UI

1. Go to Projects page
2. Create a new project
3. Click on the project to view details
4. Go to "Agents" tab
5. Click "+ Add Agent"
6. Fill in the form and submit

## Expected Behavior

### With No Agents
- Shows empty state message: "No agents registered"
- Provides helpful text: "Agents will appear here when they are registered to projects"

### With Agents
- Displays agents in a responsive grid
- Each agent card shows:
  - Agent name (header)
  - Status badge (active/inactive)
  - Role badge with color
  - Team name
  - Project ID
  - Last seen timestamp
  - Created timestamp

### Loading State
- Shows spinner and "Loading agents..." message

### Error State
- Shows red error banner with error message

## Troubleshooting

### Problem: Empty Page
**Solution:** Check if backend is running and accessible at http://localhost:8080

### Problem: API Error
**Solution:** 
1. Check browser console for error details
2. Verify backend is running
3. Check backend logs for errors

### Problem: No Agents Showing
**Solution:**
1. Create a project first
2. Add agents to the project
3. Refresh the agents page

### Problem: Backend Changes Not Applied
**Solution:**
1. Rebuild the backend: `go build -o mcp-tracker ./cmd/server`
2. Restart the backend server

## File Changes Summary

### Modified Files
1. `internal/handlers/agents.go` - Made project_id optional in ListAgents
2. `web/src/views/Agents.vue` - Already had Tailwind styling (verified)

### No Changes Needed
- `web/src/stores/agentStore.js` - Already correct
- `web/src/services/api.js` - Already correct
- `web/src/router/index.js` - Already has agents route

## Integration with Other Pages

### Dashboard
- Shows recent/active agents
- Links to full agents page

### Project Detail
- Shows agents for specific project
- Uses `fetchProjectAgents(projectId)` method

### Agents Page
- Shows all agents globally
- Uses `fetchAgents()` method

## Future Enhancements

- [ ] Add agent filtering by status
- [ ] Add agent filtering by role
- [ ] Add agent filtering by team
- [ ] Add agent filtering by project
- [ ] Add search functionality
- [ ] Add agent details modal
- [ ] Add agent editing capability
- [ ] Add agent deletion capability
- [ ] Add bulk operations
- [ ] Add agent activity timeline
- [ ] Add agent performance metrics

---

**Status**: ✅ Backend updated, Frontend ready  
**Next Step**: Rebuild and restart backend server  
**Date**: January 21, 2026
