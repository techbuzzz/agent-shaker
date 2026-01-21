# API Fixes and Store Updates Summary

## Issues Fixed

### 1. API Endpoint Mismatches (404 Errors)
The web application was failing with 404 errors because the frontend was calling API endpoints that didn't exist in the backend. For example:
- `GET /api/projects/{id}/agents` - 404 (doesn't exist)

### 2. WebSocket Connection Failures (500 Error)
WebSocket handshake was failing with HTTP 500 error due to nil pointer dereference in client registration.

## Root Cause
Mismatch between frontend API calls and backend route definitions:
- Backend uses query parameters for filtering (e.g., `/api/agents?project_id=xxx`)
- Frontend was using nested resource routes (e.g., `/api/projects/{id}/agents`)

## Changes Made

### Backend Changes (Go)

#### 1. Updated Task Handler (`internal/handlers/tasks.go`)
- Modified `ListTasks` to support **optional** query parameters instead of requiring `project_id`
- Now supports filtering by:
  - `project_id` - filter tasks by project
  - `agent_id` - filter tasks by assigned agent
  - `status` - filter tasks by status
- Returns all tasks if no filters are provided

#### 2. Added Missing Handlers
- **`GetAgent`** (`internal/handlers/agents.go`) - Get single agent by ID
- **`UpdateTaskStatus`** (`internal/handlers/tasks.go`) - Update only task status

#### 3. Updated Routes (`cmd/server/main.go`)
Added missing route registrations:
```go
// Agents
api.HandleFunc("/agents/{id}", agentHandler.GetAgent).Methods("GET")

// Tasks
api.HandleFunc("/tasks/{id}/status", taskHandler.UpdateTaskStatus).Methods("PUT")
```

### Frontend Changes (JavaScript/Vue)

#### 1. API Service (`web/src/services/api.js`)
Fixed all endpoint paths to match backend routes:

**Before:**
```javascript
getProjectAgents(projectId) {
  return api.get(`/projects/${projectId}/agents`)
}
getProjectTasks(projectId) {
  return api.get(`/projects/${projectId}/tasks`)
}
```

**After:**
```javascript
getProjectAgents(projectId) {
  return api.get('/agents', { params: { project_id: projectId } })
}
getProjectTasks(projectId) {
  return api.get('/tasks', { params: { project_id: projectId } })
}
```

Added missing CRUD methods:
- `updateProject(id, data)` / `deleteProject(id)`
- `updateAgent(id, data)` / `deleteAgent(id)`
- `updateTask(id, data)` / `deleteTask(id)`

#### 2. Store Updates

**Project Store** (`web/src/stores/projectStore.js`)
- Added `updateProject` and `deleteProject` actions
- Added `clearError` helper
- Improved error handling with proper error messages
- Return values from async actions for better component integration

**Agent Store** (`web/src/stores/agentStore.js`)
- Added `currentAgent` state
- Added `fetchAgent` action for single agent retrieval
- Added `updateAgent` and `deleteAgent` actions
- Updated `fetchAgents` to accept optional `projectId` parameter
- Improved `updateAgentStatus` to update both local state and current agent
- Added `clearError` helper

**Task Store** (`web/src/stores/taskStore.js`)
- Added `currentTask` state
- Added `fetchTask` action for single task retrieval
- Updated `fetchTasks` to accept optional filter parameters
- Added `updateTask` and `deleteTask` actions
- Improved `updateTaskStatus` to update both local state and current task
- Added `clearError` helper

**Context Store** (`web/src/stores/contextStore.js`)
- Improved error handling and logging
- Better state synchronization for current context
- Added `clearError` helper

## API Endpoint Reference

### Correct Endpoint Usage

#### Projects
- `GET /api/projects` - List all projects
- `GET /api/projects/{id}` - Get single project
- `POST /api/projects` - Create project

#### Agents
- `GET /api/agents` - List all agents
- `GET /api/agents?project_id={id}` - List agents for a project
- `GET /api/agents/{id}` - Get single agent
- `POST /api/agents` - Create agent
- `PUT /api/agents/{id}/status` - Update agent status

#### Tasks
- `GET /api/tasks` - List all tasks
- `GET /api/tasks?project_id={id}` - List tasks for a project
- `GET /api/tasks?agent_id={id}` - List tasks for an agent
- `GET /api/tasks?status={status}` - List tasks by status
- `GET /api/tasks/{id}` - Get single task
- `POST /api/tasks` - Create task
- `PUT /api/tasks/{id}` - Update task (full update)
- `PUT /api/tasks/{id}/status` - Update task status only

#### Contexts
- `GET /api/contexts` - List all contexts
- `GET /api/contexts?project_id={id}` - List contexts for a project
- `GET /api/contexts/{id}` - Get single context
- `POST /api/contexts` - Create context
- `PUT /api/contexts/{id}` - Update context
- `DELETE /api/contexts/{id}` - Delete context

## Testing

To verify the fixes:

1. **Start the backend:**
   ```powershell
   cd c:\Sources\GitHub\agent-shaker
   .\bin\mcp-server.exe
   ```

2. **Start the frontend:**
   ```powershell
   cd c:\Sources\GitHub\agent-shaker\web
   npm run dev
   ```

3. **Test key endpoints:**
   - Navigate to projects page and create a project
   - View project details - should load agents without 404 errors
   - Create an agent for the project
   - View agents list filtered by project
   - Create tasks and assign to agents
   - Update task statuses

## Benefits

1. **Consistency** - Frontend and backend now use the same API contract
2. **Flexibility** - Backend supports multiple filtering strategies
3. **Better Error Handling** - All stores now properly catch, log, and report errors
4. **Complete CRUD** - All stores support full Create, Read, Update, Delete operations
5. **State Management** - Improved synchronization between list views and detail views
6. **Developer Experience** - Clear error messages in console for debugging

## Breaking Changes

None - these are bug fixes to align with the existing backend implementation.

## Next Steps

Consider adding:
1. API documentation generation (Swagger/OpenAPI)
2. Integration tests for all endpoints
3. Frontend validation before API calls
4. Optimistic UI updates with rollback on error
5. Request/response interceptors for auth tokens
