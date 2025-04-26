// services/admin/internal/server/server.go
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
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/http/middleware"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/config"
	handlers "github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/handlers/health"
)

// Server represents the admin service server
type Server struct {
	cfg      *config.Config
	router   *gin.Engine
	logger   logging.Logger
	postgres *postgres.Client
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
	// Initialize database client
	s.postgres = postgres.NewClient(s.cfg.Postgres, s.logger)

	// Connect to database
	if err := s.postgres.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Register routes
	s.registerRoutes()

	return nil
}

// registerRoutes registers all API routes
func (s *Server) registerRoutes() {
	// Create handlers
	healthHandler := handlers.NewHealthHandler(s.postgres, s.logger)

	// Health check endpoints
	s.router.GET("/health", healthHandler.Check)
	s.router.GET("/health/live", healthHandler.LivenessCheck)
	s.router.GET("/health/ready", healthHandler.ReadinessCheck)

	// API v1 group
	v1 := s.router.Group("/api/v1")
	{
		// Admin endpoints will go here
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
		s.logger.Info("Starting admin service server",
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

	// Close database connection
	if err := s.postgres.Close(); err != nil {
		s.logger.Error("Failed to close PostgreSQL connection", logging.Error(err))
	}

	s.logger.Info("Server stopped gracefully")
	return nil
}
