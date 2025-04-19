// pkg/service/service.go
package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Resource represents a closable resource
type Resource interface {
	// Close closes the resource
	Close() error
}

// ContextResource represents a resource that requires a context to close
type ContextResource interface {
	// Close closes the resource with context
	Close(ctx context.Context) error
}

// Service encapsulates common functionality for all services
type Service struct {
	// Core attributes
	Name   string
	Router *gin.Engine
	Logger *zap.Logger
	Config interface{}
	Server *http.Server

	// Resource management
	resources []Resource

	// Context for the service lifecycle
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new service instance
func New(name string, configLoader func() (interface{}, error)) (*Service, error) {
	// Load configuration
	cfg, err := configLoader()
	if err != nil {
		fmt.Printf("Failed to load config for %s: %v\n", name, err)
		return nil, err
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create service instance
	svc := &Service{
		Name:      name,
		Config:    cfg,
		resources: make([]Resource, 0),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Initialize logger
	if err := svc.initLogger(); err != nil {
		return nil, err
	}

	// Initialize router
	svc.initRouter()

	return svc, nil
}

// initLogger initializes the service logger
func (s *Service) initLogger() error {
	// Initialize logger based on common config pattern
	// This assumes your config has a common structure with Debug field
	debug := false

	// Type assertion to get debug flag
	// This is a simplistic approach - in real implementation you'd use reflection
	// or a more structured approach to access config fields
	if commonConfig, ok := s.Config.(interface{ GetDebug() bool }); ok {
		debug = commonConfig.GetDebug()
	}

	// Create logger
	logger, err := s.createLogger(debug)
	if err != nil {
		fmt.Printf("Failed to create logger for %s: %v\n", s.Name, err)
		return err
	}

	s.Logger = logger
	return nil
}

// createLogger creates a new logger instance
// This would call your existing logger implementation
func (s *Service) createLogger(debug bool) (*zap.Logger, error) {
	// Placeholder - replace with your actual logger implementation
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, err
	}
	return logger.Named(s.Name), nil
}

// initRouter initializes the Gin router
func (s *Service) initRouter() {
	router := gin.New()
	router.Use(gin.Recovery())
	s.Router = router
}

// AddResource adds a resource to be managed by the service
func (s *Service) AddResource(res interface{}) {
	if r, ok := res.(Resource); ok {
		s.resources = append(s.resources, r)
	} else if r, ok := res.(ContextResource); ok {
		// Create an adapter for context resources
		adapter := contextResourceAdapter{
			resource: r,
			ctx:      s.ctx,
		}
		s.resources = append(s.resources, adapter)
	}
}

// contextResourceAdapter adapts a ContextResource to Resource
type contextResourceAdapter struct {
	resource ContextResource
	ctx      context.Context
}

// Close implements the Resource interface
func (a contextResourceAdapter) Close() error {
	return a.resource.Close(a.ctx)
}

// SetupServer configures the HTTP server with the given host and port
func (s *Service) SetupServer(host string, port int) {
	s.Server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: s.Router,
	}
}

// Run starts the service and blocks until shutdown
func (s *Service) Run() error {
	// Start server in a goroutine
	go func() {
		s.Logger.Info("Starting service",
			zap.String("name", s.Name),
			zap.String("address", s.Server.Addr))

		if err := s.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.Logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.Logger.Info("Shutting down service...", zap.String("name", s.Name))

	// Call cancel to notify all goroutines to stop
	s.cancel()

	// Create a deadline for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown server
	if err := s.Server.Shutdown(shutdownCtx); err != nil {
		s.Logger.Error("Server forced to shutdown", zap.Error(err))
		return err
	}

	// Close all resources in reverse order
	for i := len(s.resources) - 1; i >= 0; i-- {
		res := s.resources[i]
		if err := res.Close(); err != nil {
			s.Logger.Error("Error closing resource", zap.Error(err))
		}
	}

	s.Logger.Info("Service exited", zap.String("name", s.Name))
	return nil
}

// Context returns the service context
func (s *Service) Context() context.Context {
	return s.ctx
}
