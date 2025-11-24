package udp

import (
	"encoding/json"
	"log"
	"net"
	"sync"
	"time"
)

// Notification represents a UDP notification message
type Notification struct {
	Type      string      `json:"type"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// RegisteredClient represents a registered UDP client
type RegisteredClient struct {
	Address  *net.UDPAddr
	LastSeen time.Time
	UserID   string
}

// Server represents the UDP broadcast server
type Server struct {
	Address      string
	clients      map[string]*RegisteredClient
	mutex        sync.RWMutex
	conn         *net.UDPConn
	done         chan bool
	broadcastIP  string
	broadcastPort int
}

// NewServer creates a new UDP server
func NewServer(address, broadcastIP string, broadcastPort int) *Server {
	return &Server{
		Address:       address,
		clients:       make(map[string]*RegisteredClient),
		done:          make(chan bool),
		broadcastIP:   broadcastIP,
		broadcastPort: broadcastPort,
	}
}

// Start starts the UDP server
func (s *Server) Start() error {
	addr, err := net.ResolveUDPAddr("udp", s.Address)
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	s.conn = conn

	log.Printf("UDP Server listening on %s", s.Address)

	go s.handleMessages()
	go s.cleanupInactiveClients()

	return nil
}

// Stop stops the UDP server
func (s *Server) Stop() {
	close(s.done)
	if s.conn != nil {
		s.conn.Close()
	}

	s.mutex.Lock()
	s.clients = make(map[string]*RegisteredClient)
	s.mutex.Unlock()

	log.Println("UDP Server stopped")
}

func (s *Server) handleMessages() {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-s.done:
			return
		default:
			s.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, clientAddr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				log.Printf("Error reading UDP message: %v", err)
				continue
			}

			var msg map[string]interface{}
			if err := json.Unmarshal(buffer[:n], &msg); err != nil {
				log.Printf("Error unmarshaling UDP message: %v", err)
				continue
			}

			s.handleMessage(msg, clientAddr)
		}
	}
}

func (s *Server) handleMessage(msg map[string]interface{}, clientAddr *net.UDPAddr) {
	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	clientKey := clientAddr.String()

	switch msgType {
	case "register":
		userID, _ := msg["user_id"].(string)
		s.mutex.Lock()
		s.clients[clientKey] = &RegisteredClient{
			Address:  clientAddr,
			LastSeen: time.Now(),
			UserID:   userID,
		}
		s.mutex.Unlock()

		response := Notification{
			Type:      "registered",
			Message:   "Successfully registered for notifications",
			Timestamp: time.Now().Format(time.RFC3339),
		}
		s.sendNotification(response, clientAddr)
		log.Printf("UDP client registered: %s (UserID: %s)", clientKey, userID)

	case "heartbeat":
		s.mutex.Lock()
		if client, exists := s.clients[clientKey]; exists {
			client.LastSeen = time.Now()
		}
		s.mutex.Unlock()
	}
}

func (s *Server) sendNotification(notification Notification, addr *net.UDPAddr) {
	data, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling notification: %v", err)
		return
	}

	// Set write deadline for UDP send
	s.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	_, err = s.conn.WriteToUDP(data, addr)
	if err != nil {
		// Check for network errors
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("UDP send timeout to %s: %v", addr.String(), err)
		} else {
			log.Printf("Error sending notification to %s: %v", addr.String(), err)
		}
		// Remove client on persistent network failure
		clientKey := addr.String()
		s.mutex.Lock()
		if _, exists := s.clients[clientKey]; exists {
			delete(s.clients, clientKey)
			log.Printf("Removed UDP client due to send failure: %s", clientKey)
		}
		s.mutex.Unlock()
	}
}

// Broadcast sends a notification to all registered clients
func (s *Server) Broadcast(notification Notification) {
	s.mutex.RLock()
	clients := make([]*RegisteredClient, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.mutex.RUnlock()

	if len(clients) == 0 {
		log.Printf("No clients registered for UDP broadcast")
		return
	}

	data, err := json.Marshal(notification)
	if err != nil {
		log.Printf("Error marshaling broadcast notification: %v", err)
		return
	}

	successCount := 0
	failedClients := make([]string, 0)

	for _, client := range clients {
		s.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		_, err := s.conn.WriteToUDP(data, client.Address)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("UDP broadcast timeout to %s", client.Address.String())
			} else {
				log.Printf("Error broadcasting to %s: %v", client.Address.String(), err)
			}
			failedClients = append(failedClients, client.Address.String())
		} else {
			successCount++
		}
	}

	// Remove failed clients
	if len(failedClients) > 0 {
		s.mutex.Lock()
		for _, addr := range failedClients {
			delete(s.clients, addr)
		}
		s.mutex.Unlock()
		log.Printf("Removed %d failed UDP clients from broadcast list", len(failedClients))
	}

	log.Printf("Broadcasted notification to %d/%d clients successfully", successCount, len(clients))
}

// BroadcastNewManga broadcasts a new manga notification
func (s *Server) BroadcastNewManga(mangaID, title string) {
	notification := Notification{
		Type:      "new_manga",
		Message:   "New manga added: " + title,
		Data:      map[string]string{"manga_id": mangaID, "title": title},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	s.Broadcast(notification)
}

// BroadcastUpdate broadcasts a general update notification
func (s *Server) BroadcastUpdate(message string, data interface{}) {
	notification := Notification{
		Type:      "update",
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}
	s.Broadcast(notification)
}

func (s *Server) cleanupInactiveClients() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mutex.Lock()
			now := time.Now()
			for key, client := range s.clients {
				if now.Sub(client.LastSeen) > 2*time.Minute {
					delete(s.clients, key)
					log.Printf("Removed inactive UDP client: %s", key)
				}
			}
			s.mutex.Unlock()
		}
	}
}

// GetClientCount returns the number of registered clients
func (s *Server) GetClientCount() int {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return len(s.clients)
}

