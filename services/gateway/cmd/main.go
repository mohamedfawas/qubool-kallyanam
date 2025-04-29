package main

import (
	"fmt"
	"log"

	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize and start server
	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start the server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting gateway server on %s", addr)
	if err := srv.Run(addr); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
