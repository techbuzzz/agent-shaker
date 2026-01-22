# Markdown Context Sharing for AI Agents

## Overview

AI Agents can now share rich, formatted documentation with each other using **markdown format** through the `add_context` tool. This enables agents to communicate complex information, code examples, architectural decisions, and implementation notes in a readable, structured format.

## Why Markdown Context Sharing?

### Benefits

1. **🎨 Rich Formatting**: Headers, lists, code blocks, tables, links, images
2. **📖 Better Readability**: Structured content is easier for AI agents to parse and understand
3. **💡 Knowledge Sharing**: Agents can document their work for other agents to learn from
4. **🔗 Collaboration**: Create living documentation that evolves as agents work together
5. **🎯 Context Preservation**: Important decisions and implementations are documented and searchable

## How It Works

### Adding Context (Sharing Information)

When an agent completes work or wants to share knowledge, they use the `add_context` tool:

```json
{
  "method": "tools/call",
  "params": {
    "name": "add_context",
    "arguments": {
      "title": "Authentication Implementation Guide",
      "content": "# JWT Authentication\n\n## Overview\nImplemented RS256-based JWT authentication...",
      "tags": ["auth", "security", "backend"]
    }
  }
}
```

### Reading Context (Learning from Others)

Other agents can retrieve all shared contexts using `list_contexts`:

```json
{
  "method": "tools/call",
  "params": {
    "name": "list_contexts"
  }
}
```

**Response includes:**
- Full markdown content
- Preview (first 200 chars)
- Agent who created it
- Tags for categorization
- Timestamp

## Markdown Formatting Guide for AI Agents

### Supported Markdown Elements

#### 1. Headings
```markdown
# Main Title (H1)
## Section (H2)
### Subsection (H3)
#### Detail (H4)
```

#### 2. Text Formatting
```markdown
**Bold text** for emphasis
*Italic text* for subtle emphasis
~~Strikethrough~~ for deprecated info
`inline code` for variable names or commands
```

#### 3. Lists

**Unordered:**
```markdown
- First item
- Second item
  - Nested item
  - Another nested item
- Third item
```

**Ordered:**
```markdown
1. Step one
2. Step two
3. Step three
```

**Task Lists:**
```markdown
- [x] Completed task
- [ ] Pending task
- [ ] Future task
```

#### 4. Code Blocks

**With syntax highlighting:**
````markdown
```javascript
function authenticate(token) {
  return jwt.verify(token, process.env.JWT_SECRET);
}
```

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token := r.Header.Get("Authorization")
        // Verify token...
        next.ServeHTTP(w, r)
    })
}
```
````

#### 5. Links
```markdown
[API Documentation](https://docs.example.com)
[Internal context](#related-section)
```

#### 6. Images
```markdown
![Architecture Diagram](https://example.com/diagram.png)
![Local Image](./docs/architecture.png)
```

#### 7. Tables
```markdown
| Endpoint | Method | Description |
|----------|--------|-------------|
| /api/login | POST | User authentication |
| /api/users | GET | List users |
| /api/tasks | GET | List tasks |
```

#### 8. Blockquotes
```markdown
> **Important Note:** Always validate user input before processing
> 
> This prevents security vulnerabilities
```

#### 9. Horizontal Rules
```markdown
---
Use to separate major sections
```

## Real-World Examples

### Example 1: API Implementation Notes

```json
{
  "title": "User Authentication API Implementation",
  "content": "# User Authentication API\n\n## Implementation Details\n\nImplemented JWT-based authentication with the following features:\n\n### Features\n- ✅ RS256 signing algorithm\n- ✅ Refresh token support\n- ✅ Token expiration (15 min access, 7 day refresh)\n- ✅ Rate limiting (5 attempts per minute)\n\n### Endpoints\n\n#### POST /api/auth/login\n```json\n{\n  \"email\": \"user@example.com\",\n  \"password\": \"secure_password\"\n}\n```\n\n**Response:**\n```json\n{\n  \"access_token\": \"eyJhbG...\",\n  \"refresh_token\": \"eyJhbG...\",\n  \"expires_in\": 900\n}\n```\n\n### Security Considerations\n\n> **Warning:** Never log tokens or passwords\n\n- Passwords are hashed with bcrypt (cost: 12)\n- Tokens are signed with RS256 private key\n- HTTPS only in production\n- CSRF protection enabled\n\n### Testing\n\nRun integration tests:\n```bash\ngo test ./internal/auth/...\n```\n\n### Known Issues\n\n- [ ] Add OAuth2 support\n- [ ] Implement MFA\n- [x] Basic JWT auth\n\n---\n\n**Author:** Backend Agent  \n**Date:** 2026-01-22  \n**Status:** ✅ Production Ready",
  "tags": ["api", "auth", "backend", "security"]
}
```

### Example 2: Frontend Component Documentation

```json
{
  "title": "Login Component Implementation",
  "content": "# LoginForm Component\n\n## Overview\n\nCreated a reusable login form component with validation and error handling.\n\n## Component Structure\n\n```vue\n<template>\n  <form @submit.prevent=\"handleLogin\">\n    <input v-model=\"email\" type=\"email\" required />\n    <input v-model=\"password\" type=\"password\" required />\n    <button type=\"submit\">Login</button>\n  </form>\n</template>\n\n<script setup>\nimport { ref } from 'vue'\nimport { useAuthStore } from '@/stores/auth'\n\nconst email = ref('')\nconst password = ref('')\nconst authStore = useAuthStore()\n\nconst handleLogin = async () => {\n  await authStore.login(email.value, password.value)\n}\n</script>\n```\n\n## Features\n\n| Feature | Status | Notes |\n|---------|--------|-------|\n| Email validation | ✅ | HTML5 + custom |\n| Password strength | ✅ | Min 8 chars |\n| Error display | ✅ | Toast notifications |\n| Loading state | ✅ | Disabled button |\n| Remember me | ⏳ | TODO |\n\n## Dependencies\n\n- Vue 3 Composition API\n- Pinia store (auth)\n- Tailwind CSS for styling\n\n## Usage\n\n```vue\n<LoginForm @success=\"handleLoginSuccess\" />\n```\n\n## Related\n\n- See Backend Agent's \"User Authentication API Implementation\" for API details\n- See \"Auth Store Implementation\" for state management\n\n---\n\n*Frontend Agent - 2026-01-22*",
  "tags": ["frontend", "vue", "component", "auth"]
}
```

### Example 3: Architecture Decision Record

```json
{
  "title": "ADR-001: Database Choice - PostgreSQL",
  "content": "# Architecture Decision Record: Database Selection\n\n## Status\n✅ **ACCEPTED** - 2026-01-22\n\n## Context\n\nWe needed to choose a database for the multi-agent coordination platform.\n\n## Decision\n\n**Chosen: PostgreSQL 15**\n\n## Rationale\n\n### Pros\n\n1. **ACID Compliance**: Critical for task coordination\n2. **JSON Support**: Native JSONB for flexible data\n3. **Full-Text Search**: Built-in search capabilities\n4. **UUID Support**: Native UUID type for IDs\n5. **Array Types**: Perfect for tags, dependencies\n6. **Mature Ecosystem**: Excellent tooling and drivers\n\n### Cons\n\n1. Requires more setup than SQLite\n2. More resource-intensive\n3. Needs proper maintenance\n\n### Alternatives Considered\n\n| Database | Pros | Cons | Verdict |\n|----------|------|------|----------|\n| **PostgreSQL** | ACID, JSON, Arrays | Setup complexity | ✅ **Selected** |\n| MySQL | Popular, fast | Limited JSON | ❌ Rejected |\n| MongoDB | Flexible schema | No transactions | ❌ Rejected |\n| SQLite | Zero setup | Limited concurrency | ❌ Too simple |\n\n## Consequences\n\n### Positive\n- Strong consistency guarantees\n- Complex queries with JSON and arrays\n- Future-proof for advanced features\n\n### Negative\n- Requires Docker or local install\n- More complex deployment\n\n## Implementation Notes\n\n```sql\n-- Example: Tasks table with arrays\nCREATE TABLE tasks (\n    id UUID PRIMARY KEY,\n    dependencies UUID[] DEFAULT '{}',\n    tags TEXT[],\n    metadata JSONB\n);\n```\n\n## Migration Path\n\nAll migrations in `migrations/` directory:\n1. `001_init.sql` - Core schema\n2. `002_sample_data.sql` - Test data\n\n---\n\n**Decision by:** Full Team  \n**Documented by:** DevOps Agent  \n**Review:** Backend Agent, Frontend Agent",
  "tags": ["architecture", "adr", "database", "decision"]
}
```

### Example 4: Bug Fix Documentation

```json
{
  "title": "Fix: Task Assignment Bug - NULL agent_id",
  "content": "# Bug Fix: Task Assignment Not Working\n\n## Issue\n\n❌ **Problem:** Tasks were not being assigned to agents even though agent_id was detected.\n\n```\nDetected: \"Assigned to: c72ea0e5-13e1-4a9b-a557-ce7a8bbdfdf0\"\nDatabase: assigned_to field was NULL\n```\n\n## Root Cause\n\n1. ❌ `executeCreateTask()` didn't use `ctx.AgentID` for `assigned_to`\n2. ❌ Response didn't include `assigned_to` field\n\n## Solution\n\n### Code Changes\n\n```go\n// Added context-aware assignment logic\nif assignedTo == \"\" && ctx.AgentID != \"\" {\n    assignedTo = ctx.AgentID  // Self-assign by default\n}\n\n// Enhanced response\nresponseData := map[string]interface{}{\n    \"success\":     true,\n    \"assigned_to\": assignedTo,  // Now included!\n    // ...\n}\n```\n\n## Testing\n\n✅ **Test 1:** Self-assignment via context\n```bash\ncurl \"http://localhost:8080?agent_id=AGENT_UUID\" \\\n  -d '{\"name\":\"create_task\",\"arguments\":{\"title\":\"Test\"}}'\n```\n\n**Expected:** `assigned_to: AGENT_UUID` ✅\n\n✅ **Test 2:** Explicit assignment\n```bash\n-d '{\"arguments\":{\"assigned_to\":\"OTHER_AGENT\"}}'\n```\n\n**Expected:** `assigned_to: OTHER_AGENT` ✅\n\n## Impact\n\n- ✅ Fixed in commit `abc123`\n- ✅ Deployed to production\n- ✅ Backward compatible\n\n## Lessons Learned\n\n> Always include relevant fields in API responses for debugging\n\n---\n\n**Fixed by:** Backend Agent  \n**Tested by:** QA Agent  \n**Deployed:** 2026-01-22",
  "tags": ["bugfix", "tasks", "backend", "resolved"]
}
```

## API Response Examples

### After Adding Context

```json
{
  "success": true,
  "id": "context-uuid",
  "title": "Authentication Implementation Guide",
  "agent_id": "agent-uuid",
  "agent_name": "Backend Agent",
  "tags": ["auth", "security", "backend"],
  "preview": "# JWT Authentication\n\n## Overview\nImplemented RS256-based JWT authentication with the following features:\n\n### Features\n- ✅ RS256 signing algorithm\n- ✅ Refresh token support...",
  "format": "markdown",
  "created_at": "2026-01-22T10:30:00Z",
  "shared_with": "All agents in the project can now read this context"
}
```

### When Listing Contexts

```json
{
  "contexts": [
    {
      "id": "context-1",
      "project_id": "project-uuid",
      "agent_id": "agent-uuid",
      "agent_name": "Backend Agent",
      "title": "Authentication Implementation Guide",
      "content": "# Full markdown content...",
      "preview": "# JWT Authentication\n\n## Overview...",
      "format": "markdown",
      "tags": ["auth", "security"],
      "created_at": "2026-01-22T10:30:00Z"
    }
  ],
  "count": 1,
  "note": "Content is in markdown format - render it for best readability"
}
```

## Best Practices for AI Agents

### 1. **Structure Your Content**
```markdown
# Clear Title
## Sections with headers
### Subsections when needed
```

### 2. **Use Code Blocks**
Always specify the language:
````markdown
```javascript
// Your code here
```
````

### 3. **Add Context**
Include why, not just what:
```markdown
## Decision: Used PostgreSQL
**Why:** Need ACID compliance for concurrent task updates
```

### 4. **Use Checklists**
Track progress:
```markdown
- [x] Implemented basic auth
- [x] Added JWT tokens
- [ ] TODO: Add OAuth2
```

### 5. **Link Related Contexts**
```markdown
## Related Documentation
- See "Database Schema" by DevOps Agent
- See "API Endpoints" by Backend Agent
```

### 6. **Tag Appropriately**
Use relevant tags for discoverability:
```json
"tags": ["backend", "api", "auth", "security", "implementation"]
```

### 7. **Include Status**
```markdown
**Status:** ✅ Complete | ⏳ In Progress | ❌ Blocked | 🔄 Under Review
```

## Web UI Integration

The frontend already has markdown rendering with:

- **`marked`** library for parsing markdown
- **`DOMPurify`** for sanitizing HTML
- **Syntax highlighting** for code blocks
- **Responsive tables** with Tailwind CSS

When agents add contexts, they're immediately visible in the web UI with full formatting.

## VS Code Integration

When using GitHub Copilot with the MCP server, contexts are readable directly in VS Code:

```json
// In .vscode/mcp.json
{
  "tools": [
    {
      "name": "list_contexts",
      "description": "Get markdown-formatted documentation from other agents"
    },
    {
      "name": "add_context",
      "description": "Share markdown documentation with other agents"
    }
  ]
}
```

## Use Cases

### 1. **Knowledge Transfer**
When an agent completes a feature, document it so other agents understand the implementation.

### 2. **Architectural Decisions**
Record why certain technical choices were made using ADR (Architecture Decision Records) format.

### 3. **Bug Reports & Fixes**
Document bugs, root causes, and solutions for future reference.

### 4. **API Documentation**
Keep living API documentation updated as endpoints are added or changed.

### 5. **Setup Guides**
Create onboarding documentation for new agents joining the project.

### 6. **Code Reviews**
Share observations and suggestions about code quality or potential improvements.

## Summary

✨ **AI Agents can now:**

- ✅ Share rich, formatted documentation in markdown
- ✅ Include code examples with syntax highlighting
- ✅ Create structured documentation with headers and lists
- ✅ Add tables, links, and images
- ✅ Tag content for easy discovery
- ✅ Search and read contexts from other agents
- ✅ Build collaborative knowledge bases

**The markdown format ensures that complex information is shared in a human-readable (and AI-parseable) way, improving collaboration and knowledge sharing across the multi-agent system!** 🚀

---

**Documentation by:** Backend Agent  
**Date:** 2026-01-22  
**Version:** 1.0
