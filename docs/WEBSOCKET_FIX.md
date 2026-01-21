# WebSocket 500 Error Fix

## Issue
WebSocket connections were failing with HTTP 500 error during handshake:
```
Error during WebSocket handshake: Unexpected response code: 500
```

## Root Cause
The `Client` struct in the WebSocket package has a `hub` field that references the parent Hub, but this field was never being set when clients were registered. When `ReadPump()` tried to call `c.hub.Unregister(c)`, it caused a nil pointer dereference, resulting in a 500 error.

### Code Flow:
1. `HandleWebSocket` creates a new `Client` struct but doesn't set the `hub` field
2. `h.hub.Register(client)` sends the client to a channel
3. The Hub's `Run()` goroutine receives the client and adds it to maps
4. **Problem:** The `hub` field is never set
5. When `ReadPump()` runs and tries to unregister: `c.hub.Unregister(c)` → **nil pointer panic → 500 error**

## Solution
Set the `hub` field on the client during registration in the Hub's `Run()` method.

### Changed File: `internal/websocket/hub.go`

**Before:**
```go
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			if h.projects[client.ProjectID] == nil {
				h.projects[client.ProjectID] = make(map[string]*Client)
			}
			h.projects[client.ProjectID][client.ID] = client
			h.mu.Unlock()
			log.Printf("Client %s registered for project %s", client.ID, client.ProjectID)
```

**After:**
```go
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			client.hub = h // Set the hub reference
			h.clients[client.ID] = client
			if h.projects[client.ProjectID] == nil {
				h.projects[client.ProjectID] = make(map[string]*Client)
			}
			h.projects[client.ProjectID][client.ID] = client
			h.mu.Unlock()
			log.Printf("Client %s registered for project %s", client.ID, client.ProjectID)
```

## Testing
After the fix, WebSocket connections should work properly:

1. **Start the backend:**
   ```powershell
   .\bin\mcp-server.exe
   ```

2. **Test WebSocket connection:**
   ```javascript
   // In browser console or frontend
   const ws = new WebSocket('ws://localhost:8080/ws?project_id=550e8400-e29b-41d4-a716-446655440001');
   
   ws.onopen = () => console.log('WebSocket connected!');
   ws.onerror = (error) => console.error('WebSocket error:', error);
   ws.onmessage = (event) => console.log('Received:', event.data);
   ```

3. **Expected behavior:**
   - Connection should succeed (no 500 error)
   - Client should be registered successfully
   - Real-time updates should be received when projects/agents/tasks change

## Architecture Notes

### Client Lifecycle:
1. **Connection:** `HandleWebSocket` upgrades HTTP to WebSocket
2. **Registration:** Client sent to `h.register` channel
3. **Processing:** Hub's `Run()` goroutine processes registration
4. **Operation:** `ReadPump()` and `WritePump()` goroutines run concurrently
5. **Disconnection:** `ReadPump()` detects close and calls `Unregister()`
6. **Cleanup:** Hub removes client from maps and closes channels

### Key Components:
- **Hub:** Central manager for all WebSocket clients
- **Client:** Represents a single WebSocket connection
- **Channels:** 
  - `register`: Queue for new clients
  - `unregister`: Queue for disconnected clients
  - `broadcast`: Queue for messages to send
  - `Send`: Per-client outgoing message queue

### Thread Safety:
- Hub uses `sync.RWMutex` for concurrent map access
- Registration/unregistration happens in single goroutine (Run loop)
- Each client has dedicated read/write goroutines

## Related Files
- `internal/websocket/hub.go` - WebSocket hub implementation
- `internal/websocket/websocket.go` - WebSocket pump implementations (legacy)
- `internal/handlers/websocket.go` - HTTP handler for WebSocket upgrades
- `cmd/server/main.go` - Server initialization (starts Hub.Run())

## Future Improvements
1. Add WebSocket connection authentication/authorization
2. Implement heartbeat/keepalive for connection health
3. Add reconnection logic on the client side
4. Implement message acknowledgments
5. Add rate limiting for messages
6. Consider using structured message types (protobuf/msgpack)
7. Add metrics for WebSocket connections
8. Clean up duplicate pump method implementations
