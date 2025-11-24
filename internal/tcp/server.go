package tcp

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// Message represents a JSON message protocol
type Message struct {
	Type      string      `json:"type"`
	UserID    string      `json:"user_id,omitempty"`
	MangaID   string      `json:"manga_id,omitempty"`
	Chapter   int         `json:"chapter,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// Client represents a TCP client connection
type Client struct {
	ID       string
	Conn     net.Conn
	UserID   string
	LastSeen time.Time
}

// Server represents the TCP server
type Server struct {
	Address  string
	clients  map[string]*Client
	mutex    sync.RWMutex
	listener net.Listener
	done     chan bool
}

// NewServer creates a new TCP server
func NewServer(address string) *Server {
	return &Server{
		Address: address,
		clients: make(map[string]*Client),
		done:    make(chan bool),
	}
}

// Start starts the TCP server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.Address)
	if err != nil {
		return err
	}
	s.listener = listener

	log.Printf("TCP Server listening on %s", s.Address)

	go s.acceptConnections()
	return nil
}

// Stop stops the TCP server
func (s *Server) Stop() {
	close(s.done)
	if s.listener != nil {
		s.listener.Close()
	}

	s.mutex.Lock()
	for _, client := range s.clients {
		client.Conn.Close()
	}
	s.clients = make(map[string]*Client)
	s.mutex.Unlock()

	log.Println("TCP Server stopped")
}

func (s *Server) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				log.Printf("Error accepting connection: %v", err)
				continue
			}
		}

		clientID := conn.RemoteAddr().String()
		client := &Client{
			ID:       clientID,
			Conn:     conn,
			LastSeen: time.Now(),
		}

		s.mutex.Lock()
		s.clients[clientID] = client
		s.mutex.Unlock()

		log.Printf("New TCP client connected: %s", clientID)

		go s.handleClient(client)
	}
}

func (s *Server) handleClient(client *Client) {
	defer func() {
		s.mutex.Lock()
		delete(s.clients, client.ID)
		s.mutex.Unlock()
		if err := client.Conn.Close(); err != nil {
			log.Printf("Error closing connection for %s: %v", client.ID, err)
		}
		log.Printf("TCP client disconnected: %s", client.ID)
	}()

	// Set connection deadline for read operations
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	
	decoder := json.NewDecoder(client.Conn)
	encoder := json.NewEncoder(client.Conn)

	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			// Check if it's a network error or timeout
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("TCP client %s read timeout, disconnecting", client.ID)
			} else if err == io.EOF {
				log.Printf("TCP client %s closed connection", client.ID)
			} else {
				log.Printf("Error decoding message from %s: %v", client.ID, err)
			}
			return
		}

		// Reset read deadline after successful read
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		client.LastSeen = time.Now()
		msg.Timestamp = time.Now().Format(time.RFC3339)

		// Handle different message types
		switch msg.Type {
		case "register":
			client.UserID = msg.UserID
			response := Message{
				Type:      "registered",
				UserID:    client.UserID,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			if err := encoder.Encode(response); err != nil {
				log.Printf("Error encoding register response to %s: %v", client.ID, err)
				return
			}

		case "progress_update":
			// Broadcast progress update to all clients
			s.broadcastMessage(msg, client.ID)
			response := Message{
				Type:      "progress_ack",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			if err := encoder.Encode(response); err != nil {
				log.Printf("Error encoding progress_ack to %s: %v", client.ID, err)
				return
			}

		case "ping":
			response := Message{
				Type:      "pong",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			if err := encoder.Encode(response); err != nil {
				log.Printf("Error encoding pong to %s: %v", client.ID, err)
				return
			}

		default:
			response := Message{
				Type:      "error",
				Data:      "Unknown message type",
				Timestamp: time.Now().Format(time.RFC3339),
			}
			if err := encoder.Encode(response); err != nil {
				log.Printf("Error encoding error response to %s: %v", client.ID, err)
				return
			}
		}
	}
}

func (s *Server) broadcastMessage(msg Message, excludeID string) {
	s.mutex.RLock()
	clients := make([]*Client, 0, len(s.clients))
	for id, client := range s.clients {
		if id != excludeID {
			clients = append(clients, client)
		}
	}
	s.mutex.RUnlock()

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	// Send to all clients with error handling
	for _, client := range clients {
		client.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_, err := client.Conn.Write(append(data, '\n'))
		if err != nil {
			log.Printf("Error sending message to client %s: %v", client.ID, err)
			// Remove failed client
			s.mutex.Lock()
			delete(s.clients, client.ID)
			s.mutex.Unlock()
			client.Conn.Close()
		}
	}
}

// BroadcastProgress broadcasts a progress update to all connected clients
func (s *Server) BroadcastProgress(userID, mangaID string, chapter int) {
	msg := Message{
		Type:      "progress_broadcast",
		UserID:    userID,
		MangaID:   mangaID,
		Chapter:   chapter,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	s.broadcastMessage(msg, "")
}

// GetClientCount returns the number of connected clients
func (s *Server) GetClientCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.clients)
}

