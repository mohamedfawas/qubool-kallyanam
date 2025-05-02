package http

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/errors"
)

const (
	UserIDKey       = "userID"
	RoleKey         = "role"
	VerifiedKey     = "verified"
	PremiumUntilKey = "premiumUntil"
	ServicesKey     = "services"
)

// AuthMiddleware authenticates requests using JWT and checks service access
func AuthMiddleware(jwtManager *jwt.Manager, service string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			Error(c, errors.NewUnauthorized("Missing authorization header", nil))
			c.Abort()
			return
		}

		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			Error(c, errors.NewUnauthorized("Invalid authorization format", nil))
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(tokenParts[1])
		if err != nil {
			Error(c, errors.NewUnauthorized("Invalid or expired token", err))
			c.Abort()
			return
		}

		// Check if user has access to the requested service
		if !jwtManager.CheckServiceAccess(claims, service) {
			Error(c, errors.NewForbidden("Access to this service is not allowed", nil))
			c.Abort()
			return
		}

		// Add user info to context
		c.Set(UserIDKey, claims.UserID)
		c.Set(RoleKey, claims.Role)
		c.Set(VerifiedKey, claims.Verified)
		c.Set(PremiumUntilKey, claims.PremiumUntil)
		c.Set(ServicesKey, claims.Services)

		c.Next()
	}
}

// AdminOnly middleware ensures only admins can access a route
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get(RoleKey)
		if !exists {
			Error(c, errors.NewUnauthorized("User not authenticated", nil))
			c.Abort()
			return
		}

		role, ok := roleValue.(jwt.Role)
		if !ok {
			Error(c, errors.NewInternalServerError("Invalid role type in context", nil))
			c.Abort()
			return
		}

		if role != jwt.RoleAdmin {
			Error(c, errors.NewForbidden("Admin access required", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}

// PremiumOnly middleware ensures only premium users or admins can access a route
func PremiumOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get(RoleKey)
		if !exists {
			Error(c, errors.NewUnauthorized("User not authenticated", nil))
			c.Abort()
			return
		}

		role, ok := roleValue.(jwt.Role)
		if !ok {
			Error(c, errors.NewInternalServerError("Invalid role type in context", nil))
			c.Abort()
			return
		}

		if role != jwt.RolePremiumUser && role != jwt.RoleAdmin {
			Error(c, errors.NewForbidden("Premium access required", nil))
			c.Abort()
			return
		}

		// For premium users, check if premium is still valid
		if role == jwt.RolePremiumUser {
			premiumUntilValue, exists := c.Get(PremiumUntilKey)
			if !exists || premiumUntilValue == nil {
				Error(c, errors.NewForbidden("Premium subscription expired", nil))
				c.Abort()
				return
			}

			premiumUntil, ok := premiumUntilValue.(*int64)
			if !ok || premiumUntil == nil {
				Error(c, errors.NewInternalServerError("Invalid premium expiry in context", nil))
				c.Abort()
				return
			}

			currentTime := time.Now().Unix()
			if *premiumUntil < currentTime {
				Error(c, errors.NewForbidden("Premium subscription expired", nil))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// VerifiedOnly middleware ensures only verified users can access a route
func VerifiedOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		verifiedValue, exists := c.Get(VerifiedKey)
		if !exists {
			Error(c, errors.NewUnauthorized("User not authenticated", nil))
			c.Abort()
			return
		}

		verified, ok := verifiedValue.(bool)
		if !ok {
			Error(c, errors.NewInternalServerError("Invalid verification status in context", nil))
			c.Abort()
			return
		}

		if !verified {
			Error(c, errors.NewForbidden("Account verification required", nil))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uint, error) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return 0, errors.NewUnauthorized("User not authenticated", nil)
	}

	id, ok := userID.(uint)
	if !ok {
		return 0, errors.NewUnauthorized("Invalid user ID in context", nil)
	}

	return id, nil
}

// GetUserRole extracts user role from context
func GetUserRole(c *gin.Context) (jwt.Role, error) {
	roleValue, exists := c.Get(RoleKey)
	if !exists {
		return "", errors.NewUnauthorized("User not authenticated", nil)
	}

	role, ok := roleValue.(jwt.Role)
	if !ok {
		return "", errors.NewUnauthorized("Invalid role in context", nil)
	}

	return role, nil
}

// IsUserVerified checks if the user is verified
func IsUserVerified(c *gin.Context) (bool, error) {
	verifiedValue, exists := c.Get(VerifiedKey)
	if !exists {
		return false, errors.NewUnauthorized("User not authenticated", nil)
	}

	verified, ok := verifiedValue.(bool)
	if !ok {
		return false, errors.NewInternalServerError("Invalid verification status in context", nil)
	}

	return verified, nil
}

// IsUserPremium checks if the user has a valid premium subscription
func IsUserPremium(c *gin.Context) (bool, error) {
	roleValue, exists := c.Get(RoleKey)
	if !exists {
		return false, errors.NewUnauthorized("User not authenticated", nil)
	}

	role, ok := roleValue.(jwt.Role)
	if !ok {
		return false, errors.NewInternalServerError("Invalid role in context", nil)
	}

	// Admin is always considered premium
	if role == jwt.RoleAdmin {
		return true, nil
	}

	// Check if user is premium and subscription is still valid
	if role == jwt.RolePremiumUser {
		premiumUntilValue, exists := c.Get(PremiumUntilKey)
		if !exists || premiumUntilValue == nil {
			return false, nil
		}

		premiumUntil, ok := premiumUntilValue.(*int64)
		if !ok || premiumUntil == nil {
			return false, errors.NewInternalServerError("Invalid premium expiry in context", nil)
		}

		return time.Now().Unix() < *premiumUntil, nil
	}

	return false, nil
}
