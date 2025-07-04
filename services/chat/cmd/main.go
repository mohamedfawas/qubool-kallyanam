package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/server"
)

func main() {
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

	logger.Info("Initializing server")
	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", "error", err)
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting chat service", "port", cfg.GRPC.Port)
		if err := srv.Start(); err != nil {
			logger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	srv.Stop()
	logger.Info("Server stopped")
}
