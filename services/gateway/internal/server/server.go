package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/auth"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/config"
	authHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/auth"
)

// Server represents the HTTP server
type Server struct {
	config     *config.Config
	logger     logging.Logger
	httpServer *http.Server
	router     *gin.Engine
	authClient *auth.Client
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, logger logging.Logger) (*Server, error) {
	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()

	// Create auth client
	authClient, err := auth.NewClient(cfg.Services.Auth.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth client: %w", err)
	}

	// // Create user client
	// userClient, err := user.NewClient(cfg.Services.User.Address)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create user client: %w", err)
	// }

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  time.Second * time.Duration(cfg.HTTP.ReadTimeoutSecs),
		WriteTimeout: time.Second * time.Duration(cfg.HTTP.WriteTimeoutSecs),
		IdleTimeout:  time.Second * time.Duration(cfg.HTTP.IdleTimeoutSecs),
	}

	server := &Server{
		config:     cfg,
		logger:     logger,
		httpServer: httpServer,
		router:     router,
		authClient: authClient,
		// userClient: userClient,
	}

	// Initialize routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	// Create handlers
	authHandler := authHandler.NewHandler(s.authClient, s.logger)
	// userHandler := userHandler.NewHandler(s.userClient, s.logger)

	// Setup router
	SetupRouter(s.router,
		authHandler,
		// userHandler,
		s.logger)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", "port", s.config.HTTP.Port)
	return s.httpServer.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")

	// Close clients
	if s.authClient != nil {
		s.authClient.Close()
	}

	// if s.userClient != nil {
	// 	s.userClient.Close()
	// }

	// Shutdown HTTP server
	return s.httpServer.Shutdown(ctx)
}
