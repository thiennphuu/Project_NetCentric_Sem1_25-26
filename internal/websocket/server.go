package websocket

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	UserID    string      `json:"user_id,omitempty"`
	Username  string      `json:"username,omitempty"`
	Content   string      `json:"content,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	Conn     *websocket.Conn
	UserID   string
	Username string
	Send     chan Message
	Hub      *Hub
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Printf("WebSocket client connected: %s (Total: %d)", client.ID, len(h.clients))

			// Notify others about new user
			welcomeMsg := Message{
				Type:      "user_joined",
				UserID:    client.UserID,
				Username:  client.Username,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			h.broadcastToOthers(welcomeMsg, client)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()
			log.Printf("WebSocket client disconnected: %s (Total: %d)", client.ID, len(h.clients))

			// Notify others about user leaving
			leaveMsg := Message{
				Type:      "user_left",
				UserID:    client.UserID,
				Username:  client.Username,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			h.broadcastToOthers(leaveMsg, nil)

		case message := <-h.broadcast:
			h.mutex.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mutex.RUnlock()

			// Broadcast to all clients with error handling
			for _, client := range clients {
				select {
				case client.Send <- message:
				default:
					// Channel full, remove client
					log.Printf("WebSocket client %s send channel full during broadcast, removing", client.ID)
					h.mutex.Lock()
					if _, exists := h.clients[client]; exists {
						delete(h.clients, client)
						close(client.Send)
					}
					h.mutex.Unlock()
				}
			}
		}
	}
}

func (h *Hub) broadcastToOthers(message Message, exclude *Client) {
	h.mutex.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		if exclude == nil || client != exclude {
			clients = append(clients, client)
		}
	}
	h.mutex.RUnlock()

	// Send to clients with error handling
	for _, client := range clients {
		select {
		case client.Send <- message:
		default:
			// Channel full or client disconnected
			log.Printf("WebSocket client %s send channel full, removing", client.ID)
			h.mutex.Lock()
			if _, exists := h.clients[client]; exists {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	clientID := r.RemoteAddr
	client := &Client{
		ID:   clientID,
		Conn: conn,
		Send: make(chan Message, 256),
		Hub:  hub,
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		msg.Timestamp = time.Now().Format(time.RFC3339)

		switch msg.Type {
		case "register":
			c.UserID = msg.UserID
			c.Username = msg.Username
			response := Message{
				Type:      "registered",
				UserID:    c.UserID,
				Username:  c.Username,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			c.Send <- response

		case "chat":
			// Broadcast chat message to all clients
			broadcastMsg := Message{
				Type:      "chat",
				UserID:    c.UserID,
				Username:  c.Username,
				Content:   msg.Content,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			c.Hub.broadcast <- broadcastMsg

		case "ping":
			response := Message{
				Type:      "pong",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			c.Send <- response

		default:
			response := Message{
				Type:      "error",
				Content:   "Unknown message type",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			c.Send <- response
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		if err := c.Conn.Close(); err != nil {
			log.Printf("Error closing WebSocket connection for %s: %v", c.ID, err)
		}
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed, send close message
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket write error for %s: %v", c.ID, err)
				}
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("WebSocket ping error for %s: %v", c.ID, err)
				return
			}
		}
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

