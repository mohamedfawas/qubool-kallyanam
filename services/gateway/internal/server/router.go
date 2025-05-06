package server

import (
	"github.com/gin-gonic/gin"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	authHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/auth"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// SetupRouter configures all routes
func SetupRouter(
	r *gin.Engine,
	authHandler *authHandler.Handler,
	logger logging.Logger,
) {
	// Add middleware
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.ErrorHandler())

	// API v1 routes
	v1 := r.Group("/api/v1")

	// Auth routes
	auth := v1.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/verify", authHandler.Verify)
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout)
		auth.POST("/refresh", authHandler.RefreshToken)
		auth.POST("/admin/login", authHandler.AdminLogin)
		auth.POST("/admin/logout", authHandler.AdminLogout)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})
}
