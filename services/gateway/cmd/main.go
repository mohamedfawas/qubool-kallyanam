package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/server"
)

func main() {
	// Initialize logger
	logger := logging.Default()

	// Load configuration
	configPath := "./configs/config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	logger.Info("Loading configuration", "path", configPath)
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Fatal("Failed to load config", "error", err)
	}

	// Create server
	logger.Info("Initializing server")
	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", "error", err)
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting gateway service", "port", cfg.HTTP.Port)
		if err := srv.Start(); err != nil {
			if err != http.ErrServerClosed {
				logger.Fatal("Failed to start server", "error", err)
			}
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a context with a timeout of 10 seconds to wait for the server to shut down gracefully.
	// This prevents the shutdown from hanging indefinitely.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel() // important for cleaning up resources and stopping any background processes related to that context.
	// If the operation finishes early or you hit a timeout, cancel() ensures that the context is cleaned up.

	// Stop the server gracefully
	if err := srv.Stop(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("Server stopped")
}
