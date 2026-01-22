# Quick Markdown Reference for AI Agents

## Common Patterns for Context Sharing

### 1. API Endpoint Documentation

```markdown
# [Endpoint Name] API

## Endpoint
`[METHOD] /api/path`

## Description
Brief description of what this endpoint does.

## Request
```json
{
  "field": "value"
}
```

## Response
```json
{
  "result": "value"
}
```

## Example
```bash
curl -X POST http://localhost:8080/api/endpoint \
  -H "Content-Type: application/json" \
  -d '{"field":"value"}'
```

## Notes
- Important considerations
- Edge cases
- Security notes
```

### 2. Bug Report Template

```markdown
# Bug: [Short Description]

## Issue
❌ **Problem:** Clear description of the bug

## Steps to Reproduce
1. Do this
2. Then this
3. Observe the error

## Expected Behavior
What should happen

## Actual Behavior
What actually happens

## Root Cause
Technical explanation

## Solution
How it was fixed

## Testing
- [x] Test case 1
- [x] Test case 2

## Status
✅ Fixed | ⏳ In Progress | ❌ Blocked
```

### 3. Feature Implementation

```markdown
# Feature: [Feature Name]

## Overview
Brief description

## Implementation Details

### Components Changed
- `file1.js` - Added new function
- `file2.vue` - Updated component

### Code Examples
```javascript
// Key implementation
function newFeature() {
  // code
}
```

## Features
- ✅ Feature 1
- ✅ Feature 2
- ⏳ Feature 3 (TODO)

## Testing
How to test this feature

## Dependencies
- Library 1 (v1.0)
- Library 2 (v2.0)
```

### 4. Architecture Decision

```markdown
# ADR-[###]: [Title]

## Status
✅ Accepted | 🔄 Proposed | ❌ Rejected

## Context
Why this decision is needed

## Decision
What was decided

## Alternatives Considered
| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| A | Pro | Con | ✅/❌ |
| B | Pro | Con | ✅/❌ |

## Consequences
### Positive
- Benefit 1
- Benefit 2

### Negative
- Tradeoff 1
- Tradeoff 2

## Implementation
Key implementation notes
```

### 5. Setup/Configuration Guide

```markdown
# Setup: [Component Name]

## Prerequisites
- Requirement 1
- Requirement 2

## Installation
```bash
npm install package-name
```

## Configuration
```json
{
  "setting": "value"
}
```

## Environment Variables
| Variable | Description | Example |
|----------|-------------|---------|
| API_KEY | API key | `abc123` |

## Verification
```bash
# Test command
npm test
```

## Troubleshooting
**Problem:** Issue
**Solution:** Fix
```

### 6. Code Review Notes

```markdown
# Review: [Component/Feature Name]

## Summary
Overall assessment

## Observations

### ✅ Good Practices
- Thing 1
- Thing 2

### ⚠️ Suggestions
- Improvement 1
- Improvement 2

### 🔴 Issues
- Critical issue 1
- Critical issue 2

## Code Snippets

**Before:**
```javascript
// old code
```

**After:**
```javascript
// improved code
```

## Recommendations
1. Recommendation 1
2. Recommendation 2
```

### 7. Task Completion Report

```markdown
# Completed: [Task Name]

## Task ID
`task-uuid-here`

## What Was Done
- Implemented feature X
- Fixed bug Y
- Updated documentation

## Files Changed
- `src/file1.js` - Added function
- `src/file2.vue` - Updated component
- `docs/README.md` - Updated docs

## Key Code
```javascript
// Important implementation
```

## Testing
- [x] Unit tests pass
- [x] Integration tests pass
- [x] Manual testing done

## Next Steps
- [ ] TODO item 1
- [ ] TODO item 2

## Notes for Other Agents
Important information for teammates
```

### 8. Database Schema Documentation

```markdown
# Database: [Table Name]

## Schema
```sql
CREATE TABLE table_name (
    id UUID PRIMARY KEY,
    field TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Relationships
- `table_name.id` → `other_table.foreign_key`

## Indexes
```sql
CREATE INDEX idx_field ON table_name(field);
```

## Sample Data
```sql
INSERT INTO table_name (id, field) VALUES
('uuid-1', 'value1'),
('uuid-2', 'value2');
```

## Queries

**Get all records:**
```sql
SELECT * FROM table_name ORDER BY created_at DESC;
```

**Filter by field:**
```sql
SELECT * FROM table_name WHERE field = 'value';
```
```

## Formatting Tips

### Emphasis
- `**bold**` for important terms
- `*italic*` for subtle emphasis
- `~~strikethrough~~` for deprecated
- `` `code` `` for variables/commands

### Lists
```markdown
Unordered:
- Item 1
- Item 2
  - Nested item

Ordered:
1. Step 1
2. Step 2

Tasks:
- [x] Done
- [ ] Todo
```

### Status Badges
- ✅ Complete/Success
- ⏳ In Progress
- ❌ Failed/Blocked
- 🔄 Under Review
- 📝 Draft
- 🔒 Locked
- 🎯 Important
- 💡 Idea
- ⚠️ Warning
- 🐛 Bug
- 🚀 Deployed

### Code Blocks
Always specify language:
````markdown
```javascript
// JS code
```

```python
# Python code
```

```go
// Go code
```

```sql
-- SQL
```

```bash
# Shell commands
```

```json
{
  "json": "data"
}
```
````

### Tables
```markdown
| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Data     | Data     | Data     |
```

Alignment:
```markdown
| Left | Center | Right |
|:-----|:------:|------:|
| L    | C      | R     |
```

### Links
```markdown
[Link Text](https://url.com)
[Internal Link](#section-heading)
```

### Blockquotes
```markdown
> Important note or quote
> 
> Can be multiple lines
```

### Horizontal Rules
```markdown
---
Use to separate major sections
```

## Example: Complete Context

```json
{
  "title": "User Service API - Complete Implementation",
  "content": "# User Service API\n\n## Overview\nImplemented full CRUD operations for user management.\n\n## Endpoints\n\n### GET /api/users\n**Description:** List all users\n\n**Response:**\n```json\n[\n  {\n    \"id\": \"uuid\",\n    \"name\": \"John Doe\",\n    \"email\": \"john@example.com\"\n  }\n]\n```\n\n### POST /api/users\n**Description:** Create new user\n\n**Request:**\n```json\n{\n  \"name\": \"Jane Doe\",\n  \"email\": \"jane@example.com\",\n  \"password\": \"secure_password\"\n}\n```\n\n**Response:**\n```json\n{\n  \"id\": \"new-uuid\",\n  \"name\": \"Jane Doe\",\n  \"email\": \"jane@example.com\"\n}\n```\n\n## Implementation Notes\n\n### Security\n- ✅ Passwords hashed with bcrypt\n- ✅ Email validation\n- ✅ Rate limiting enabled\n- ⚠️ TODO: Add email verification\n\n### Database\n```sql\nCREATE TABLE users (\n    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),\n    name TEXT NOT NULL,\n    email TEXT UNIQUE NOT NULL,\n    password_hash TEXT NOT NULL,\n    created_at TIMESTAMP DEFAULT NOW()\n);\n```\n\n### Testing\n```bash\n# Run tests\ngo test ./internal/handlers/users_test.go\n\n# Coverage\ngo test -cover\n```\n\n## Status\n✅ **Production Ready**\n\n---\n\n**Author:** Backend Agent  \n**Date:** 2026-01-22  \n**Version:** 1.0",
  "tags": ["api", "users", "backend", "crud", "complete"]
}
```

## Tips for AI Agents

1. **Be Clear**: Use descriptive titles and headers
2. **Be Structured**: Organize with headers and lists
3. **Be Complete**: Include code examples and context
4. **Be Helpful**: Add notes for other agents
5. **Be Tagged**: Use relevant tags for discovery
6. **Be Current**: Update docs when code changes
7. **Be Visual**: Use status badges and emojis
8. **Be Specific**: Include actual code, not pseudocode

## Quick Test

Try adding this simple context:

```bash
curl "http://localhost:8080?project_id=PROJECT_ID&agent_id=AGENT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "add_context",
      "arguments": {
        "title": "Quick Test",
        "content": "# Test\n\n## Working\n\n- [x] Markdown formatting\n- [x] Code blocks\n\n```javascript\nconsole.log(\"Hello from agent!\");\n```\n\n**Status:** ✅ Working!",
        "tags": ["test"]
      }
    }
  }'
```

---

**Quick Reference Version:** 1.0  
**Last Updated:** 2026-01-22
