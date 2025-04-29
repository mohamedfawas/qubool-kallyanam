package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related requests
type UserHandler struct {
	// TODO: Add user service client
}

// NewUserHandler creates a new UserHandler
func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

// GetProfile handles user profile retrieval
func (h *UserHandler) GetProfile(c *gin.Context) {
	userId, _ := c.Get("userId")

	// TODO: Call user service

	c.JSON(http.StatusOK, gin.H{
		"id":        userId,
		"firstName": "John",
		"lastName":  "Doe",
		"email":     "john.doe@example.com",
		"profile": gin.H{
			"age":      28,
			"gender":   "male",
			"location": "Bangalore",
			// Other profile fields
		},
	})
}

// UpdateProfile handles user profile updates
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userId, _ := c.Get("userId")

	var req struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Profile   struct {
			Age      int    `json:"age"`
			Gender   string `json:"gender"`
			Location string `json:"location"`
			// Other profile fields
		} `json:"profile"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Call user service

	c.JSON(http.StatusOK, gin.H{
		"id":      userId,
		"message": "Profile updated successfully",
	})
}

// Search handles matrimonial profile search
func (h *UserHandler) Search(c *gin.Context) {
	// Get search parameters from query
	ageMin := c.DefaultQuery("ageMin", "18")
	ageMax := c.DefaultQuery("ageMax", "50")
	gender := c.DefaultQuery("gender", "")
	location := c.DefaultQuery("location", "")

	// TODO: Call user service

	// Mock response
	c.JSON(http.StatusOK, gin.H{
		"results": []gin.H{
			{
				"id":        "user-1",
				"firstName": "Jane",
				"lastName":  "Smith",
				"profile": gin.H{
					"age":      26,
					"gender":   "female",
					"location": "Mumbai",
				},
			},
			{
				"id":        "user-2",
				"firstName": "Alice",
				"lastName":  "Johnson",
				"profile": gin.H{
					"age":      29,
					"gender":   "female",
					"location": "Delhi",
				},
			},
		},
		"total": 2,
	})
}
