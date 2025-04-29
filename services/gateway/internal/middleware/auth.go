package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthRequired is a middleware that checks if the user is authenticated
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// Check if it's a Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}

		token := parts[1]

		// TODO: Call auth service to validate token
		// For now, just validate that token exists
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Set user ID in context (mock value for now)
		c.Set("userId", "mock-user-id")

		c.Next()
	}
}

// AdminRequired is a middleware that checks if the user is an admin
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// First check if user is authenticated
		AuthRequired()(c)

		if c.IsAborted() {
			return
		}

		// TODO: Check if user is admin via auth service
		// For now, just a placeholder
		isAdmin := false // Placeholder for admin check

		if !isAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		c.Next()
	}
}
