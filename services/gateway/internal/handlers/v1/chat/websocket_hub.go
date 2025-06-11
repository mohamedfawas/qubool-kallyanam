package chat

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[string]*Client

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Inbound messages from the clients
	broadcast chan *Message

	// Logger
	logger logging.Logger

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// Client is a middleman between the websocket connection and the hub
type Client struct {
	hub *Hub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan *Message

	// User ID
	userID string

	// Current conversation ID (if in a conversation)
	conversationID string

	// Last activity time
	lastActivity time.Time
}

// Message represents a chat message
type Message struct {
	Type           string      `json:"type"`
	ConversationID string      `json:"conversation_id,omitempty"`
	MessageID      string      `json:"message_id,omitempty"`
	SenderID       string      `json:"sender_id,omitempty"`
	Text           string      `json:"text,omitempty"`
	SentAt         time.Time   `json:"sent_at,omitempty"`
	Data           interface{} `json:"data,omitempty"`
	Error          string      `json:"error,omitempty"`
}

// Message types
const (
	MessageTypeJoin        = "join"
	MessageTypeLeave       = "leave"
	MessageTypeMessage     = "message"
	MessageTypeTyping      = "typing"
	MessageTypeStopTyping  = "stop_typing"
	MessageTypeError       = "error"
	MessageTypeAck         = "ack"
	MessageTypeUserOnline  = "user_online"
	MessageTypeUserOffline = "user_offline"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// NewHub creates a new Hub
func NewHub(logger logging.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		clients:    make(map[string]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	h.logger.Info("WebSocket hub started")

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-ticker.C:
			h.cleanupInactiveClients()

		case <-h.ctx.Done():
			h.logger.Info("WebSocket hub shutting down")
			h.cleanupAllClients()
			return
		}
	}
}

// Stop gracefully stops the hub
func (h *Hub) Stop() {
	h.cancel()
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close existing connection if user is already connected
	if existingClient, exists := h.clients[client.userID]; exists {
		h.logger.Info("Closing existing connection for user", "userID", client.userID)
		close(existingClient.send)
		existingClient.conn.Close()
	}

	h.clients[client.userID] = client
	h.logger.Info("Client registered", "userID", client.userID, "totalClients", len(h.clients))

	// Send registration confirmation to the client
	confirmMessage := &Message{
		Type:   MessageTypeAck,
		SentAt: time.Now(),
		Data: map[string]interface{}{
			"action":  "connected",
			"user_id": client.userID,
		},
	}

	select {
	case client.send <- confirmMessage:
	default:
		h.logger.Error("Failed to send registration confirmation", "userID", client.userID)
	}
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.clients[client.userID]; exists {
		delete(h.clients, client.userID)
		close(client.send)
		h.logger.Info("Client unregistered", "userID", client.userID, "totalClients", len(h.clients))

		// Notify others that user is offline only if they were in a conversation
		if client.conversationID != "" {
			offlineMessage := &Message{
				Type:           MessageTypeUserOffline,
				ConversationID: client.conversationID,
				SenderID:       client.userID,
				SentAt:         time.Now(),
			}
			// We can't call broadcastToConversationParticipants here because it would cause deadlock
			// Instead, we'll send this through the broadcast channel
			go func() {
				select {
				case h.broadcast <- offlineMessage:
				case <-time.After(time.Second):
					h.logger.Error("Failed to broadcast offline message - timeout")
				}
			}()
		}
	}
}

func (h *Hub) broadcastMessage(message *Message) {
	if message.ConversationID != "" {
		h.broadcastToConversationParticipants(message.ConversationID, message, "")
	} else {
		// Broadcast to all clients if no conversation ID specified
		h.broadcastToAllClients(message, "")
	}
}

func (h *Hub) broadcastToConversationParticipants(conversationID string, message *Message, excludeUserID string) {
	h.mu.RLock()

	// Collect clients to send to
	var targetClients []*Client
	for userID, client := range h.clients {
		if userID != excludeUserID && client.conversationID == conversationID {
			targetClients = append(targetClients, client)
		}
	}

	h.mu.RUnlock()

	// Send messages to collected clients
	var failedClients []string
	for _, client := range targetClients {
		select {
		case client.send <- message:
			h.logger.Debug("Message sent to client", "userID", client.userID, "type", message.Type)
		default:
			// Client's send channel is full, mark for removal
			failedClients = append(failedClients, client.userID)
			h.logger.Warn("Client send channel full, marking for removal", "userID", client.userID)
		}
	}

	// Remove failed clients in a separate operation
	if len(failedClients) > 0 {
		h.removeFailedClients(failedClients)
	}
}

func (h *Hub) broadcastToAllClients(message *Message, excludeUserID string) {
	h.mu.RLock()

	var targetClients []*Client
	for userID, client := range h.clients {
		if userID != excludeUserID {
			targetClients = append(targetClients, client)
		}
	}

	h.mu.RUnlock()

	var failedClients []string
	for _, client := range targetClients {
		select {
		case client.send <- message:
		default:
			failedClients = append(failedClients, client.userID)
		}
	}

	if len(failedClients) > 0 {
		h.removeFailedClients(failedClients)
	}
}

func (h *Hub) removeFailedClients(userIDs []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, userID := range userIDs {
		if client, exists := h.clients[userID]; exists {
			close(client.send)
			client.conn.Close()
			delete(h.clients, userID)
			h.logger.Info("Removed failed client", "userID", userID)
		}
	}
}

func (h *Hub) cleanupInactiveClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	var inactiveClients []string

	for userID, client := range h.clients {
		if now.Sub(client.lastActivity) > 5*time.Minute {
			inactiveClients = append(inactiveClients, userID)
		}
	}

	for _, userID := range inactiveClients {
		if client, exists := h.clients[userID]; exists {
			close(client.send)
			client.conn.Close()
			delete(h.clients, userID)
			h.logger.Info("Cleaned up inactive client", "userID", userID)
		}
	}

	if len(inactiveClients) > 0 {
		h.logger.Info("Cleaned up inactive clients", "count", len(inactiveClients))
	}
}

func (h *Hub) cleanupAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for userID, client := range h.clients {
		close(client.send)
		client.conn.Close()
		h.logger.Info("Cleaning up client on shutdown", "userID", userID)
	}

	// Clear the map
	h.clients = make(map[string]*Client)
}

// GetOnlineUsers returns list of online users in a conversation
func (h *Hub) GetOnlineUsers(conversationID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var onlineUsers []string
	for userID, client := range h.clients {
		if client.conversationID == conversationID {
			onlineUsers = append(onlineUsers, userID)
		}
	}
	return onlineUsers
}

// IsUserOnline checks if a user is currently online
func (h *Hub) IsUserOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	_, exists := h.clients[userID]
	return exists
}

// SendMessageToUser sends a message to a specific user
func (h *Hub) SendMessageToUser(userID string, message *Message) bool {
	h.mu.RLock()
	client, exists := h.clients[userID]
	h.mu.RUnlock()

	if !exists {
		return false
	}

	select {
	case client.send <- message:
		return true
	default:
		// Channel is full, try to remove the client
		go func() {
			h.removeFailedClients([]string{userID})
		}()
		return false
	}
}

// BroadcastToConversation broadcasts a message to all participants in a conversation
func (h *Hub) BroadcastToConversation(conversationID string, message *Message) {
	select {
	case h.broadcast <- message:
		h.logger.Debug("Message queued for broadcast", "conversationID", conversationID, "type", message.Type)
	default:
		h.logger.Error("Broadcast channel full, dropping message", "conversationID", conversationID)
	}
}

// UpdateClientConversation updates the current conversation for a client
func (h *Hub) UpdateClientConversation(userID, conversationID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client, exists := h.clients[userID]; exists {
		oldConversationID := client.conversationID
		client.conversationID = conversationID
		h.logger.Info("Updated client conversation", "userID", userID, "oldConversationID", oldConversationID, "newConversationID", conversationID)
	}
}

// GetClientCount returns the current number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
