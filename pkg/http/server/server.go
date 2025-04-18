package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents an HTTP server
type Server struct {
	router     *gin.Engine
	httpServer *http.Server
	logger     *zap.Logger
	port       int
}

// NewServer creates a new HTTP server
func NewServer(logger *zap.Logger, port int) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	// Add recovery middleware
	router.Use(gin.Recovery())

	return &Server{
		router: router,
		logger: logger,
		port:   port,
	}
}

// Router returns the Gin router
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	s.logger.Info("Starting HTTP server", zap.Int("port", s.port))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	return s.httpServer.Shutdown(ctx)
}
