package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create server
	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting admin service on port %d", cfg.GRPC.Port)
		if err := srv.Start(); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	srv.Stop()
	log.Println("Server stopped")
}
