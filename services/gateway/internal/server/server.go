// services/gateway/internal/server/server.go
package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/http/middleware"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/config"
	handlers "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/health"
)

// Server represents the API gateway server
type Server struct {
	cfg          *config.Config
	router       *gin.Engine
	logger       logging.Logger
	healthClient *clients.ServiceHealthClient
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	// Initialize logger
	logger := logging.Get()

	// Setup Gin router with middlewares
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS())

	return &Server{
		cfg:    cfg,
		router: router,
		logger: logger,
	}
}

// Initialize initializes the server and its dependencies
func (s *Server) Initialize(ctx context.Context) error {
	// Create service map for health client
	serviceMap := map[string]string{
		"auth":  s.cfg.Services.Auth,
		"user":  s.cfg.Services.User,
		"chat":  s.cfg.Services.Chat,
		"admin": s.cfg.Services.Admin,
	}

	// Initialize health client
	s.healthClient = clients.NewServiceHealthClient(serviceMap, s.logger)

	// Register routes
	s.registerRoutes()

	return nil
}

// registerRoutes registers all API routes
func (s *Server) registerRoutes() {
	// Create handlers
	healthHandler := handlers.NewHealthHandler(s.healthClient, s.logger)

	// Health check endpoints
	s.router.GET("/health", healthHandler.Check)
	s.router.GET("/health/live", healthHandler.LivenessCheck)
	s.router.GET("/health/ready", healthHandler.ReadinessCheck)
	s.router.GET("/health/detailed", healthHandler.DetailedCheck)

	// API v1 group
	v1 := s.router.Group("/api/v1")
	{
		// Gateway endpoints will go here
		v1.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "pong"})
		})
	}
}

// Start starts the server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Channel for server errors
	errCh := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		s.logger.Info("Starting API gateway server",
			logging.String("address", addr),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Channel for shutdown signals
	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case err := <-errCh:
		return err
	case <-shutdownCh:
		s.logger.Info("Received shutdown signal")
	}

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("Server stopped gracefully")
	return nil
}
