#!/bin/bash

# MCP Task Tracker Demo Script
# This script demonstrates the complete workflow of the MCP Task Tracker

set -e

BASE_URL="http://localhost:8080"

echo "=== MCP Task Tracker Demo ==="
echo

# 1. Create a project
echo "1. Creating a project..."
PROJECT_RESPONSE=$(curl -s -X POST $BASE_URL/api/v1/projects \
  -H "Content-Type: application/json" \
  -d '{
    "name": "E-Commerce Platform",
    "description": "Microservice architecture with backend and frontend teams"
  }')

PROJECT_ID=$(echo "$PROJECT_RESPONSE" | jq -r '.id')
echo "✓ Project created: $PROJECT_ID"
echo

# 2. Register agents
echo "2. Registering agents..."
BACKEND_AGENT=$(curl -s -X POST $BASE_URL/api/v1/agents \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": \"$PROJECT_ID\",
    \"name\": \"Backend Agent - API Team\",
    \"role\": \"backend\"
  }" | jq -r '.id')

FRONTEND_AGENT=$(curl -s -X POST $BASE_URL/api/v1/agents \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": \"$PROJECT_ID\",
    \"name\": \"Frontend Agent - UI Team\",
    \"role\": \"frontend\"
  }" | jq -r '.id')

echo "✓ Backend Agent: $BACKEND_AGENT"
echo "✓ Frontend Agent: $FRONTEND_AGENT"
echo

# 3. Create tasks
echo "3. Creating tasks..."
BACKEND_TASK=$(curl -s -X POST $BASE_URL/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": \"$PROJECT_ID\",
    \"agent_id\": \"$BACKEND_AGENT\",
    \"title\": \"Implement User Authentication API\",
    \"description\": \"Build JWT-based authentication endpoints\",
    \"priority\": 10
  }" | jq -r '.id')

FRONTEND_TASK=$(curl -s -X POST $BASE_URL/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d "{
    \"project_id\": \"$PROJECT_ID\",
    \"agent_id\": \"$FRONTEND_AGENT\",
    \"title\": \"Create Login UI\",
    \"description\": \"Design and implement login page with React\",
    \"priority\": 9
  }" | jq -r '.id')

echo "✓ Backend Task: $BACKEND_TASK"
echo "✓ Frontend Task: $FRONTEND_TASK"
echo

# 4. Backend agent starts working
echo "4. Backend agent picks up task and starts working..."
curl -s -X PUT $BASE_URL/api/v1/tasks/$BACKEND_TASK/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in_progress",
    "message": "Started implementing authentication endpoints"
  }' > /dev/null

echo "✓ Status updated to: in_progress"
echo

# 5. Backend agent completes task
sleep 1
echo "5. Backend agent completes the task..."
curl -s -X PUT $BASE_URL/api/v1/tasks/$BACKEND_TASK/status \
  -H "Content-Type: application/json" \
  -d '{
    "status": "completed",
    "message": "Authentication API ready"
  }' > /dev/null

echo "✓ Status updated to: completed"
echo

# 6. Add documentation
echo "6. Adding documentation..."
curl -s -X POST $BASE_URL/api/v1/documentation \
  -H "Content-Type: application/json" \
  -d "{
    \"task_id\": \"$BACKEND_TASK\",
    \"content\": \"# Authentication API\n\n## Endpoints\n\n### POST /api/auth/login\nLogin with credentials\n\n### POST /api/auth/register\nRegister new user\n\n### POST /api/auth/refresh\nRefresh JWT token\n\n## Usage\n\`\`\`bash\ncurl -X POST http://api/auth/login \\\\\n  -H 'Content-Type: application/json' \\\\\n  -d '{\\\"email\\\":\\\"user@example.com\\\",\\\"password\\\":\\\"pass123\\\"}'\n\`\`\`\",
    \"created_by\": \"$BACKEND_AGENT\"
  }" > /dev/null

echo "✓ Documentation added"
echo

# 7. Display project status
echo "7. Current project status:"
echo
echo "Project Tasks:"
curl -s $BASE_URL/api/v1/projects/$PROJECT_ID/tasks | jq '.[] | {title, status, priority}'
echo

echo "Backend Agent Tasks:"
curl -s $BASE_URL/api/v1/agents/$BACKEND_AGENT/tasks | jq '.[] | {title, status}'
echo

echo "Documentation:"
curl -s $BASE_URL/api/v1/tasks/$BACKEND_TASK/documentation | jq '.[0].content' -r
echo

echo "=== Demo Complete ==="
