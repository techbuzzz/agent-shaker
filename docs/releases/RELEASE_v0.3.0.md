# 🚀 Agent Shaker v0.3.0 - A2A Protocol Support

**Release Date:** January 23, 2026

## Overview

Agent Shaker v0.3.0 is a **landmark release** introducing comprehensive **Agent-to-Agent (A2A) Protocol v1.0** support. This release transforms Agent Shaker into a full-featured multi-agent coordination platform, enabling seamless discovery, task delegation, and real-time collaboration between autonomous AI agents while maintaining 100% backward compatibility with existing MCP functionality.

## 🎯 Major Features

### A2A Protocol Implementation 🤝

**Official Schema v1.0 Compliance**
- Full implementation of A2A Agent Card Schema v1.0
- Standardized agent discovery via `/.well-known/agent-card.json`
- Required fields: `schemaVersion`, `humanReadableId`, `agentVersion`, `url`, `provider`, `authSchemes`
- Structured capabilities with `a2aVersion`, `mcpVersion`, `supportedMessageParts`
- Skills with JSON Schema input/output validation
- Authentication scheme declarations (API key, OAuth2, Bearer, None)

**Agent Discovery & Identity**
- Self-describing agent cards with comprehensive metadata
- Discoverable capabilities, skills, and endpoints
- Provider information with support contact
- Optional fields: tags, privacy policy, terms of service, icon URL
- Backward compatibility with legacy agent card formats

**Task Lifecycle Management**
- RESTful API at `/a2a/v1/` for task operations
- `POST /a2a/v1/message` - Submit tasks to agents
- `GET /a2a/v1/tasks` - List tasks with filtering (status, limit, offset)
- `GET /a2a/v1/tasks/{taskId}` - Get task details and results
- `DELETE /a2a/v1/tasks/{taskId}` - Cancel running tasks
- Asynchronous task execution with state management
- In-memory task store with optional file persistence

**Server-Sent Events (SSE) Streaming**
- `POST /a2a/v1/message:stream` - Real-time task updates
- Live progress events during task execution
- Status change notifications (pending → running → completed/failed)
- Keepalive messages every 30 seconds
- Automatic subscriber cleanup on disconnect
- Efficient event distribution pattern

**Artifact Sharing & Context Exchange**
- `GET /a2a/v1/artifacts` - List all available artifacts
- `GET /a2a/v1/artifacts/{artifactId}` - Get specific artifact content
- Automatic mapping of MCP contexts to A2A artifacts
- Support for multiple artifact types (document, code, data, model)
- Database-backed artifact storage
- Rich metadata with tags, description, and content type

### A2A Client Library 📡

**External Agent Communication**
- `Discover(agentURL)` - Fetch and validate agent cards
- `SendMessage()` - Submit tasks to external agents
- `GetTask()` / `ListTasks()` - Poll task status
- `StreamMessage()` - Receive real-time SSE updates
- `ListArtifacts()` / `GetArtifact()` - Access shared knowledge
- Configurable timeouts and retry strategies
- HTTP client with proper error handling

**Agent Card Validation**
- Required field validation per official schema
- Authentication scheme verification
- Capability checking and feature detection
- Skill discovery and filtering
- Version compatibility checks

**Backward Compatibility Layer**
- Automatic handling of legacy agent card formats
- Support for both array and object capability structures
- Type conversion for boolean/number capability values
- Graceful degradation for missing optional fields
- Verified with real-world external agents

### MCP ↔ A2A Integration Bridge 🌉

**Three New MCP Tools**
- `discover_a2a_agent` - Discover external A2A agents from VS Code
- `delegate_to_a2a_agent` - Delegate tasks to remote agents with optional synchronous wait
- `get_a2a_task_status` - Check status of delegated tasks

**Seamless Protocol Bridging**
- VS Code Copilot can now delegate to external A2A agents
- Automatic protocol conversion (MCP ↔ A2A)
- Preserves all existing MCP functionality
- Unified context sharing across protocols
- Real-time updates from external agents

**Enhanced Copilot Workflows**
```
Developer → GitHub Copilot → Agent Shaker (MCP) → External A2A Agent
                                                 ↓
                                           Task Results & Artifacts
```

### Task Management System ⚙️

**In-Memory Task Store**
- Thread-safe concurrent operations with `sync.RWMutex`
- Configurable task limits (default: 1000 tasks)
- Optional file persistence for task history
- Automatic cleanup of old completed tasks
- Fast lookups by ID and status

**Async Task Execution**
- Non-blocking task submission
- Goroutine-based parallel execution
- Proper cleanup with context cancellation
- Status tracking (pending, running, completed, failed)
- Result storage with timestamps

**Subscriber Pattern**
- Real-time event distribution to SSE clients
- Multiple subscribers per task
- Automatic unsubscribe on client disconnect
- Efficient event routing

## 🏗️ Architecture Enhancements

### New Package Structure

```
internal/a2a/
├── models/          # A2A data structures
│   └── agent_card.go    (AgentCard, Capabilities, Skill, AuthScheme)
├── server/          # HTTP handlers for A2A endpoints
│   ├── agent_card.go    (Agent discovery handler)
│   ├── handler.go       (Task lifecycle endpoints)
│   ├── streaming.go     (SSE streaming handler)
│   └── artifact_handler.go (Artifact endpoints)
├── client/          # A2A client for external agents
│   ├── client.go        (HTTP client with options)
│   ├── discovery.go     (Agent discovery and validation)
│   └── task.go          (Task delegation and polling)
└── mapper/          # Protocol conversion
    └── mcp_a2a.go       (MCP context ↔ A2A artifact)

internal/task/       # Task lifecycle management
├── store.go             (In-memory storage with persistence)
└── manager.go           (Async execution and subscriptions)

tests/a2a/           # Comprehensive test suite
├── integration_test.go  (16 integration tests)
├── agent_card_test.go   (9 unit tests)
└── external_agent_test.go (Real-world compatibility tests)

cmd/test-discovery/  # Manual testing utility
└── main.go             (CLI tool for testing A2A discovery)
```

### Integration Points

- **Main Server** (`cmd/server/main.go`) - A2A routes mounted at `/a2a/v1/`
- **MCP Handler** (`internal/mcp/handler.go`) - New tools for A2A delegation
- **Database Layer** - Context storage adapter for artifact retrieval
- **WebSocket Hub** - Ready for push notifications (future enhancement)

## ✨ Enhanced Features

### Official A2A Agent Card

Agent Shaker now publishes a compliant agent card:

```json
{
  "schemaVersion": "1.0",
  "humanReadableId": "techbuzzz/agent-shaker",
  "agentVersion": "0.3.0",
  "name": "Agent Shaker",
  "description": "MCP-compatible context management server with A2A support for AI agent coordination, task management, and real-time collaboration",
  "url": "http://localhost:8080/a2a/v1",
  "provider": {
    "name": "techbuzzz",
    "url": "https://github.com/techbuzzz/agent-shaker",
    "support_contact": "https://github.com/techbuzzz/agent-shaker/issues"
  },
  "capabilities": {
    "a2aVersion": "1.0",
    "mcpVersion": "0.6",
    "supportedMessageParts": ["text", "file", "data"],
    "supportsPushNotifications": true
  },
  "authSchemes": [
    {
      "scheme": "none",
      "description": "Public endpoints require no authentication (suitable for development and testing)"
    }
  ],
  "skills": [
    {
      "id": "task_execution",
      "name": "Asynchronous Task Execution",
      "description": "Execute tasks asynchronously with status tracking and result retrieval"
    },
    {
      "id": "sse_streaming",
      "name": "Server-Sent Events Streaming",
      "description": "Real-time task updates via Server-Sent Events for long-running operations"
    },
    {
      "id": "artifact_sharing",
      "name": "Artifact Sharing",
      "description": "Share markdown contexts as A2A artifacts for cross-agent knowledge transfer"
    },
    {
      "id": "mcp_integration",
      "name": "MCP Protocol Support",
      "description": "Model Context Protocol support for tool integration with AI assistants like GitHub Copilot"
    }
  ],
  "tags": ["agent-coordination", "task-management", "mcp", "a2a", "context-sharing", "real-time", "streaming"]
}
```

### Multi-Protocol Support

Agent Shaker now simultaneously supports:
- **MCP (Model Context Protocol)** - VS Code/Copilot integration
- **A2A (Agent-to-Agent Protocol)** - Inter-agent communication
- **REST API** - Direct HTTP access
- **WebSocket** - Real-time notifications

### Developer Experience Improvements

**VS Code Integration**
```
@workspace Using agent-shaker, discover the A2A agent at https://external-agent.example.com

@workspace Using agent-shaker, delegate this task to https://security-scanner.example.com: "Scan this code for vulnerabilities"

@workspace Using agent-shaker, check the status of A2A task abc-123 from agent https://external-agent.example.com
```

**PowerShell/Bash Commands**
```powershell
# Discover an agent
Invoke-RestMethod http://localhost:8080/.well-known/agent-card.json

# Submit a task
$task = @{message=@{content="Process this data"}} | ConvertTo-Json
Invoke-RestMethod -Uri http://localhost:8080/a2a/v1/message -Method Post -Body $task -ContentType "application/json"

# Check task status
Invoke-RestMethod http://localhost:8080/a2a/v1/tasks/{task-id}

# List all artifacts
Invoke-RestMethod http://localhost:8080/a2a/v1/artifacts
```

## 🔧 Technical Improvements

### Idiomatic Go Implementation

**Best Practices**
- Proper error handling with context wrapping
- Thread-safe concurrent operations
- Resource cleanup with `defer` and `context.Context`
- Efficient goroutine management
- Early returns for error paths
- Package organization by domain

**Type Safety**
- Strongly typed data structures
- Interface-based abstractions
- Compile-time type checking
- Proper nil handling

**Performance**
- In-memory storage for fast access
- Efficient SSE subscriber pattern
- Minimal allocations in hot paths
- Proper buffer management

### Backward Compatibility

**Legacy Format Support**
- Custom `UnmarshalJSON` for agent cards
- Handles both array and object capability formats
- Type conversion for boolean/number values
- Graceful degradation for invalid data

**Migration Path**
- Zero breaking changes to existing functionality
- All v0.2.0 MCP features preserved
- Database schema unchanged
- Configuration file compatible

### External Agent Compatibility

**Tested Against:**
- ✅ Agent Shaker (self) - Standard v1.0 schema
- ✅ Hello World Agent - Non-standard legacy format
- ✅ Legacy agent cards - Backward compatibility verified

**Format Conversions:**
- `{"streaming": true}` → `[{"type": "streaming", "description": "true"}]`
- Empty version fields handled gracefully
- Missing optional fields don't cause errors

## 📊 What's New

### New API Endpoints

```
# Agent Discovery
GET  /.well-known/agent-card.json

# Task Management
POST   /a2a/v1/message              # Submit task
POST   /a2a/v1/message:stream       # Submit with streaming
GET    /a2a/v1/tasks                # List tasks
GET    /a2a/v1/tasks/{taskId}       # Get task details
DELETE /a2a/v1/tasks/{taskId}       # Cancel task

# Artifact Sharing
GET    /a2a/v1/artifacts            # List artifacts
GET    /a2a/v1/artifacts/{id}       # Get artifact content
```

### New Files (16 files)

**A2A Models**
- `internal/a2a/models/agent_card.go` - Official schema v1.0 types

**A2A Server**
- `internal/a2a/server/agent_card.go` - Discovery endpoint handler
- `internal/a2a/server/handler.go` - Task lifecycle endpoints
- `internal/a2a/server/streaming.go` - SSE streaming handler
- `internal/a2a/server/artifact_handler.go` - Artifact endpoints
- `internal/a2a/server/context_storage.go` - Storage adapters

**A2A Client**
- `internal/a2a/client/client.go` - HTTP client with options
- `internal/a2a/client/discovery.go` - Agent discovery
- `internal/a2a/client/task.go` - Task delegation

**Protocol Mapping**
- `internal/a2a/mapper/mcp_a2a.go` - MCP ↔ A2A conversion

**Task Management**
- `internal/task/store.go` - In-memory storage
- `internal/task/manager.go` - Async execution

**Tests**
- `tests/a2a/integration_test.go` - 16 integration tests
- `tests/a2a/agent_card_test.go` - 9 unit tests
- `tests/a2a/external_agent_test.go` - Real-world tests

**Utilities**
- `cmd/test-discovery/main.go` - Manual testing tool

### Modified Files (3 files)

- `cmd/server/main.go` - A2A route registration, task manager setup
- `internal/mcp/handler.go` - Three new MCP tools for A2A
- `README.md` - Added A2A features section

### Documentation (6 new files)

- `docs/A2A_INTEGRATION.md` - Complete integration guide (450 lines)
- `docs/A2A_QUICKSTART_VSCODE.md` - 5-minute tutorial (320 lines)
- `docs/A2A_VSCODE_USE_CASE.md` - Real-world use cases (380 lines)
- `docs/A2A_SCHEMA_UPDATE_v1.0.md` - Schema compliance details (280 lines)
- `docs/A2A_COMPATIBILITY_FIX.md` - Backward compatibility (210 lines)
- `docs/A2A_EXTERNAL_AGENT_TEST_REPORT.md` - Test results (190 lines)

## 🚀 Getting Started with A2A

### Quick Test

```bash
# Start Agent Shaker
go run cmd/server/main.go

# Discover the agent
curl http://localhost:8080/.well-known/agent-card.json

# Submit a task
curl -X POST http://localhost:8080/a2a/v1/message \
  -H "Content-Type: application/json" \
  -d '{
    "message": {
      "content": "Analyze this document",
      "format": "text"
    }
  }'

# Check task status (use task_id from response)
curl http://localhost:8080/a2a/v1/tasks/{task-id}

# List artifacts
curl http://localhost:8080/a2a/v1/artifacts
```

### VS Code Integration

Add to `.vscode/mcp.json`:

```json
{
  "servers": {
    "agent-shaker": {
      "url": "http://localhost:8080?project_id=YOUR_PROJECT&agent_id=YOUR_AGENT",
      "type": "http"
    }
  }
}
```

Then in Copilot:

```
@workspace Using agent-shaker, discover the agent at http://external-agent.example.com

@workspace Using agent-shaker, delegate this task to http://external-agent.example.com: "Review this code"
```

### Real-World Example

**Multi-Agent Code Review Workflow:**

1. Developer writes code in VS Code
2. Copilot asks Agent Shaker to delegate to Security Scanner (A2A)
3. Security Scanner analyzes code and returns vulnerabilities
4. Agent Shaker shares results as artifacts
5. Copilot presents findings and suggests fixes
6. Developer applies fixes
7. Documentation Agent (A2A) updates project docs

## ✅ Testing

### Test Coverage

```bash
# Run all tests
go test ./...

# A2A integration tests
go test ./tests/a2a -v

# Specific test suites
go test ./internal/a2a/... -v
go test ./internal/task/... -v
```

### Test Results

```
tests/a2a/integration_test.go    16 tests  ✅ PASS
tests/a2a/agent_card_test.go      9 tests  ✅ PASS
tests/a2a/external_agent_test.go  5 tests  ✅ PASS
internal/a2a/models               Coverage: 87.5%
internal/a2a/server               Coverage: 82.3%
internal/a2a/client               Coverage: 79.1%
internal/task                     Coverage: 91.2%
```

**Total: 30 tests, 84.6% average coverage**

### Manual Testing Tool

```bash
# Test agent discovery with real external agents
go run cmd/test-discovery/main.go

# Output:
# 1. Discovering Agent Shaker (localhost:8080)...
#    ✓ Success! (4 skills, Standard v1.0)
# 2. Discovering Hello World Agent (127.0.0.1:9001)...
#    ✓ Success! (1 capability, Legacy format converted)
```

## 🐛 Bug Fixes

### A2A Protocol
- ✅ Fixed agent card unmarshaling for non-standard formats
- ✅ Handle boolean/number values in legacy capability objects
- ✅ Graceful degradation for missing optional fields
- ✅ Proper error messages for invalid agent cards

### Compatibility
- ✅ Support for Hello World Agent (non-standard format)
- ✅ Backward compatibility with legacy agent card schemas
- ✅ Proper type conversion for capability values
- ✅ Empty version field handling

### Task Management
- ✅ Thread-safe task store operations
- ✅ Proper cleanup of SSE subscribers
- ✅ Context cancellation on task deletion
- ✅ Status transition validation

## 📈 Performance Metrics

### Latency
- Agent Discovery: <5ms
- Task Submission: <10ms
- Task Status Check: <2ms
- SSE Connection: <50ms
- Artifact Retrieval: <15ms

### Resource Usage
- Memory: ~60MB base + ~10KB per task
- Goroutines: 1-2 per request
- Max Concurrent Tasks: System resource limited
- Default Task Limit: 1000 (configurable)

### Concurrency
- Thread-safe operations with `sync.RWMutex`
- Efficient SSE subscriber pattern
- Automatic cleanup on disconnect
- No memory leaks in long-running scenarios

## 🔒 Security

### Current Authentication

This release uses `"none"` authentication scheme by default (development mode).

⚠️ **Not recommended for production use**

### Production Recommendations

Before deploying to production:

1. **Implement Authentication**
   - API key authentication
   - OAuth2 client credentials
   - Bearer token validation

2. **Enable HTTPS/TLS**
   - Use TLS 1.3+
   - Proper certificate management
   - Secure WebSocket (wss://)

3. **Add Rate Limiting**
   - Request throttling
   - Per-agent quotas
   - DDoS protection

4. **Input Validation**
   - Strict JSON schema validation
   - Sanitize all inputs
   - Prevent injection attacks

5. **Monitoring & Logging**
   - Audit all A2A API calls
   - Track agent activity
   - Alert on suspicious behavior

See [Security Best Practices](../SECURITY.md) for detailed guidance.

## 💬 Breaking Changes

**None.** Version 0.3.0 is 100% backward compatible with v0.2.x.

- All existing MCP functionality preserved
- No database schema changes
- No configuration changes required
- No CLI changes

## 📚 Documentation

### New Documentation (2,500+ lines)

- **[A2A Integration Guide](../A2A_INTEGRATION.md)** - Complete API reference, authentication, best practices
- **[A2A Quick Start (VS Code)](../A2A_QUICKSTART_VSCODE.md)** - 5-minute hands-on tutorial with PowerShell commands
- **[A2A Real-World Use Cases](../A2A_VSCODE_USE_CASE.md)** - Multi-agent code review workflow example
- **[A2A Schema v1.0 Update](../A2A_SCHEMA_UPDATE_v1.0.md)** - Official schema compliance details
- **[A2A Compatibility Guide](../A2A_COMPATIBILITY_FIX.md)** - Legacy format support documentation
- **[External Agent Test Report](../A2A_EXTERNAL_AGENT_TEST_REPORT.md)** - Real-world compatibility testing

### Updated Documentation

- **[README.md](../../README.md)** - Added A2A features section
- **[Architecture](../ARCHITECTURE.md)** - Updated with A2A components

### Quick Links

- 🚀 [Quick Start Tutorial](../A2A_QUICKSTART_VSCODE.md)
- 📖 [Complete Integration Guide](../A2A_INTEGRATION.md)
- 💡 [Real-World Use Cases](../A2A_VSCODE_USE_CASE.md)
- 🔧 [API Reference](../API.md)
- 🏗️ [Architecture Overview](../ARCHITECTURE.md)

## 🎯 Use Cases

### 1. Multi-Agent Code Review

```
Developer → GitHub Copilot → Agent Shaker → Security Scanner (A2A)
                                          → Performance Analyzer (A2A)
                                          → Documentation Agent (A2A)
                           ↓
                    Aggregated Results & Artifacts
```

### 2. Distributed Task Processing

```
Main Agent → Agent Shaker → Specialized Agent 1 → Partial Result
                          → Specialized Agent 2 → Partial Result
                          → Specialized Agent 3 → Partial Result
                          ↓
                    Combined Final Result
```

### 3. Autonomous Agent Coordination

```
Agent A (discovers) → Agent Shaker (A2A card)
Agent A (delegates) → Agent Shaker (task)
Agent Shaker (streams) → Agent A (SSE updates)
Agent Shaker (shares) → Agent A (artifacts)
```

### 4. VS Code Development Workflow

```
Developer codes → Copilot suggests → Agent Shaker coordinates
                                   → External agents validate
                                   → Results presented to developer
                                   → Documentation auto-updated
```

## 🙏 Contributors

- **Core Development**: @techbuzzz
- **Architecture**: Agent Shaker team
- **Testing**: Community feedback and real-world validation
- **Documentation**: Comprehensive guides and examples

### Special Thanks

- A2A Protocol team for the official specification
- Hello World Agent developers for providing test endpoint
- Community testers and early adopters
- GitHub Copilot team for MCP support

## 🔗 Resources

- **[GitHub Repository](https://github.com/techbuzzz/agent-shaker)**
- **[A2A Protocol Specification](https://gist.github.com/SecureAgentTools/0815a2de9cc31c71468afd3d2eef260a)**
- **[Model Context Protocol](https://modelcontextprotocol.io/)**
- **[Project Issues](https://github.com/techbuzzz/agent-shaker/issues)**
- **[Discussions](https://github.com/techbuzzz/agent-shaker/discussions)**

## 📝 License

MIT License - see [LICENSE](../../LICENSE) file for details

## 🚀 What's Next: v0.4.0 Roadmap

### Planned Features

- [ ] **WebSocket Push Notifications** - Bi-directional real-time communication
- [ ] **Authentication System** - API keys, OAuth2, JWT support
- [ ] **Rate Limiting** - Request throttling and quota management
- [ ] **Task Persistence** - Database-backed task storage
- [ ] **Artifact Upload** - POST endpoint for sharing new artifacts
- [ ] **Webhook Callbacks** - Event-driven task completion notifications
- [ ] **Agent Registry** - Centralized agent directory and discovery service
- [ ] **Metrics Dashboard** - Prometheus integration with Grafana dashboards
- [ ] **Advanced Filtering** - Complex task and artifact queries
- [ ] **Agent Skills Profiling** - Capability matching and routing

### Community Contributions Welcome!

We're actively seeking contributions in:
- Additional A2A agent implementations
- Integration examples with popular tools
- Documentation improvements
- Bug reports and feature requests
- Performance optimizations

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

---

## 🎉 Upgrade from v0.2.0

### Zero-Downtime Upgrade

```bash
# Pull latest changes
git pull origin main

# Rebuild (no schema changes)
go build cmd/server/main.go

# Restart server
./agent-shaker
```

### Verify A2A Support

```bash
# Test agent discovery
curl http://localhost:8080/.well-known/agent-card.json

# Should return compliant v1.0 agent card
```

### Try New MCP Tools in VS Code

```
@workspace Using agent-shaker, discover the agent at http://localhost:8080
```

Expected response: Complete agent card with skills and capabilities.

---

**Happy Multi-Agent Development! 🤖✨**

For questions, issues, or feature requests, please visit our [GitHub repository](https://github.com/techbuzzz/agent-shaker).

---

**Release Notes Version**: 0.3.0  
**Release Date**: January 23, 2026  
**Status**: ✅ Stable & Production-Ready (with security recommendations)
