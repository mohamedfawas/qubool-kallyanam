package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
)

// Constants for context keys
const (
	UserIDKey = "userID"
	RoleKey   = "userRole"
)

// Auth provides JWT authentication middleware for the gateway
type Auth struct {
	jwtManager *jwt.Manager
}

// NewAuth creates a new auth middleware
func NewAuth(jwtManager *jwt.Manager) *Auth {
	return &Auth{
		jwtManager: jwtManager,
	}
}

// Authenticate is a middleware that checks the Authorization header for a valid JWT.
// If valid, it extracts user information and stores it in the Gin context.
// If invalid or missing, it aborts the request with an HTTP 401 Unauthorized error.
func (a *Auth) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read the raw "Authorization" header from the incoming HTTP request.
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			pkghttp.Error(c, pkghttp.NewUnauthorized("Missing authorization header", nil))
			c.Abort()
			return
		}

		// Extract the token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			pkghttp.Error(c, pkghttp.NewUnauthorized("Invalid authorization format", nil))
			c.Abort()
			return
		}

		tokenString := tokenParts[1]

		// Validate the token
		claims, err := a.jwtManager.ValidateToken(tokenString)
		if err != nil {
			// If the error message contains "expired", the token timed out.
			if strings.Contains(err.Error(), "expired") {
				pkghttp.Error(c, pkghttp.NewUnauthorized("Token has expired", nil))
			} else {
				pkghttp.Error(c, pkghttp.NewUnauthorized("Invalid token", nil))
			}
			c.Abort()
			return
		}

		// Store essential claims in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(RoleKey, claims.Role)

		// Store token for forwarding to microservices
		c.Set("token", tokenString)

		c.Next()
	}
}

// RequireRole returns a middleware that checks if the authenticated user has
// one of the specified roles. For example, RequireRole(jwt.RoleAdmin) will
// only allow users with role "admin" to proceed.
func (a *Auth) RequireRole(roles ...jwt.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve the stored role from context
		role, exists := c.Get(RoleKey)
		if !exists {
			// No role means user is not authenticated
			pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
			c.Abort()
			return
		}

		// Convert to jwt.Role type
		//
		// role.(jwt.Role) converts the generic interface{} value you pulled from the Gin context
		// back into the concrete jwt.Role type
		// so you can work with it in your permission-checking logic
		userRole := role.(jwt.Role)

		// Check if user's role matches any allowed role
		for _, r := range roles {
			if userRole == r {
				c.Next()
				return
			}
		}

		// If no match, respond with 403 Forbidden
		pkghttp.Error(c, pkghttp.NewForbidden("Insufficient permissions", nil))
		c.Abort()
	}
}

// WithUserID extracts the userID from Gin context and injects it into Go's request context
// so that any downstream HTTP client or service call can access userID via c.Request.Context()
func (a *Auth) WithUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(UserIDKey)
		if exists {
			// Add userID to the request context
			ctx := context.WithValue(c.Request.Context(), UserIDKey, userID.(string))
			c.Request = c.Request.WithContext(ctx)
		}
		c.Next()
	}
}

// ForwardToken injects the Authorization header value into Go's request context
// so that downstream HTTP clients (e.g., microservice calls) can automatically
// forward the same token in their outbound requests
func (a *Auth) ForwardToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add authorization token to the context
		token, exists := c.Get("token")
		if exists {
			ctx := context.WithValue(
				c.Request.Context(),
				"authorization",
				"Bearer "+token.(string),
			)
			c.Request = c.Request.WithContext(ctx)
		}

		// Add user ID to the context
		userID, exists := c.Get(UserIDKey)
		if exists {
			ctx := context.WithValue(
				c.Request.Context(),
				"user-id",
				userID.(string),
			)
			c.Request = c.Request.WithContext(ctx)
		}

		c.Next()
	}
}

// GetUserID reads the userID from Gin context and returns it.
// Returns ("", false) if not found.
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// // GetUserRole reads the userRole from Gin context and returns it.
// // Returns ("", false) if not found
// func GetUserRole(c *gin.Context) (jwt.Role, bool) {
// 	role, exists := c.Get(RoleKey)
// 	if !exists {
// 		return "", false
// 	}
// 	return role.(jwt.Role), true
// }
