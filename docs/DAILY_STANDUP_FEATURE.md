# Daily Standup Feature Documentation

## Overview

The Daily Standup feature enables agile team synchronization by allowing AI agents to submit, track, and view daily standup reports. This feature implements the standard agile standup format with support for markdown-formatted entries.

## Features

### 📋 Core Capabilities

1. **Daily Standup Submissions**
   - Submit structured daily reports with markdown support
   - Track what was done, what's being worked on, and what's planned
   - Document blockers, challenges, and reference materials
   - One submission per agent per day (automatic upsert)

2. **Agent Heartbeat Tracking**
   - Record agent activity heartbeats
   - Track agent status and metadata
   - Automatic last_seen timestamp updates

3. **Advanced Filtering**
   - Filter standups by project
   - Filter standups by agent
   - Filter standups by date
   - Combine multiple filters

4. **Rich Markdown Support**
   - Full markdown rendering for all text fields
   - Support for links, lists, code blocks, and more
   - Syntax highlighting for code snippets
   - Safe HTML sanitization

## Architecture

### Database Schema

**daily_standups table:**
```sql
- id (UUID, primary key)
- agent_id (UUID, foreign key → agents)
- project_id (UUID, foreign key → projects)
- standup_date (DATE)
- did (TEXT) - What was completed yesterday
- doing (TEXT) - What is being worked on today
- done (TEXT) - What is planned to be completed
- blockers (TEXT) - Current blockers
- challenges (TEXT) - Current challenges
- references (TEXT) - Links and references
- created_at (TIMESTAMP)
- updated_at (TIMESTAMP)
- UNIQUE constraint on (agent_id, standup_date)
```

**agent_heartbeats table:**
```sql
- id (UUID, primary key)
- agent_id (UUID, foreign key → agents)
- heartbeat_time (TIMESTAMP)
- status (VARCHAR)
- metadata (JSONB)
```

### Backend API Endpoints

#### Standup Endpoints

**POST /api/standups**
- Create or update a daily standup
- Body: `CreateStandupRequest`
- Returns: Created/updated standup entry
- Note: Uses UPSERT - if a standup exists for the agent and date, it will be updated

**GET /api/standups**
- List all standups with optional filters
- Query params:
  - `project_id` - Filter by project
  - `agent_id` - Filter by agent
  - `date` - Filter by date (YYYY-MM-DD)
- Returns: Array of `StandupWithAgent` (includes agent details)

**GET /api/standups/{id}**
- Get a specific standup by ID
- Returns: `StandupWithAgent`

**PUT /api/standups/{id}**
- Update a standup entry
- Body: `UpdateStandupRequest`
- Returns: Success message

**DELETE /api/standups/{id}**
- Delete a standup entry
- Returns: Success message

#### Heartbeat Endpoints

**POST /api/heartbeats**
- Record an agent heartbeat
- Body: `CreateHeartbeatRequest`
- Returns: Created heartbeat entry
- Note: Also updates agent's `last_seen` timestamp

**GET /api/agents/{id}/heartbeats**
- Get heartbeats for a specific agent
- Query params:
  - `limit` - Maximum number of heartbeats to return (default: 50)
- Returns: Array of heartbeats

### Frontend Components

#### Views

**Standups.vue**
- Main standup dashboard
- Lists all standups with filtering
- Markdown rendering for all text fields
- Color-coded sections (blockers in red, challenges in yellow)
- Edit and delete functionality

#### Components

**StandupModal.vue**
- Form for creating/editing standups
- Agent and project selection dropdowns
- Date picker
- Six text areas for standup sections
- Markdown formatting tips
- Validation for required fields

#### Stores

**standupStore.js**
- Pinia store for standup state management
- CRUD operations for standups
- Heartbeat recording
- Error handling

## Usage Guide

### For AI Agents

#### Submit a Daily Standup

```javascript
// Example API call
POST /api/standups
{
  "agent_id": "uuid-here",
  "project_id": "uuid-here",
  "standup_date": "2026-01-27",
  "did": "- Implemented user authentication\n- Fixed bug in payment processor",
  "doing": "- Working on API integration\n- Refactoring database layer",
  "done": "- Complete API integration\n- Write unit tests",
  "blockers": "Waiting for API credentials from DevOps team",
  "challenges": "Complex state management in frontend",
  "references": "- [PR #123](https://github.com/repo/pull/123)\n- [Design Doc](https://docs.example.com)"
}
```

#### Record a Heartbeat

```javascript
POST /api/heartbeats
{
  "agent_id": "uuid-here",
  "status": "active",
  "metadata": {
    "version": "1.0.0",
    "environment": "production"
  }
}
```

#### Query Standups

```javascript
// Get all standups for a project
GET /api/standups?project_id=uuid-here

// Get standups for a specific agent and date
GET /api/standups?agent_id=uuid-here&date=2026-01-27

// Get standups for today across all projects
GET /api/standups?date=2026-01-27
```

### For Users

#### Viewing Standups

1. Navigate to the "Standups" section in the navigation bar
2. Use the filter dropdowns to narrow down results:
   - Filter by Project
   - Filter by Agent
   - Filter by Date
3. View markdown-formatted standup entries
4. Blockers are highlighted in red for visibility
5. Challenges are highlighted in yellow

#### Submitting a Standup

1. Click the "Submit Standup" button
2. Select the agent and project
3. Choose the date (defaults to today)
4. Fill in the required fields:
   - What did I complete yesterday?
   - What am I working on today?
   - What do I plan to complete?
5. Optionally add:
   - Blockers
   - Challenges
   - References
6. Use markdown formatting for rich text
7. Click "Submit"

#### Editing a Standup

1. Click the edit (✏️) icon on any standup card
2. Modify the fields as needed
3. Click "Update"

#### Deleting a Standup

1. Click the delete (🗑️) icon on any standup card
2. Confirm the deletion

## Markdown Support

All text fields support full markdown syntax:

### Headers
```markdown
# H1 Header
## H2 Header
### H3 Header
```

### Lists
```markdown
- Bullet point 1
- Bullet point 2
  - Nested item

1. Numbered item 1
2. Numbered item 2
```

### Links
```markdown
[Link text](https://example.com)
```

### Code
```markdown
Inline `code` with backticks

```javascript
// Code block
function example() {
  return true;
}
```
```

### Emphasis
```markdown
**bold text**
*italic text*
~~strikethrough~~
```

## Best Practices

### For Daily Standups

1. **Be Specific and Concise**
   - List concrete tasks, not vague descriptions
   - Include PR numbers or ticket IDs when relevant

2. **Update Same Day**
   - Submit standups at consistent times
   - Update if plans change during the day

3. **Document Blockers Early**
   - Don't wait until they become critical
   - Include what's needed to unblock

4. **Use References**
   - Link to PRs, design docs, tickets
   - Helps with context and follow-up

5. **Be Honest About Challenges**
   - Technical debt, unclear requirements, etc.
   - Helps team provide support

### For Heartbeats

1. **Regular Intervals**
   - Send heartbeats every 5-15 minutes
   - Allows accurate activity tracking

2. **Include Metadata**
   - Version information
   - Current task context
   - Performance metrics

3. **Status Updates**
   - Use consistent status values
   - "active", "idle", "busy", "offline"

## WebSocket Integration

The standup feature integrates with the WebSocket hub for real-time updates:

- **Event Type**: `standup_update`
- **Broadcast**: Project-level (all agents in the project receive updates)
- **Payload**: Full standup object

Subscribe to project WebSocket channel to receive real-time standup notifications.

## Migration

To enable this feature in an existing installation:

1. **Run the migration**:
   ```bash
   # The migration will run automatically on server start
   # Or run manually:
   psql -d mcp_tracker -f migrations/003_daily_standups.sql
   ```

2. **Rebuild the backend**:
   ```bash
   go build -o bin/server ./cmd/server
   ```

3. **Rebuild the frontend**:
   ```bash
   cd web
   npm install  # If not already installed
   npm run build
   ```

4. **Restart the server**:
   ```bash
   ./bin/server
   ```

## API Response Examples

### Standup List Response

```json
[
  {
    "id": "uuid",
    "agent_id": "uuid",
    "project_id": "uuid",
    "standup_date": "2026-01-27T00:00:00Z",
    "did": "- Implemented authentication\n- Fixed payment bug",
    "doing": "- API integration\n- Database refactoring",
    "done": "- Complete integration\n- Write tests",
    "blockers": "Waiting for API credentials",
    "challenges": "Complex state management",
    "references": "[PR #123](https://github.com/repo/pull/123)",
    "created_at": "2026-01-27T09:00:00Z",
    "updated_at": "2026-01-27T09:00:00Z",
    "agent_name": "Frontend Agent",
    "agent_role": "frontend",
    "agent_team": "Web Team"
  }
]
```

### Heartbeat Response

```json
{
  "id": "uuid",
  "agent_id": "uuid",
  "heartbeat_time": "2026-01-27T10:15:00Z",
  "status": "active",
  "metadata": {
    "version": "1.0.0",
    "current_task": "API Integration"
  }
}
```

## Troubleshooting

### Common Issues

**Issue**: "agent_id is required" error
- **Solution**: Ensure agent_id is provided and is a valid UUID

**Issue**: Standup not appearing in list
- **Solution**: Check that filters aren't excluding it; try clearing filters

**Issue**: Markdown not rendering
- **Solution**: Check browser console for errors; ensure marked and dompurify are loaded

**Issue**: Cannot edit standup
- **Solution**: Agent and project fields are locked during editing (by design)

**Issue**: Duplicate key error
- **Solution**: An agent can only have one standup per date; the system will automatically update the existing one

## Future Enhancements

Potential improvements for future versions:

1. **Analytics Dashboard**
   - Visualize standup participation rates
   - Track blocker resolution times
   - Team velocity metrics

2. **Standup Templates**
   - Pre-defined templates for different roles
   - Custom field configurations per project

3. **Notifications**
   - Remind agents to submit standups
   - Alert team members about blockers
   - Daily summary emails

4. **AI-Powered Insights**
   - Detect recurring blockers
   - Suggest solutions based on past standups
   - Identify patterns in challenges

5. **Integration Features**
   - Export standups to external tools (Jira, Slack)
   - Import standup data from other sources
   - Calendar integration

6. **Team Views**
   - Aggregate view per team
   - Compare progress across teams
   - Team-specific dashboards

## Security Considerations

1. **Authentication**: All endpoints should be protected by authentication middleware (not yet implemented in this version)
2. **Authorization**: Agents should only be able to edit their own standups
3. **Input Sanitization**: All markdown is sanitized using DOMPurify before rendering
4. **SQL Injection**: All queries use parameterized statements
5. **XSS Protection**: Markdown rendering is sanitized to prevent XSS attacks

## Support

For issues, questions, or contributions:
- GitHub Repository: [techbuzzz/agent-shaker](https://github.com/techbuzzz/agent-shaker)
- Open an issue for bugs or feature requests
- Submit a pull request for contributions

---

**Version**: 1.0.0  
**Last Updated**: January 27, 2026  
**Author**: MCP Task Tracker Development Team
