# MCP Task Tracker

MCP Task Tracker координирует работу AI агентов между backend и frontend проектами. Система обеспечивает координацию задач, real-time синхронизацию через WebSocket и управление документацией для команд агентов.

## 🚀 Features

- **Project Management**: Создание и управление проектами
- **Agent Registration**: Регистрация агентов с ролями (backend/frontend)
- **Task Assignment**: Назначение и получение задач через REST API
- **Status Tracking**: Отслеживание статусов выполнения задач
- **Documentation**: Добавление markdown документации после выполнения задач
- **Real-time Sync**: WebSocket для синхронизации обновлений между командами
- **Docker Support**: Простое развертывание через docker-compose

## 📋 Prerequisites

- Docker и Docker Compose
- (Опционально) Go 1.21+ для локальной разработки

## 🔧 Installation & Setup

### Запуск с Docker Compose

1. Клонируйте репозиторий:
```bash
git clone https://github.com/techbuzzz/agent-shaker.git
cd agent-shaker
```

2. Запустите сервер:
```bash
docker-compose up -d
```

3. Проверьте статус:
```bash
docker-compose ps
curl http://localhost:8080/health
```

### Локальная разработка

```bash
# Установите зависимости
go mod download

# Создайте директорию для данных
mkdir -p data

# Запустите сервер
go run main.go
```

## 📚 API Documentation

Base URL: `http://localhost:8080/api/v1`

### Projects

#### Создать проект
```bash
POST /api/v1/projects
Content-Type: application/json

{
  "name": "My Microservice Project",
  "description": "Backend and frontend coordination"
}
```

#### Получить список проектов
```bash
GET /api/v1/projects
```

#### Получить проект
```bash
GET /api/v1/projects/{id}
```

### Agents

#### Зарегистрировать агента
```bash
POST /api/v1/agents
Content-Type: application/json

{
  "project_id": "project-uuid",
  "name": "Backend Agent 1",
  "role": "backend"
}
```

Доступные роли: `backend`, `frontend`

#### Получить список агентов проекта
```bash
GET /api/v1/projects/{project_id}/agents
```

#### Получить агента
```bash
GET /api/v1/agents/{id}
```

### Tasks

#### Создать задачу
```bash
POST /api/v1/tasks
Content-Type: application/json

{
  "project_id": "project-uuid",
  "agent_id": "agent-uuid",
  "title": "Implement user authentication",
  "description": "Add JWT-based authentication to the API",
  "priority": 5
}
```

#### Получить задачу
```bash
GET /api/v1/tasks/{id}
```

#### Обновить статус задачи
```bash
PUT /api/v1/tasks/{id}/status
Content-Type: application/json

{
  "status": "in_progress",
  "message": "Started working on authentication"
}
```

Доступные статусы: `pending`, `in_progress`, `completed`, `failed`

#### Получить задачи агента
```bash
GET /api/v1/agents/{agent_id}/tasks
```

#### Получить задачи проекта
```bash
GET /api/v1/projects/{project_id}/tasks
```

### Documentation

#### Добавить документацию
```bash
POST /api/v1/documentation
Content-Type: application/json

{
  "task_id": "task-uuid",
  "content": "# Authentication Implementation\n\n## Overview\n...",
  "created_by": "agent-uuid"
}
```

#### Получить документацию задачи
```bash
GET /api/v1/tasks/{task_id}/documentation
```

### WebSocket

Подключение к WebSocket для real-time обновлений:

```javascript
const ws = new WebSocket('ws://localhost:8080/ws?project_id=your-project-id');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Update received:', data);
  // { type: 'task_update', data: { task_id, agent_id, status, message, timestamp } }
};
```

## 💡 Usage Example

Полный пример координации агентов:

```bash
# 1. Создать проект
PROJECT_ID=$(curl -s -X POST http://localhost:8080/api/v1/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"Microservice App","description":"Backend + Frontend"}' \
  | jq -r '.id')

echo "Project created: $PROJECT_ID"

# 2. Зарегистрировать backend агента
BACKEND_AGENT=$(curl -s -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"$PROJECT_ID\",\"name\":\"Backend Agent\",\"role\":\"backend\"}" \
  | jq -r '.id')

echo "Backend agent: $BACKEND_AGENT"

# 3. Зарегистрировать frontend агента
FRONTEND_AGENT=$(curl -s -X POST http://localhost:8080/api/v1/agents \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"$PROJECT_ID\",\"name\":\"Frontend Agent\",\"role\":\"frontend\"}" \
  | jq -r '.id')

echo "Frontend agent: $FRONTEND_AGENT"

# 4. Создать задачу для backend агента
TASK_ID=$(curl -s -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -d "{\"project_id\":\"$PROJECT_ID\",\"agent_id\":\"$BACKEND_AGENT\",\"title\":\"API Implementation\",\"description\":\"Build REST API\",\"priority\":10}" \
  | jq -r '.id')

echo "Task created: $TASK_ID"

# 5. Агент получает свои задачи
curl -s http://localhost:8080/api/v1/agents/$BACKEND_AGENT/tasks | jq

# 6. Обновить статус задачи
curl -X PUT http://localhost:8080/api/v1/tasks/$TASK_ID/status \
  -H "Content-Type: application/json" \
  -d '{"status":"in_progress","message":"Started implementation"}'

# 7. Завершить задачу и добавить документацию
curl -X PUT http://localhost:8080/api/v1/tasks/$TASK_ID/status \
  -H "Content-Type: application/json" \
  -d '{"status":"completed","message":"API completed"}'

curl -X POST http://localhost:8080/api/v1/documentation \
  -H "Content-Type: application/json" \
  -d "{\"task_id\":\"$TASK_ID\",\"content\":\"# API Implementation\\n\\n## Endpoints\\n- POST /api/users\\n- GET /api/users/:id\",\"created_by\":\"$BACKEND_AGENT\"}"

# 8. Получить документацию
curl -s http://localhost:8080/api/v1/tasks/$TASK_ID/documentation | jq
```

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    MCP Task Tracker                      │
├─────────────────────────────────────────────────────────┤
│                                                           │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐ │
│  │   REST API  │    │  WebSocket  │    │   Database  │ │
│  │             │    │     Hub     │    │   SQLite    │ │
│  └─────────────┘    └─────────────┘    └─────────────┘ │
│         │                   │                   │        │
│         └───────────────────┴───────────────────┘        │
│                           │                              │
└───────────────────────────┼──────────────────────────────┘
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
    ┌────▼────┐        ┌────▼────┐       ┌────▼────┐
    │ Backend │        │ Backend │       │Frontend │
    │ Agent 1 │        │ Agent 2 │       │ Agent 1 │
    └─────────┘        └─────────┘       └─────────┘
```

## 🛠️ Technology Stack

- **Backend**: Go 1.21+
- **Database**: SQLite
- **WebSocket**: Gorilla WebSocket
- **Router**: Gorilla Mux
- **Container**: Docker & Docker Compose

## 🔍 Project Structure

```
agent-shaker/
├── main.go                 # Application entry point
├── internal/
│   ├── models/            # Data models
│   │   └── models.go
│   ├── db/                # Database layer
│   │   └── db.go
│   ├── api/               # REST API handlers
│   │   └── api.go
│   └── websocket/         # WebSocket implementation
│       └── websocket.go
├── Dockerfile             # Docker image configuration
├── docker-compose.yml     # Docker Compose configuration
├── go.mod                 # Go module definition
└── README.md             # This file
```

## 🤝 Integration with GitHub Copilot

MCP Task Tracker идеально подходит для координации нескольких экземпляров GitHub Copilot в микросервисной архитектуре:

1. **Регистрация агентов**: Каждый экземпляр Copilot регистрируется как агент с определенной ролью
2. **Получение задач**: Агенты получают задачи через API в соответствии со своей ролью
3. **Обновление статуса**: Real-time обновления статуса выполнения задач
4. **Документация**: Автоматическое создание markdown документации после выполнения
5. **Синхронизация**: WebSocket обеспечивает мгновенную синхронизацию между командами

## 📝 License

MIT License - see LICENSE file for details

## 🤖 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.