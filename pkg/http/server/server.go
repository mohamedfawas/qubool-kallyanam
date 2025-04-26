package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Server wraps an HTTP server with additional functionality
type Server struct {
	config Config
	server *http.Server
	logger logging.Logger
	url    string
}

// New creates a new HTTP server
func New(handler http.Handler, config Config, logger logging.Logger) *Server {
	// Use default config if needed
	if config.Port == 0 {
		config = DefaultConfig()
	}

	// Create server address
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  config.Timeout.Read,
		WriteTimeout: config.Timeout.Write,
		IdleTimeout:  config.Timeout.Idle,
	}

	// Determine server URL
	scheme := "http"
	if config.TLS.Enabled {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s", scheme, addr)

	return &Server{
		config: config,
		server: httpServer,
		logger: logger.Named("http"),
		url:    url,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server",
		logging.String("url", s.url),
		logging.Int("port", s.config.Port),
	)

	var err error
	if s.config.TLS.Enabled {
		err = s.server.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
	} else {
		err = s.server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	s.logger.Info("Stopping HTTP server")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout.Shutdown)
	defer cancel()

	// Shutdown the server
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.logger.Info("HTTP server stopped")
	return nil
}

// URL returns the server's URL
func (s *Server) URL() string {
	return s.url
}
