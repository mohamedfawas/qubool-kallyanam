package chat

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump(ctx context.Context, chatHandler *Handler) {
	defer func() {
		c.hub.logger.Info("Closing readPump for user", "userID", c.userID)
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.lastActivity = time.Now()
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			chatHandler.logger.Info("Context cancelled for readPump", "userID", c.userID)
			return
		default:
			var message Message
			err := c.conn.ReadJSON(&message)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					chatHandler.logger.Error("WebSocket error", "error", err, "userID", c.userID)
				} else {
					chatHandler.logger.Info("WebSocket connection closed", "userID", c.userID, "error", err)
				}
				break
			}

			c.lastActivity = time.Now()
			chatHandler.logger.Debug("Received WebSocket message", "userID", c.userID, "type", message.Type, "conversationID", message.ConversationID)
			c.handleMessage(ctx, &message, chatHandler)
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		c.hub.logger.Info("Closing writePump for user", "userID", c.userID)
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			c.hub.logger.Debug("Sending WebSocket message", "userID", c.userID, "type", message.Type)
			if err := c.conn.WriteJSON(message); err != nil {
				c.hub.logger.Error("Error writing WebSocket message", "error", err, "userID", c.userID)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.hub.logger.Error("Error sending ping", "error", err, "userID", c.userID)
				return
			}

		case <-ctx.Done():
			c.hub.logger.Info("Context cancelled for writePump", "userID", c.userID)
			return
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *Client) handleMessage(ctx context.Context, message *Message, chatHandler *Handler) {
	c.hub.logger.Debug("Handling message", "userID", c.userID, "type", message.Type, "conversationID", message.ConversationID)

	switch message.Type {
	case MessageTypeJoin:
		c.handleJoinConversation(message)

	case MessageTypeLeave:
		c.handleLeaveConversation()

	case MessageTypeMessage:
		c.handleChatMessage(ctx, message, chatHandler)

	case MessageTypeTyping:
		c.handleTyping(message)

	case MessageTypeStopTyping:
		c.handleStopTyping(message)

	default:
		c.hub.logger.Warn("Unknown message type", "type", message.Type, "userID", c.userID)
		c.sendError("Unknown message type")
	}
}

func (c *Client) handleJoinConversation(message *Message) {
	if message.ConversationID == "" {
		c.sendError("Conversation ID required")
		return
	}

	oldConversationID := c.conversationID
	c.conversationID = message.ConversationID
	c.hub.UpdateClientConversation(c.userID, message.ConversationID)

	// Send acknowledgment
	ackMessage := &Message{
		Type:           MessageTypeAck,
		ConversationID: message.ConversationID,
		SentAt:         time.Now(),
		Data: map[string]interface{}{
			"action":              "joined_conversation",
			"conversation_id":     message.ConversationID,
			"old_conversation_id": oldConversationID,
		},
	}
	c.send <- ackMessage

	c.hub.logger.Info("User joined conversation", "userID", c.userID, "conversationID", message.ConversationID, "oldConversationID", oldConversationID)

	// Notify other users in the conversation that this user is now online
	onlineMessage := &Message{
		Type:           MessageTypeUserOnline,
		ConversationID: message.ConversationID,
		SenderID:       c.userID,
		SentAt:         time.Now(),
	}
	c.hub.broadcastToConversationParticipants(message.ConversationID, onlineMessage, c.userID)
}

func (c *Client) handleLeaveConversation() {
	if c.conversationID != "" {
		oldConversationID := c.conversationID

		// Notify others in the conversation that user is leaving
		offlineMessage := &Message{
			Type:           MessageTypeUserOffline,
			ConversationID: c.conversationID,
			SenderID:       c.userID,
			SentAt:         time.Now(),
		}
		c.hub.broadcastToConversationParticipants(c.conversationID, offlineMessage, c.userID)

		c.hub.logger.Info("User left conversation", "userID", c.userID, "conversationID", oldConversationID)
		c.conversationID = ""
		c.hub.UpdateClientConversation(c.userID, "")

		// Send acknowledgment
		ackMessage := &Message{
			Type:   MessageTypeAck,
			SentAt: time.Now(),
			Data: map[string]interface{}{
				"action":          "left_conversation",
				"conversation_id": oldConversationID,
			},
		}
		c.send <- ackMessage
	}
}

func (c *Client) handleChatMessage(ctx context.Context, message *Message, chatHandler *Handler) {
	if message.ConversationID == "" || message.Text == "" {
		c.sendError("Conversation ID and text are required")
		return
	}

	c.hub.logger.Info("Processing chat message", "userID", c.userID, "conversationID", message.ConversationID, "textLength", len(message.Text))

	// Send message through the existing chat service
	savedMessage, err := chatHandler.chatClient.SendMessage(ctx, c.userID, message.ConversationID, message.Text)
	if err != nil {
		c.hub.logger.Error("Failed to save message", "error", err, "userID", c.userID)
		c.sendError("Failed to send message: " + err.Error())
		return
	}

	c.hub.logger.Info("Message saved successfully", "messageID", savedMessage.Id, "userID", c.userID)

	// Create broadcast message
	broadcastMessage := &Message{
		Type:           MessageTypeMessage,
		ConversationID: message.ConversationID,
		MessageID:      savedMessage.Id,
		SenderID:       savedMessage.SenderId,
		Text:           savedMessage.Text,
		SentAt:         savedMessage.SentAt.AsTime(),
	}

	// Broadcast to all participants in the conversation (including sender for confirmation)
	c.hub.BroadcastToConversation(message.ConversationID, broadcastMessage)
}

func (c *Client) handleTyping(message *Message) {
	if message.ConversationID == "" {
		return
	}

	typingMessage := &Message{
		Type:           MessageTypeTyping,
		ConversationID: message.ConversationID,
		SenderID:       c.userID,
		SentAt:         time.Now(),
	}

	c.hub.logger.Debug("User typing", "userID", c.userID, "conversationID", message.ConversationID)
	c.hub.broadcastToConversationParticipants(message.ConversationID, typingMessage, c.userID)
}

func (c *Client) handleStopTyping(message *Message) {
	if message.ConversationID == "" {
		return
	}

	stopTypingMessage := &Message{
		Type:           MessageTypeStopTyping,
		ConversationID: message.ConversationID,
		SenderID:       c.userID,
		SentAt:         time.Now(),
	}

	c.hub.logger.Debug("User stopped typing", "userID", c.userID, "conversationID", message.ConversationID)
	c.hub.broadcastToConversationParticipants(message.ConversationID, stopTypingMessage, c.userID)
}

func (c *Client) sendError(errorMsg string) {
	errorMessage := &Message{
		Type:   MessageTypeError,
		Error:  errorMsg,
		SentAt: time.Now(),
	}

	select {
	case c.send <- errorMessage:
		c.hub.logger.Debug("Error message sent", "userID", c.userID, "error", errorMsg)
	default:
		c.hub.logger.Error("Failed to send error message", "userID", c.userID, "error", errorMsg)
	}
}
