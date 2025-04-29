package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AdminHandler handles admin-related requests
type AdminHandler struct {
	// TODO: Add admin service client
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

// ListUsers returns a list of users
func (h *AdminHandler) ListUsers(c *gin.Context) {
	// Get pagination parameters
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

	// TODO: Call admin service

	c.JSON(http.StatusOK, gin.H{
		"users": []gin.H{
			{
				"id":        "user-1",
				"firstName": "Jane",
				"lastName":  "Smith",
				"email":     "jane.smith@example.com",
				"status":    "active",
			},
			{
				"id":        "user-2",
				"firstName": "Alice",
				"lastName":  "Johnson",
				"email":     "alice.johnson@example.com",
				"status":    "active",
			},
		},
		"total": 2,
		"page":  page,
		"limit": limit,
	})
}

// UpdateUser updates a user's details
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userId := c.Param("id")

	var req struct {
		Status    string `json:"status"`
		IsBlocked bool   `json:"isBlocked"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Call admin service

	c.JSON(http.StatusOK, gin.H{
		"id":      userId,
		"message": "User updated successfully",
	})
}
