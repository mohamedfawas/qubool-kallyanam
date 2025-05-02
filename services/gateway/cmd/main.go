package main

import (
	"log"

	authclient "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/auth"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/server"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize service clients
	authClient, err := authclient.NewClient(cfg.Services.Auth.Address)
	if err != nil {
		log.Fatalf("Failed to create auth client: %v", err)
	}
	defer authClient.Close()

	// Create and setup server
	srv := server.NewServer(cfg, authClient)
	srv.SetupRoutes()

	// Start the server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for shutdown signal
	srv.WaitForShutdown()
}
