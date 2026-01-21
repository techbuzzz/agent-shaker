package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type Client struct {
	ID        string
	ProjectID uuid.UUID
	Conn      *websocket.Conn
	Send      chan []byte
}

type Hub struct {
	clients    map[string]*Client
	projects   map[uuid.UUID]map[string]*Client
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		projects:   make(map[uuid.UUID]map[string]*Client),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

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

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

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

	// Broadcast to all clients of the project
	if projectClients, ok := h.projects[projectID]; ok {
		for _, client := range projectClients {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.clients, client.ID)
				delete(projectClients, client.ID)
			}
		}
	}
}

func (h *Hub) BroadcastToProject(projectID uuid.UUID, messageType string, payload interface{}) {
	message := &Message{
		Type:    messageType,
		Payload: payload,
	}
	h.broadcast <- message
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.Unregister(c)
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
