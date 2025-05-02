package server

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	authclient "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/auth"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/config"
	authhandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/auth"
)

// Server represents the HTTP server for the gateway service
type Server struct {
	cfg        *config.Config
	router     *gin.Engine
	httpServer *http.Server
	authClient *authclient.Client
}

// NewServer creates a new HTTP server instance
func NewServer(cfg *config.Config, authClient *authclient.Client) *Server {
	// Set Gin mode
	gin.SetMode(cfg.Server.Mode)

	// Initialize Gin router
	router := gin.Default()

	// Create the server
	return &Server{
		cfg:        cfg,
		router:     router,
		authClient: authClient,
	}
}

// SetupRoutes configures all the routes for the server
func (s *Server) SetupRoutes() {

	// Add CORS middleware first
	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	// Simple health check endpoints
	s.router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "UP",
			"service": "gateway",
		})
	})

	s.router.GET("/health/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "READY",
			"service": "gateway",
		})
	})

	// // Simple test endpoint
	// s.router.GET("/ping", func(c *gin.Context) {
	// 	c.JSON(http.StatusOK, gin.H{
	// 		"message": "pong",
	// 	})
	// })

	// API endpoints
	apiGroup := s.router.Group("/api/v1")

	// Register auth handler
	authHandler := authhandler.NewHandler(s.authClient)
	authHandler.RegisterRoutes(apiGroup)

	// Log all registered routes for debugging
	for _, route := range s.router.Routes() {
		log.Printf("Registered route: %s %s", route.Method, route.Path)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Setup HTTP server
	s.httpServer = &http.Server{
		Addr:    s.cfg.Server.Address,
		Handler: s.router,
	}

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Gateway server starting on %s", s.cfg.Server.Address)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start gateway server: %v", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// WaitForShutdown waits for an interrupt signal to gracefully shut down the server
func (s *Server) WaitForShutdown() {
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gateway server...")

	if err := s.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}

	log.Println("Gateway server stopped")
}
