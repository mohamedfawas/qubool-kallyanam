package chat

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, you should validate the origin
		return true
	},
}

// HandleWebSocket upgrades HTTP connection to WebSocket for real-time chat
func (h *Handler) HandleWebSocket(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Error("Missing user ID in WebSocket request")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	userIDStr := userID.(string)
	h.logger.Info("WebSocket connection request", "userID", userIDStr)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade WebSocket connection", "error", err, "userID", userIDStr)
		return
	}

	h.logger.Info("WebSocket connection established", "userID", userIDStr)

	client := &Client{
		hub:          h.hub,
		conn:         conn,
		send:         make(chan *Message, 256),
		userID:       userIDStr,
		lastActivity: time.Now(),
	}

	// Register the client with the hub
	h.logger.Debug("Registering client with hub", "userID", userIDStr)
	client.hub.register <- client

	// Create a context for this connection
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer func() {
		h.logger.Info("WebSocket connection cleanup", "userID", userIDStr)
		cancel()
	}()

	// Start the client pumps in separate goroutines
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("WritePump panic recovered", "userID", userIDStr, "panic", r)
			}
		}()
		client.writePump(ctx)
	}()

	// ReadPump blocks until connection is closed
	func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("ReadPump panic recovered", "userID", userIDStr, "panic", r)
			}
		}()
		client.readPump(ctx, h)
	}()

	h.logger.Info("WebSocket connection closed", "userID", userIDStr)
}

// GetOnlineStatus returns the online status of users in a conversation
func (h *Handler) GetOnlineStatus(c *gin.Context) {
	conversationID := c.Param("id")
	if conversationID == "" {
		pkghttp.Error(c, pkghttp.NewBadRequest("Missing conversation ID", nil))
		return
	}

	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	h.logger.Info("Getting online status", "conversationID", conversationID, "userID", userID)

	// Check if user is participant in this conversation (you might want to add this validation)

	onlineUsers := h.hub.GetOnlineUsers(conversationID)
	totalClients := h.hub.GetClientCount()

	response := gin.H{
		"conversation_id": conversationID,
		"online_users":    onlineUsers,
		"total_online":    len(onlineUsers),
		"total_clients":   totalClients,
	}

	pkghttp.Success(c, http.StatusOK, "Online status retrieved successfully", response)
}
