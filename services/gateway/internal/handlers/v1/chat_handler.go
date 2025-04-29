package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ChatHandler handles chat-related requests
type ChatHandler struct {
	// TODO: Add chat service client
}

// NewChatHandler creates a new ChatHandler
func NewChatHandler() *ChatHandler {
	return &ChatHandler{}
}

// GetChats returns all chats for a user
func (h *ChatHandler) GetChats(c *gin.Context) {
	userId, _ := c.Get("userId")

	// TODO: Call chat service

	c.JSON(http.StatusOK, gin.H{
		"chats": []gin.H{
			{
				"id":          "chat-1",
				"withUser":    "user-1",
				"userName":    "Jane Smith",
				"lastMessage": "Hi there!",
				"timestamp":   "2023-09-15T14:30:00Z",
				"unread":      2,
			},
			{
				"id":          "chat-2",
				"withUser":    "user-2",
				"userName":    "Alice Johnson",
				"lastMessage": "Nice to meet you",
				"timestamp":   "2023-09-14T18:45:00Z",
				"unread":      0,
			},
		},
	})
}

// GetChat returns a specific chat
func (h *ChatHandler) GetChat(c *gin.Context) {
	chatId := c.Param("id")
	userId, _ := c.Get("userId")

	// TODO: Call chat service

	c.JSON(http.StatusOK, gin.H{
		"id":       chatId,
		"withUser": "user-1",
		"userName": "Jane Smith",
		"messages": []gin.H{
			{
				"id":        "msg-1",
				"sender":    "user-1",
				"content":   "Hi there!",
				"timestamp": "2023-09-15T14:30:00Z",
			},
			{
				"id":        "msg-2",
				"sender":    userId,
				"content":   "Hello! Nice to meet you.",
				"timestamp": "2023-09-15T14:31:00Z",
			},
		},
	})
}

// SendMessage sends a message in a chat
func (h *ChatHandler) SendMessage(c *gin.Context) {
	chatId := c.Param("id")
	userId, _ := c.Get("userId")

	var req struct {
		Content string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Call chat service

	c.JSON(http.StatusCreated, gin.H{
		"id":        "new-msg-id",
		"chatId":    chatId,
		"sender":    userId,
		"content":   req.Content,
		"timestamp": "2023-09-15T15:00:00Z",
	})
}
