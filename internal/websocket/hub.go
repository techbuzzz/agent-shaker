package websocket

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/techbuzzz/agent-shaker/internal/models"
)

// newUpgrader builds a websocket.Upgrader with a strict, env-driven origin allow-list.
// Origins are matched against the request's Origin header. By default only same-host
// (empty Origin, http://localhost:* , http://127.0.0.1:*) origins are accepted.
func newUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     checkOrigin(defaultAllowedOrigins()),
	}
}

// defaultAllowedOrigins reads the WS_ALLOWED_ORIGINS env var (comma-separated) and
// merges it with the built-in localhost defaults. An empty env value keeps the
// defaults; a "*" entry allows any origin (development only).
func defaultAllowedOrigins() []string {
	envVal := os.Getenv("WS_ALLOWED_ORIGINS")
	if envVal == "*" {
		return []string{"*"}
	}
	defaults := []string{
		"http://localhost",
		"http://127.0.0.1",
	}
	if envVal == "" {
		return defaults
	}
	merged := make([]string, 0, len(defaults)+8)
	merged = append(merged, defaults...)
	for _, o := range strings.Split(envVal, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			merged = append(merged, o)
		}
	}
	return merged
}

// checkOrigin returns an http.Handler-friendly CheckOrigin that matches the
// request Origin header against the supplied allow-list. Empty Origin (same-origin
// request) is always accepted so curl, Postman, and server-to-server traffic work.
func checkOrigin(allowed []string) func(r *http.Request) bool {
	allowAll := false
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		if o == "*" {
			allowAll = true
			continue
		}
		set[strings.ToLower(o)] = struct{}{}
	}
	return func(r *http.Request) bool {
		if allowAll {
			return true
		}
		origin := strings.ToLower(r.Header.Get("Origin"))
		if origin == "" {
			return true
		}
		if _, ok := set[origin]; ok {
			return true
		}
		// Allow sub-paths of a registered origin (e.g. http://localhost:3000 vs http://localhost).
		for allowed := range set {
			if strings.HasPrefix(origin, allowed+":") || origin == allowed {
				return true
			}
		}
		return false
	}
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	ID        string
	ProjectID uuid.UUID
	Conn      *websocket.Conn
	Send      chan []byte
	hub       *Hub
}

type Hub struct {
	clients    map[string]*Client
	projects   map[uuid.UUID]map[string]*Client
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	done       chan struct{}
	closeOnce  sync.Once
	wg         sync.WaitGroup
	mu         sync.Mutex // serializes broadcasts + register/unregister; client send queues are per-channel
	upgrader   *websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		projects:   make(map[uuid.UUID]map[string]*Client),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
		upgrader:   newUpgrader(),
	}
}

// Run drives the hub event loop until Shutdown is called.
func (h *Hub) Run() {
	h.wg.Add(1)
	defer h.wg.Done()
	for {
		select {
		case <-h.done:
			return
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

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				if projectClients, ok := h.projects[client.ProjectID]; ok {
					delete(projectClients, client.ID)
					if len(projectClients) == 0 {
						delete(h.projects, client.ProjectID)
					}
				}
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Client %s unregistered", client.ID)

		case message, ok := <-h.broadcast:
			if !ok {
				return
			}
			h.broadcastMessage(message)
		}
	}
}

// Shutdown closes the hub event loop, waits for Run to exit, then closes every
// remaining client's send channel so pumps terminate cleanly.
func (h *Hub) Shutdown() {
	h.closeOnce.Do(func() {
		close(h.done)
	})
	h.wg.Wait()
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, client := range h.clients {
		close(client.Send)
		delete(h.clients, id)
	}
	for pid, projectClients := range h.projects {
		for id := range projectClients {
			delete(projectClients, id)
		}
		delete(h.projects, pid)
	}
}

func (h *Hub) broadcastMessage(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal message: %v", err)
		return
	}

	// Extract project ID from payload
	var projectID uuid.UUID

	// Try to extract from different payload types
	switch payload := message.Payload.(type) {
	case map[string]interface{}:
		if pid, ok := payload["project_id"].(string); ok {
			projectID, _ = uuid.Parse(pid)
		}
	default:
		// Try to use reflection to get ProjectID field
		// This handles structs like models.Task, models.Agent, etc.
		payloadBytes, err := json.Marshal(payload)
		if err == nil {
			var temp map[string]interface{}
			if json.Unmarshal(payloadBytes, &temp) == nil {
				if pid, ok := temp["project_id"].(string); ok {
					projectID, _ = uuid.Parse(pid)
				}
			}
		}
	}

	// Hold the write lock for the whole broadcast so map mutations on overflow
	// (delete of slow clients) are race-free against Register/Unregister.
	h.mu.Lock()
	defer h.mu.Unlock()

	projectClients, ok := h.projects[projectID]
	if !ok {
		return
	}
	for id, client := range projectClients {
		select {
		case client.Send <- data:
		default:
			// Slow client: close its send channel, drop it from both maps.
			close(client.Send)
			delete(projectClients, id)
			delete(h.clients, client.ID)
		}
	}
	if len(projectClients) == 0 {
		delete(h.projects, projectID)
	}
}

func (h *Hub) BroadcastToProject(projectID uuid.UUID, messageType string, payload interface{}) {
	message := &Message{
		Type:    messageType,
		Payload: payload,
	}
	select {
	case h.broadcast <- message:
	case <-h.done:
	}
}

// BroadcastTaskUpdate sends a task update to all connected clients
func (h *Hub) BroadcastTaskUpdate(update *models.TaskUpdate) {
	message := &Message{
		Type:    "task_update",
		Payload: update,
	}
	select {
	case h.broadcast <- message:
	case <-h.done:
	}
}

func (h *Hub) Register(client *Client) {
	select {
	case h.register <- client:
	case <-h.done:
	}
}

func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
	case <-h.done:
	}
}

// HandleWebSocket handles WebSocket connections. When OpenTelemetry tracing
// is enabled, the handshake opens a span ("websocket.handle") that the
// hub pumps link to via their own spans (added in a follow-up).
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("agent-shaker/websocket")
	ctx, span := tracer.Start(r.Context(), "websocket.handle")
	defer span.End()

	conn, err := h.upgrader.Upgrade(w, r.WithContext(ctx), nil)
	if err != nil {
		slog.ErrorContext(ctx, "websocket upgrade error", "error", err)
		return
	}

	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		projectIDStr = "00000000-0000-0000-0000-000000000000"
	}

	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		slog.ErrorContext(ctx, "invalid project_id", "error", err, "raw", projectIDStr)
		conn.Close()
		return
	}
	span.SetAttributes(
		attribute.String("ws.project_id", projectID.String()),
		attribute.String("ws.client_id", ""),
	)

	client := &Client{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		hub:       h,
	}
	span.SetAttributes(attribute.String("ws.client_id", client.ID))

	h.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Conn.Close()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}
	}
}

func (c *Client) WritePump() {
	defer c.Conn.Close()

	for {
		message, ok := <-c.Send
		if !ok {
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}
