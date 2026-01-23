# A2A Agent Card Schema Update - Official v1.0

## Overview

Updated Agent Shaker to comply with the **official A2A Agent Card Schema v1.0** as specified at:
https://gist.githubusercontent.com/SecureAgentTools/0815a2de9cc31c71468afd3d2eef260a/raw/agent-card-schema.md

## Major Changes

### 1. Agent Card Structure

**Old (Custom) Schema:**
```json
{
  "name": "Agent Shaker",
  "version": "1.0.0",
  "capabilities": [
    {"type": "task", "description": "..."}
  ],
  "endpoints": [...]
}
```

**New (Official v1.0) Schema:**
```json
{
  "schemaVersion": "1.0",
  "humanReadableId": "techbuzzz/agent-shaker",
  "agentVersion": "1.0.0",
  "name": "Agent Shaker",
  "description": "...",
  "url": "http://localhost:8080/a2a/v1",
  "provider": {
    "name": "techbuzzz",
    "url": "https://github.com/techbuzzz/agent-shaker"
  },
  "capabilities": {
    "a2aVersion": "1.0",
    "mcpVersion": "0.6",
    "supportedMessageParts": ["text", "file", "data"]
  },
  "authSchemes": [
    {"scheme": "none", "description": "..."}
  ],
  "skills": [
    {"id": "task_execution", "name": "...", "description": "..."}
  ]
}
```

### 2. Required Fields (Per Official Schema)

| Field | Type | Description |
|-------|------|-------------|
| `schemaVersion` | string | Schema version (e.g., "1.0") |
| `humanReadableId` | string | Unique identifier (e.g., "org/agent-name") |
| `agentVersion` | string | Agent software version (semantic) |
| `name` | string | Display name |
| `description` | string | Detailed description |
| `url` | string | Primary A2A endpoint URL |
| `provider` | object | Provider information |
| `provider.name` | string | Provider name |
| `capabilities` | object | Protocol capabilities |
| `capabilities.a2aVersion` | string | A2A protocol version |
| `authSchemes` | array | Authentication schemes (min 1) |

### 3. Capabilities Structure Change

**Old:** Array of capability objects
```json
"capabilities": [
  {"type": "task", "description": "Task execution"},
  {"type": "streaming", "description": "SSE"}
]
```

**New:** Structured object with specific fields
```json
"capabilities": {
  "a2aVersion": "1.0",
  "mcpVersion": "0.6",
  "supportedMessageParts": ["text", "file", "data"],
  "supportsPushNotifications": true,
  "teeDetails": {...}  // Optional TEE information
}
```

### 4. Skills vs Endpoints

**Old:** `endpoints` array (custom)
```json
"endpoints": [
  {"path": "/a2a/v1/message", "method": "POST", "protocol": "A2A"}
]
```

**New:** `skills` array (official)
```json
"skills": [
  {
    "id": "task_execution",
    "name": "Asynchronous Task Execution",
    "description": "Execute tasks asynchronously...",
    "input_schema": {...},  // JSON Schema
    "output_schema": {...}  // JSON Schema
  }
]
```

### 5. Authentication Schemes (New Required Field)

```json
"authSchemes": [
  {
    "scheme": "apiKey",
    "description": "API key in Authorization header",
    "service_identifier": "myservice"
  },
  {
    "scheme": "oauth2",
    "tokenUrl": "https://auth.example.com/oauth/token",
    "scopes": ["read", "write"]
  },
  {
    "scheme": "bearer",
    "description": "Pre-shared bearer token"
  },
  {
    "scheme": "none",
    "description": "No authentication required"
  }
]
```

## Code Changes

### Files Modified

1. **`internal/a2a/models/agent_card.go`** - Complete restructure
   - Added new official schema types
   - Kept legacy types for backward compatibility
   - Enhanced `UnmarshalJSON` for legacy format support

2. **`internal/a2a/server/agent_card.go`** - Updated generator
   - Now outputs official schema v1.0 format
   - Includes all required fields
   - Maintains legacy fields for compatibility

3. **`internal/a2a/client/discovery.go`** - Updated helpers
   - `ValidateAgentCard()` checks all required fields
   - `HasSkill()` replaces capability checking
   - `Has Capability()` maintained for backward compatibility
   - Added `SupportsAuth()` for auth scheme checking

### New Types

```go
type AgentCard struct {
    // Required
    SchemaVersion    string
    HumanReadableID  string
    AgentVersion     string
    Name             string
    Description      string
    URL              string
    Provider         Provider
    Capabilities     Capabilities
    AuthSchemes      []AuthScheme
    
    // Optional
    Skills            []Skill
    Tags              []string
    PrivacyPolicyURL  string
    TermsOfServiceURL string
    IconURL           string
    LastUpdated       string
    
    // Legacy (deprecated)
    Version      string
    Endpoints    []Endpoint
    Metadata     map[string]any
}

type Provider struct {
    Name           string
    URL            string
    SupportContact string
}

type Capabilities struct {
    A2AVersion                string
    MCPVersion                string
    SupportedMessageParts     []string
    SupportsPushNotifications bool
    TEEDetails                *TEEDetails
}

type TEEDetails struct {
    Type                string
    AttestationEndpoint string
    PublicKey           string
    Description         string
}

type AuthScheme struct {
    Scheme            string
    Description       string
    TokenURL          string
    Scopes            []string
    ServiceIdentifier string
}

type Skill struct {
    ID           string
    Name         string
    Description  string
    InputSchema  map[string]interface{}
    OutputSchema map[string]interface{}
}
```

## Backward Compatibility

### Legacy Format Support

The `UnmarshalJSON` method handles:
1. **New official schema** - Primary format
2. **Legacy array capabilities** - Stored in `metadata.legacyCapabilities`
3. **Legacy object capabilities** - Converted and stored in metadata

### Migration Path

**For Agent Shaker** (as a server):
- ✅ Now outputs official schema v1.0
- ✅ Includes all required fields
- ✅ Maintains legacy fields for old clients

**For External Agents** (as clients):
- ✅ Can discover agents using old format
- ✅ Can discover agents using new format
- ✅ Legacy capabilities accessible via metadata

## Breaking Changes

### API Changes

1. **`card.Capabilities`** is now a struct, not an array
   - Old: `for _, cap := range card.Capabilities`
   - New: `card.Capabilities.A2AVersion`

2. **`card.Version`** deprecated in favor of `card.AgentVersion`
   - Old: `card.Version`
   - New: `card.AgentVersion` (or `card.Version` for legacy)

3. **`HasCapability()`** behavior changed
   - Now checks legacy capabilities in metadata
   - Maps to new schema features where possible

### Required Updates

**Client Code:**
```go
// Old
for _, cap := range card.Capabilities {
    fmt.Println(cap.Type)
}

// New
for _, skill := range card.Skills {
    fmt.Println(skill.ID)
}
```

**Validation:**
```go
// Old
if card.Version == "" {
    return fmt.Errorf("missing version")
}

// New
if card.AgentVersion == "" && card.Version == "" {
    return fmt.Errorf("missing agentVersion")
}
```

## TODO

1. ⏳ Update test files for new schema
2. ⏳ Update documentation
3. ⏳ Add schema validation
4. ⏳ Implement proper authentication schemes (currently only "none")
5. ⏳ Add TEE support if needed
6. ⏳ Update MCP integration to use new schema

## Testing

**Before updating tests:**
```bash
# Currently failing due to schema changes
go test ./tests/a2a -v
```

**After test updates:**
- Tests should validate official schema compliance
- Tests should verify backward compatibility
- Tests should check all required fields

## Agent Shaker's Current Agent Card

```json
{
  "schemaVersion": "1.0",
  "humanReadableId": "techbuzzz/agent-shaker",
  "agentVersion": "1.0.0",
  "name": "Agent Shaker",
  "description": "MCP-compatible context management server with A2A support...",
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
      "description": "Public endpoints require no authentication..."
    }
  ],
  "skills": [
    {
      "id": "task_execution",
      "name": "Asynchronous Task Execution",
      "description": "Execute tasks asynchronously..."
    },
    {
      "id": "sse_streaming",
      "name": "Server-Sent Events Streaming",
      "description": "Real-time task updates..."
    },
    {
      "id": "artifact_sharing",
      "name": "Artifact Sharing",
      "description": "Share markdown contexts..."
    },
    {
      "id": "mcp_integration",
      "name": "MCP Protocol Support",
      "description": "Model Context Protocol support..."
    }
  ],
  "tags": ["agent-coordination", "task-management", "mcp", "a2a", ...]
}
```

## References

- **Official Schema**: https://gist.githubusercontent.com/SecureAgentTools/0815a2de9cc31c71468afd3d2eef260a/raw/agent-card-schema.md
- **A2A Protocol**: https://a2a-protocol.org/
- **Agent Shaker Docs**: See `docs/A2A_INTEGRATION.md`

## Status

✅ Schema compliance implemented  
✅ Code compiles successfully  
⏳ Tests need updating  
⏳ Documentation needs updating  
⏳ Production auth schemes need implementation  

---

**Next Steps:**
1. Update test files to work with new schema
2. Test with external agents (Hello World Agent)
3. Update all documentation references
4. Consider implementing proper authentication
