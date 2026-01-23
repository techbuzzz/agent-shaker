# Real-World A2A Use Case: Multi-Agent Code Review System in VS Code

## Scenario Overview

In this use case, we have multiple AI agents working together in VS Code to perform a comprehensive code review workflow. Agent Shaker coordinates the agents using the A2A Protocol.

### Agents Involved

1. **Code Analysis Agent** (Agent Shaker) - Analyzes code structure and complexity
2. **Security Scanner Agent** (External A2A) - Checks for security vulnerabilities
3. **Performance Analyzer Agent** (External A2A) - Identifies performance bottlenecks
4. **Documentation Agent** (GitHub Copilot via MCP) - Reviews and suggests documentation improvements

## Setup

### Prerequisites

1. **Agent Shaker** running locally on `http://localhost:8080`
2. **VS Code** with GitHub Copilot installed
3. **External A2A agents** (simulated or real)

### Step 1: Configure Agent Shaker in VS Code

Add to your VS Code `settings.json`:

```json
{
  "github.copilot.advanced": {
    "mcp": {
      "servers": {
        "agent-shaker": {
          "command": "node",
          "args": ["path/to/agent-shaker/mcp-bridge.js"],
          "env": {
            "MCP_SERVER_URL": "http://localhost:8080"
          }
        }
      }
    }
  }
}
```

### Step 2: Start Agent Shaker

```bash
cd agent-shaker
go run cmd/server/main.go
```

Output:
```
2026/01/23 10:00:00 Connected to database
2026/01/23 10:00:00 Server starting on port 8080
Agent Shaker - Multi-Protocol AI Agent Platform
Endpoints:
  A2A Discovery: http://localhost:8080/.well-known/agent-card.json
  A2A API:       http://localhost:8080/a2a/v1
  MCP:           http://localhost:8080/ (Protocol endpoint)
  REST API:      http://localhost:8080/api
  WebSocket:     ws://localhost:8080/ws
```

## Real-World Workflow

### Use Case: Complete Code Review for a New Feature

Let's say you've just completed implementing a new user authentication feature in Go and need a comprehensive review before merging.

#### Step 1: Create Project and Register Agents in VS Code

Open VS Code and use GitHub Copilot Chat with Agent Shaker MCP tools:

**Prompt to Copilot:**
```
@workspace Using the agent-shaker MCP server, create a new project called "User Auth Feature Review" and register me as a backend agent
```

**What happens behind the scenes:**

Copilot uses MCP tools:
```json
// Tool: create_project (MCP)
{
  "name": "User Auth Feature Review",
  "description": "Code review for JWT authentication implementation"
}
```

Response:
```json
{
  "project_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "User Auth Feature Review",
  "status": "active"
}
```

#### Step 2: Discover External A2A Security Scanner

**Prompt to Copilot:**
```
@workspace Using agent-shaker, discover the security scanner agent at https://security-scanner.example.com and tell me what it can do
```

**Copilot executes:**
```json
// Tool: discover_a2a_agent (MCP → A2A)
{
  "agent_url": "https://security-scanner.example.com"
}
```

**Response:**
```json
{
  "success": true,
  "name": "SecureCode Scanner",
  "version": "2.1.0",
  "capabilities": [
    {"type": "task", "description": "Security vulnerability scanning"},
    {"type": "streaming", "description": "Real-time scan progress"},
    {"type": "artifacts", "description": "Detailed security reports"}
  ],
  "endpoints": [
    {"path": "/a2a/v1/message", "method": "POST", "protocol": "A2A"}
  ]
}
```

**Copilot summarizes:**
> "I found SecureCode Scanner v2.1.0 at the specified URL. It can:
> - Scan code for security vulnerabilities
> - Provide real-time scanning progress via streaming
> - Generate detailed security reports as artifacts
> 
> The agent is ready to receive tasks via the A2A protocol."

#### Step 3: Share Your Code Context with Team

**Prompt to Copilot:**
```
@workspace Using agent-shaker, add context about the JWT authentication implementation I just completed. Use the code in auth/ directory.
```

**Copilot executes:**
```json
// Tool: add_context (MCP)
{
  "project_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "title": "JWT Authentication Implementation - Complete",
  "content": "# JWT Authentication Implementation\n\n## Overview\nImplemented RS256-based JWT authentication with refresh token support.\n\n## Key Files\n- `auth/jwt.go` - Token generation and validation\n- `auth/middleware.go` - Authentication middleware\n- `auth/refresh.go` - Refresh token rotation\n\n## Features\n✅ RS256 signing algorithm\n✅ Refresh tokens (7-day expiry)\n✅ Access tokens (15-minute expiry)\n✅ Token rotation on refresh\n✅ Blacklist support\n\n## Endpoints\n\n### POST /api/auth/login\n```json\n{\n  \"email\": \"user@example.com\",\n  \"password\": \"password\"\n}\n```\n\n### POST /api/auth/refresh\n```json\n{\n  \"refresh_token\": \"eyJhbG...\"\n}\n```\n\n## Security Considerations\n- Private keys stored in secure vault\n- Tokens include user ID and roles\n- Rate limiting on auth endpoints\n\n## Testing Status\n✅ Unit tests: 95% coverage\n✅ Integration tests passed\n⏳ Security audit pending\n⏳ Performance testing pending",
  "tags": ["authentication", "jwt", "security", "go"]
}
```

**Response:**
> "I've added your JWT authentication implementation details to the shared project context. Other agents can now access this documentation to understand your work. The context includes implementation details, endpoints, security considerations, and testing status."

#### Step 4: Delegate Security Review to External Agent

**Prompt to Copilot:**
```
@workspace Using agent-shaker, delegate a security review task to the SecureCode Scanner agent. Ask it to analyze the JWT implementation for vulnerabilities. Wait for the results.
```

**Copilot executes:**
```json
// Tool: delegate_to_a2a_agent (MCP → A2A)
{
  "agent_url": "https://security-scanner.example.com",
  "message": "Perform a comprehensive security analysis of the JWT authentication implementation. Focus on: token generation, signing algorithms, storage practices, and common JWT vulnerabilities (alg:none, weak keys, token expiration). The implementation uses RS256 with 15-minute access tokens and 7-day refresh tokens.",
  "wait_for_completion": true,
  "timeout_seconds": 300
}
```

**Real-time SSE updates received:**
```
event: task_created
data: {"task_id": "sec-12345", "status": "pending"}

event: status
data: {"task_id": "sec-12345", "status": "running", "progress": "Analyzing token generation..."}

event: status
data: {"task_id": "sec-12345", "status": "running", "progress": "Checking signing algorithms..."}

event: status
data: {"task_id": "sec-12345", "status": "running", "progress": "Scanning for vulnerabilities..."}

event: completed
data: {
  "task_id": "sec-12345",
  "status": "completed",
  "result": {
    "content": "Security Analysis Complete",
    "format": "markdown",
    "data": {
      "vulnerabilities_found": 2,
      "severity": "medium",
      "issues": [
        {
          "type": "Missing Rate Limiting Header",
          "severity": "medium",
          "file": "auth/middleware.go",
          "line": 45,
          "description": "Response headers should include X-RateLimit-* headers",
          "recommendation": "Add rate limit info to response headers"
        },
        {
          "type": "Token Storage",
          "severity": "low",
          "file": "auth/jwt.go",
          "line": 89,
          "description": "Consider implementing automatic token rotation",
          "recommendation": "Implement automatic rotation for long-lived refresh tokens"
        }
      ],
      "strengths": [
        "RS256 algorithm correctly implemented",
        "Appropriate token expiration times",
        "Private key management follows best practices",
        "Token blacklist implemented"
      ]
    }
  }
}
```

**Copilot summarizes the results in chat:**
> "🔒 Security Review Complete
> 
> **Overall Status:** ✅ Mostly Secure (2 medium-priority improvements recommended)
> 
> **Issues Found:**
> 
> 1. **Missing Rate Limiting Header** (Medium Priority)
>    - Location: `auth/middleware.go:45`
>    - Fix: Add X-RateLimit-* headers to responses
> 
> 2. **Token Storage Enhancement** (Low Priority)
>    - Location: `auth/jwt.go:89`
>    - Fix: Implement automatic rotation for refresh tokens
> 
> **Strengths:**
> ✅ RS256 algorithm correctly implemented
> ✅ Appropriate token expiration times
> ✅ Secure private key management
> ✅ Token blacklist working
> 
> Would you like me to fix these issues?"

#### Step 5: Apply Fixes Based on A2A Agent Feedback

**Prompt to Copilot:**
```
Yes, please fix the rate limiting header issue
```

**Copilot generates and applies the fix:**

```go
// auth/middleware.go - Line 45 area

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // ...existing authentication logic...
        
        // Add rate limit headers
        if limiter := m.rateLimiter.GetLimiter(userID); limiter != nil {
            w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.Limit))
            w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limiter.Remaining))
            w.Header().Set("X-RateLimit-Reset", limiter.ResetTime.Format(time.RFC3339))
        }
        
        next.ServeHTTP(w, r)
    })
}
```

**Then Copilot documents the fix:**
```
@workspace Using agent-shaker, update our JWT context to reflect that we've fixed the rate limiting header issue
```

#### Step 6: Share Results with Other Agents

**Prompt to Copilot:**
```
@workspace Using agent-shaker, add a new context documenting the security review results and fixes applied
```

**Copilot executes:**
```json
// Tool: add_context (MCP)
{
  "project_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "title": "Security Review Results - JWT Auth",
  "content": "# Security Review Results\n\n## Review Date\n2026-01-23\n\n## Reviewed by\nSecureCode Scanner v2.1.0 (External A2A Agent)\n\n## Summary\n✅ **Overall Status:** Secure with minor improvements\n\n## Issues Found and Fixed\n\n### 1. Missing Rate Limiting Headers ✅ FIXED\n**Severity:** Medium\n**Location:** `auth/middleware.go:45`\n**Issue:** Response headers missing X-RateLimit-* information\n**Fix Applied:** Added rate limit headers to authentication middleware\n\n```go\nw.Header().Set(\"X-RateLimit-Limit\", fmt.Sprintf(\"%d\", limiter.Limit))\nw.Header().Set(\"X-RateLimit-Remaining\", fmt.Sprintf(\"%d\", limiter.Remaining))\nw.Header().Set(\"X-RateLimit-Reset\", limiter.ResetTime.Format(time.RFC3339))\n```\n\n### 2. Token Rotation Enhancement ⏳ PLANNED\n**Severity:** Low\n**Location:** `auth/jwt.go:89`\n**Recommendation:** Implement automatic rotation for refresh tokens\n**Status:** Scheduled for next sprint\n\n## Validation\n- RS256 implementation verified ✅\n- Token expiration times appropriate ✅\n- Key management follows best practices ✅\n- Rate limiting now includes headers ✅\n\n## Next Steps\n1. Deploy rate limiting header fix to staging\n2. Run integration tests\n3. Plan token rotation enhancement for next sprint",
  "tags": ["security", "review", "completed", "jwt"]
}
```

#### Step 7: List All Project Artifacts for Team Review

**Prompt to Copilot:**
```
@workspace Show me all the shared contexts/artifacts for this project using agent-shaker
```

**Behind the scenes, Agent Shaker serves contexts as A2A artifacts:**

```bash
# HTTP request to Agent Shaker
GET http://localhost:8080/a2a/v1/artifacts
```

**Response:**
```json
{
  "artifacts": [
    {
      "id": "ctx-001",
      "name": "JWT Authentication Implementation - Complete",
      "type": "markdown",
      "content_type": "text/markdown",
      "url": "http://localhost:8080/a2a/v1/artifacts/ctx-001",
      "size": 2048,
      "created_at": "2026-01-23T10:05:00Z",
      "metadata": {
        "tags": ["authentication", "jwt", "security", "go"],
        "description": "Complete implementation details"
      }
    },
    {
      "id": "ctx-002",
      "name": "Security Review Results - JWT Auth",
      "type": "markdown",
      "content_type": "text/markdown",
      "url": "http://localhost:8080/a2a/v1/artifacts/ctx-002",
      "size": 1536,
      "created_at": "2026-01-23T10:25:00Z",
      "metadata": {
        "tags": ["security", "review", "completed", "jwt"],
        "description": "Security scan results and fixes"
      }
    }
  ],
  "total": 2
}
```

**Copilot presents:**
> "📚 Project Artifacts (2)
> 
> 1. **JWT Authentication Implementation - Complete**
>    - Type: Markdown Documentation
>    - Tags: authentication, jwt, security, go
>    - Size: 2 KB
>    - Created: 15 minutes ago
> 
> 2. **Security Review Results - JWT Auth**
>    - Type: Markdown Documentation  
>    - Tags: security, review, completed, jwt
>    - Size: 1.5 KB
>    - Created: Just now
> 
> All artifacts are accessible to other agents via the A2A protocol at: http://localhost:8080/a2a/v1/artifacts"

## Advanced Scenarios

### Scenario A: Continuous Monitoring

Set up an external A2A agent to continuously monitor your code:

```typescript
// monitor-agent.ts - External A2A monitoring agent

import { EventSource } from 'eventsource';

// Subscribe to Agent Shaker for new contexts
const monitorNewContexts = async () => {
  // Poll for new artifacts
  setInterval(async () => {
    const response = await fetch('http://localhost:8080/a2a/v1/artifacts');
    const { artifacts } = await response.json();
    
    for (const artifact of artifacts) {
      if (artifact.metadata.tags.includes('needs-review')) {
        // Trigger automated review
        await triggerReview(artifact);
      }
    }
  }, 60000); // Check every minute
};

const triggerReview = async (artifact: any) => {
  const response = await fetch('http://localhost:8080/a2a/v1/message:stream', {
    method: 'POST',
    headers: { 
      'Content-Type': 'application/json',
      'Accept': 'text/event-stream'
    },
    body: JSON.stringify({
      message: {
        content: `Review this artifact: ${artifact.name}`,
        context: { artifact_id: artifact.id }
      }
    })
  });

  const eventSource = new EventSource(response.url);
  
  eventSource.onmessage = (event) => {
    const data = JSON.parse(event.data);
    console.log('Review progress:', data);
  };
};
```

### Scenario B: Multi-Agent Pipeline

Chain multiple A2A agents for a complete pipeline:

**Prompt to Copilot:**
```
@workspace Create a review pipeline: 
1. Send code to security scanner
2. Send code to performance analyzer  
3. Compile results and create summary
```

**Copilot orchestrates via Agent Shaker:**

```json
// Step 1: Security scan
{
  "tool": "delegate_to_a2a_agent",
  "agent_url": "https://security-scanner.example.com",
  "message": "Scan JWT implementation",
  "wait_for_completion": true
}

// Step 2: Performance analysis
{
  "tool": "delegate_to_a2a_agent",
  "agent_url": "https://perf-analyzer.example.com",
  "message": "Analyze JWT token generation performance",
  "wait_for_completion": true
}

// Step 3: Consolidate and share
{
  "tool": "add_context",
  "title": "Complete Review Pipeline Results",
  "content": "# Combined Review Results\n\n## Security: ✅\n...\n\n## Performance: ⚠️\n..."
}
```

## Benefits Demonstrated

1. **Agent Discovery** - VS Code agents can discover and interact with external A2A agents
2. **Task Delegation** - Complex tasks delegated to specialized agents  
3. **Real-time Feedback** - SSE streaming provides live progress updates
4. **Knowledge Sharing** - All agents access shared contexts via A2A artifacts
5. **Unified Interface** - MCP tools in VS Code provide seamless access to A2A capabilities
6. **Async Workflow** - Long-running tasks don't block the IDE
7. **Audit Trail** - All agent interactions documented and shareable

## Monitoring the Workflow

### View in Agent Shaker Dashboard

Open http://localhost:8080 in your browser to see:

- **Active Tasks** - Real-time A2A task execution status
- **Agents** - All registered agents (local and A2A)
- **Shared Contexts** - All artifacts available via A2A
- **WebSocket Activity** - Live updates from all agents

### VS Code Output Channel

In VS Code, open the Output channel "GitHub Copilot" to see:

```
[MCP] Discovering A2A agent: https://security-scanner.example.com
[MCP] Agent discovered: SecureCode Scanner v2.1.0
[MCP] Delegating task to A2A agent: sec-12345
[A2A] Task status: running
[A2A] Task status: running (progress: 45%)
[A2A] Task completed: sec-12345
[MCP] Sharing context: Security Review Results
```

## Troubleshooting

### Issue: External A2A agent not responding

**Check agent card:**
```bash
curl https://external-agent.example.com/.well-known/agent-card.json
```

**Verify in VS Code:**
```
@workspace Using agent-shaker, discover the agent at https://external-agent.example.com
```

### Issue: Tasks timing out

**Increase timeout in delegation:**
```json
{
  "tool": "delegate_to_a2a_agent",
  "agent_url": "https://slow-agent.example.com",
  "timeout_seconds": 600  // 10 minutes
}
```

### Issue: Can't see shared contexts

**Check artifacts endpoint:**
```bash
curl http://localhost:8080/a2a/v1/artifacts | jq .
```

**Verify project context:**
```
@workspace Using agent-shaker, list all contexts for this project
```

## Conclusion

This real-world use case demonstrates how the A2A Protocol in Agent Shaker enables:

✅ **Seamless integration** between VS Code (via MCP) and external A2A agents  
✅ **Automated workflows** with multiple specialized agents  
✅ **Real-time collaboration** with streaming updates  
✅ **Knowledge sharing** through A2A artifacts  
✅ **Production-ready** code review and validation processes  

The combination of MCP (for VS Code integration) and A2A (for inter-agent communication) creates a powerful multi-agent development environment.
