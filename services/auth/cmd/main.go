// services/auth/cmd/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/server"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logging.Initialize(cfg.Logging); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	logger := logging.Get()

	// Create and initialize server
	srv := server.New(cfg)
	if err := srv.Initialize(context.Background()); err != nil {
		logger.Fatal("Failed to initialize server", logging.Error(err))
	}

	// Start server
	if err := srv.Start(); err != nil {
		logger.Fatal("Server error", logging.Error(err))
	}
}
