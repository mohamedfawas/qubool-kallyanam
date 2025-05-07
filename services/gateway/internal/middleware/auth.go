package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
)

// AuthRequired is middleware that validates JWT tokens
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			pkghttp.Error(c, pkghttp.NewBadRequest("Missing authorization header", nil))
			c.Abort()
			return
		}

		// Extract the token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			pkghttp.Error(c, pkghttp.NewBadRequest("Invalid authorization format", nil))
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// In a real implementation, you would validate the token here
		// For MVP, we'll just forward the token to the microservices
		// They will do their own validation

		// Set userID and token in context for handlers to use
		// This allows forwarding the token to microservices
		c.Set("token", tokenString)

		// We'll also set a service context for the middleware
		c.Set("service", "user")

		c.Next()
	}
}

// ForwardAuthToken ensures the token is included in gRPC calls
func ForwardAuthToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// This middleware would be used before making gRPC calls
		// to ensure the auth token is forwarded to the microservices

		// Logic for actual token validation will be in the microservices
		c.Next()
	}
}

// AdminOnly ensures only admin users can access a route
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For MVP, we'll just check if the token contains admin role
		// In a real implementation, you would properly validate claims

		tokenString, exists := c.Get("token")
		if !exists {
			pkghttp.Error(c, pkghttp.NewBadRequest("No token found in context", nil))
			c.Abort()
			return
		}

		// Simple check - in real app, you'd validate properly
		// This is just a placeholder for MVP
		if !strings.Contains(tokenString.(string), "role\":\"ADMIN") {
			pkghttp.Error(c, pkghttp.NewBadRequest("Admin access required", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}
