# A2A Agent Card Compatibility Fix

## Issue

When discovering external A2A agents, some implementations return the `capabilities` field in a different format than expected, causing the following error:

```
{"error": "Failed to discover agent: failed to decode agent card: json: cannot unmarshal object into Go struct field AgentCard.capabilities of type []models.Capability"}
```

## Root Cause

The A2A Protocol specification allows for flexibility in how agent capabilities are expressed. Some implementations use:

**Standard Array Format:**
```json
{
  "capabilities": [
    {"type": "task", "description": "Task execution"},
    {"type": "streaming", "description": "SSE streaming"}
  ]
}
```

**Alternative Object Format:**
```json
{
  "capabilities": {
    "task": "Task execution",
    "streaming": "SSE streaming"
  }
}
```

Agent Shaker was only supporting the array format, causing unmarshaling errors when encountering the object format.

## Solution

Implemented custom JSON unmarshaling logic in `AgentCard` that:

1. **Attempts array format first** (standard)
2. **Falls back to object format** if array fails
3. **Handles multiple value types** in object format (string, boolean, number)
4. **Converts object format to array** internally for consistent handling
5. **Gracefully handles invalid formats** by returning an empty capabilities array

### Real-World Testing

Tested with actual external A2A agent at `http://127.0.0.1:9001`:

**External Agent Response:**
```json
{
  "capabilities": {
    "streaming": true
  },
  "name": "Hello World Agent",
  "description": "Just a hello world agent",
  "version": ""
}
```

**Agent Shaker Successfully Converts To:**
```json
{
  "name": "Hello World Agent",
  "description": "Just a hello world agent",
  "version": "",
  "capabilities": [
    {
      "type": "streaming",
      "description": "true"
    }
  ]
}
```

✅ **Discovery works perfectly** - boolean value `true` converted to string `"true"`  
✅ **Empty version field handled** gracefully  
✅ **Agent is fully usable** despite format differences

### Code Changes

**File:** `internal/a2a/models/agent_card.go`

Added `UnmarshalJSON` method to `AgentCard` with support for **multiple value types**:

```go
func (a *AgentCard) UnmarshalJSON(data []byte) error {
    // First try array format (standard)
    var capArray []Capability
    if err := json.Unmarshal(aux.CapabilitiesRaw, &capArray); err == nil {
        a.Capabilities = capArray
        return nil
    }

    // Try object format with multiple value types
    var capMap map[string]interface{}
    if err := json.Unmarshal(aux.CapabilitiesRaw, &capMap); err == nil {
        a.Capabilities = make([]Capability, 0, len(capMap))
        for capType, value := range capMap {
            var description string
            switch v := value.(type) {
            case string:
                description = v
            case bool:
                description = fmt.Sprintf("%v", v) // "true" or "false"
            case float64:
                description = fmt.Sprintf("%v", v)
            default:
                // Marshal complex types to JSON string
                if jsonBytes, err := json.Marshal(v); err == nil {
                    description = string(jsonBytes)
                }
            }
            a.Capabilities = append(a.Capabilities, Capability{
                Type:        capType,
                Description: description,
            })
        }
        return nil
    }

    // Graceful fallback
    a.Capabilities = []Capability{}
    return nil
}
```

**Key Enhancement:** Now handles boolean, number, and complex types in addition to strings.

## Testing

Created comprehensive tests in `tests/a2a/agent_card_test.go` and `tests/a2a/external_agent_test.go`:

✅ **TestAgentCardUnmarshal_ArrayFormat** - Verifies standard array format  
✅ **TestAgentCardUnmarshal_ObjectFormat** - Verifies alternative object format (string values)  
✅ **TestAgentCardUnmarshal_BooleanCapabilities** - Verifies boolean value handling  
✅ **TestAgentCardUnmarshal_InvalidFormat** - Verifies graceful handling of invalid formats  
✅ **TestAgentCardUnmarshal_MissingCapabilities** - Verifies handling of missing capabilities field  
✅ **TestAgentCardMarshal** - Verifies marshaling always produces standard array format  
✅ **TestRealWorldAgentCard_HelloWorldAgent** - Parses actual external agent response  
✅ **TestDiscoverExternalAgent_Integration** - Live discovery test with real agent at port 9001  

All 16 tests pass:

```
=== RUN   TestRealWorldAgentCard_HelloWorldAgent
    external_agent_test.go:83: Successfully parsed real-world agent card:
    external_agent_test.go:84:   Name: Hello World Agent
    external_agent_test.go:85:   Description: Just a hello world agent
    external_agent_test.go:86:   Version:
    external_agent_test.go:87:   Capabilities: 1 converted from object format
    external_agent_test.go:89:     - streaming: true
--- PASS: TestRealWorldAgentCard_HelloWorldAgent (0.00s)

=== RUN   TestDiscoverExternalAgent_Integration
    external_agent_test.go:155: Successfully discovered external agent:
    external_agent_test.go:156:   Name: Hello World Agent
    external_agent_test.go:159:   Capabilities: 1
    external_agent_test.go:162:     - streaming: true
    external_agent_test.go:169: Note: Agent has empty version field (non-fatal)
    external_agent_test.go:177: ✓ Agent has streaming capability
--- PASS: TestDiscoverExternalAgent_Integration (0.00s)

PASS
ok      github.com/techbuzzz/agent-shaker/tests/a2a     0.734s
```

**Manual Testing:**

Created `cmd/test-discovery/main.go` for manual verification:

```
Testing A2A Agent Discovery with External Agent
=================================================

1. Discovering Agent Shaker (localhost:8080)...
   ✓ Success!
   Name: Agent Shaker
   Version: 1.0.0
   Capabilities: 4

2. Discovering Hello World Agent (127.0.0.1:9001)...
   ✓ Success!
   Name: Hello World Agent
   Capabilities: 1 (converted from object format)
     - streaming: true
   ✓ Agent supports streaming

=================================================
All tests completed!
```

## Behavior

### Before Fix

```bash
# Discovering agent with object-format capabilities
curl https://external-agent/.well-known/agent-card.json
# Returns object format

# Discovery fails
{"error": "Failed to discover agent: failed to decode agent card: json: cannot unmarshal object into Go struct field AgentCard.capabilities of type []models.Capability"}
```

### After Fix

```bash
# Discovering agent with object-format capabilities
curl https://external-agent/.well-known/agent-card.json
# Returns object format

# Discovery succeeds - automatically converts to array format
{
  "success": true,
  "name": "External Agent",
  "capabilities": [
    {"type": "task", "description": "Task execution"},
    {"type": "streaming", "description": "SSE streaming"}
  ]
}
```

## Compatibility

- ✅ **Backward Compatible** - Standard array format still works
- ✅ **Forward Compatible** - Alternative object format now supported
- ✅ **Graceful Degradation** - Invalid formats result in empty capabilities array, not errors
- ✅ **Consistent Output** - Agent Shaker always outputs standard array format when marshaling

## Documentation Updates

Updated `docs/A2A_INTEGRATION.md`:

1. Added explanation of both capability formats in **Data Models** section
2. Added troubleshooting entry for the unmarshaling error
3. Documented the automatic format conversion behavior

## Usage Example

### From VS Code with Copilot

```
@workspace Using agent-shaker, discover the A2A agent at https://external-agent.example.com
```

**Now works with both formats:**

- External agent returns object format → ✅ Automatically converted to array
- External agent returns array format → ✅ Used as-is
- External agent returns invalid format → ✅ Empty capabilities array (non-fatal)

## MCP Tool Impact

The `discover_a2a_agent` MCP tool now handles both formats transparently:

```json
{
  "tool": "discover_a2a_agent",
  "arguments": {
    "agent_url": "https://any-agent.example.com"
  }
}
```

**Result:** Always returns standardized agent card with capabilities as array, regardless of the source format.

## Related Files

- `internal/a2a/models/agent_card.go` - Core model with custom unmarshaling
- `tests/a2a/agent_card_test.go` - Comprehensive test suite
- `docs/A2A_INTEGRATION.md` - Updated documentation
- `internal/a2a/client/discovery.go` - Discovery client (no changes needed)

## Verification

To verify the fix is working:

1. **Test with standard format:**
   ```bash
   go test ./tests/a2a -run TestAgentCardUnmarshal_ArrayFormat -v
   ```

2. **Test with object format:**
   ```bash
   go test ./tests/a2a -run TestAgentCardUnmarshal_ObjectFormat -v
   ```

3. **Test discovery with live agent:**
   ```bash
   # Start Agent Shaker
   go run cmd/server/main.go
   
   # Discover itself (uses standard array format)
   curl http://localhost:8080/.well-known/agent-card.json
   ```

4. **Test from VS Code:**
   ```
   @workspace Using agent-shaker, discover the agent at http://localhost:8080
   ```

## Benefits

1. **Improved Interoperability** - Works with more A2A agent implementations
2. **Better Error Handling** - Graceful degradation instead of hard failures
3. **Clear Documentation** - Developers know both formats are supported
4. **Comprehensive Testing** - Edge cases covered
5. **User Experience** - Seamless discovery regardless of external agent format

## Status

✅ **Fixed and Tested**  
✅ **Documented**  
✅ **All Tests Passing**  
✅ **Ready for Production**
