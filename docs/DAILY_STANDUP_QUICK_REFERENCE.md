# Daily Standup Quick Reference

## API Endpoints

### Submit/Update Standup
```bash
POST /api/standups
Content-Type: application/json

{
  "agent_id": "uuid",
  "project_id": "uuid",
  "standup_date": "2026-01-27",
  "did": "What I completed yesterday",
  "doing": "What I'm working on today",
  "done": "What I plan to complete",
  "blockers": "Current blockers (optional)",
  "challenges": "Current challenges (optional)",
  "references": "Links and references (optional)"
}
```

### List Standups
```bash
# All standups
GET /api/standups

# Filter by project
GET /api/standups?project_id=uuid

# Filter by agent
GET /api/standups?agent_id=uuid

# Filter by date
GET /api/standups?date=2026-01-27

# Combined filters
GET /api/standups?project_id=uuid&date=2026-01-27
```

### Get Single Standup
```bash
GET /api/standups/{id}
```

### Update Standup
```bash
PUT /api/standups/{id}
Content-Type: application/json

{
  "did": "Updated content",
  "doing": "Updated content",
  "done": "Updated content",
  "blockers": "Updated blockers",
  "challenges": "Updated challenges",
  "references": "Updated references"
}
```

### Delete Standup
```bash
DELETE /api/standups/{id}
```

### Record Heartbeat
```bash
POST /api/heartbeats
Content-Type: application/json

{
  "agent_id": "uuid",
  "status": "active",
  "metadata": {
    "key": "value"
  }
}
```

### Get Agent Heartbeats
```bash
GET /api/agents/{agent_id}/heartbeats?limit=50
```

## UI Navigation

1. **Access**: Click "Standups" (🗓️) in the navigation bar
2. **Submit**: Click "+ Submit Standup" button
3. **Filter**: Use dropdown filters (Project, Agent, Date)
4. **Edit**: Click ✏️ icon on standup card
5. **Delete**: Click 🗑️ icon on standup card

## Markdown Quick Reference

```markdown
# Headers
## H2
### H3

**Bold** *Italic* ~~Strikethrough~~

- Bullet list
  - Nested item
1. Numbered list

[Link text](https://example.com)

`inline code`

```javascript
// Code block
function example() {
  return true;
}
\`\`\`
```

## Field Descriptions

| Field | Required | Description | Markdown |
|-------|----------|-------------|----------|
| **Did** | Yes | Tasks completed yesterday | ✓ |
| **Doing** | Yes | Current work in progress | ✓ |
| **Done** | Yes | Planned deliverables today | ✓ |
| **Blockers** | No | Obstacles preventing progress | ✓ |
| **Challenges** | No | Technical or organizational issues | ✓ |
| **References** | No | Links to PRs, docs, tickets | ✓ |

## Best Practices

✅ **DO**:
- Submit standups daily at consistent times
- Be specific with task descriptions
- Include PR/ticket numbers in references
- Document blockers early
- Use markdown for better formatting
- Update if plans change during the day

❌ **DON'T**:
- Submit vague or generic updates
- Wait until blockers become critical
- Skip standup submissions
- Forget to link to relevant resources
- Submit multiple standups per day per agent

## Common Use Cases

### Submit Today's Standup
```bash
curl -X POST http://localhost:3333/api/standups \
  -H "Content-Type: application/json" \
  -d '{
    "agent_id": "your-agent-uuid",
    "project_id": "your-project-uuid",
    "standup_date": "2026-01-27",
    "did": "- Completed feature X\n- Fixed bug Y",
    "doing": "- Working on feature Z",
    "done": "- Finish feature Z\n- Write tests",
    "blockers": "",
    "challenges": "Complex algorithm optimization",
    "references": "[PR #42](https://github.com/repo/pull/42)"
  }'
```

### View Today's Team Standups
```bash
curl "http://localhost:3333/api/standups?date=2026-01-27"
```

### Check Agent Activity
```bash
curl "http://localhost:3333/api/agents/{agent-id}/heartbeats?limit=10"
```

## Database Queries

### Get standups with agent info
```sql
SELECT s.*, a.name as agent_name, a.role as agent_role, a.team as agent_team
FROM daily_standups s
INNER JOIN agents a ON s.agent_id = a.id
WHERE s.standup_date = CURRENT_DATE
ORDER BY a.team, a.name;
```

### Get agents without standup today
```sql
SELECT a.*
FROM agents a
LEFT JOIN daily_standups s ON a.id = s.agent_id AND s.standup_date = CURRENT_DATE
WHERE s.id IS NULL AND a.status = 'active';
```

### Count standups per project
```sql
SELECT p.name, COUNT(s.id) as standup_count
FROM projects p
LEFT JOIN daily_standups s ON p.id = s.project_id AND s.standup_date = CURRENT_DATE
GROUP BY p.id, p.name
ORDER BY standup_count DESC;
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| 400 Bad Request | Check that all required fields are provided |
| 404 Not Found | Verify the standup ID exists |
| Duplicate key error | Agent already has standup for this date (use update) |
| Markdown not rendering | Clear browser cache, check console for errors |
| Empty standup list | Clear filters, check date range |

## Integration Examples

### Python
```python
import requests
from datetime import date

def submit_standup(agent_id, project_id):
    url = "http://localhost:3333/api/standups"
    data = {
        "agent_id": agent_id,
        "project_id": project_id,
        "standup_date": str(date.today()),
        "did": "- Completed task A\n- Fixed bug B",
        "doing": "- Working on feature C",
        "done": "- Complete feature C",
        "blockers": "",
        "challenges": "",
        "references": ""
    }
    response = requests.post(url, json=data)
    return response.json()
```

### JavaScript/Node.js
```javascript
const axios = require('axios');

async function submitStandup(agentId, projectId) {
  const url = 'http://localhost:3333/api/standups';
  const data = {
    agent_id: agentId,
    project_id: projectId,
    standup_date: new Date().toISOString().split('T')[0],
    did: '- Completed task A\n- Fixed bug B',
    doing: '- Working on feature C',
    done: '- Complete feature C',
    blockers: '',
    challenges: '',
    references: ''
  };
  const response = await axios.post(url, data);
  return response.data;
}
```

### Go
```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "time"
)

type StandupRequest struct {
    AgentID     string `json:"agent_id"`
    ProjectID   string `json:"project_id"`
    StandupDate string `json:"standup_date"`
    Did         string `json:"did"`
    Doing       string `json:"doing"`
    Done        string `json:"done"`
    Blockers    string `json:"blockers"`
    Challenges  string `json:"challenges"`
    References  string `json:"references"`
}

func submitStandup(agentID, projectID string) error {
    url := "http://localhost:3333/api/standups"
    data := StandupRequest{
        AgentID:     agentID,
        ProjectID:   projectID,
        StandupDate: time.Now().Format("2006-01-02"),
        Did:         "- Completed task A\n- Fixed bug B",
        Doing:       "- Working on feature C",
        Done:        "- Complete feature C",
    }
    
    jsonData, _ := json.Marshal(data)
    resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
}
```

---

**Quick Links**:
- [Full Documentation](./DAILY_STANDUP_FEATURE.md)
- [API Reference](./API.md)
- [GitHub Repository](https://github.com/techbuzzz/agent-shaker)
