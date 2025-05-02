package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	authv1 "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/server"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database connections
	postgresClient, err := postgres.NewClient(&postgres.Config{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		DBName:   cfg.Postgres.DBName,
		SSLMode:  cfg.Postgres.SSLMode,
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	redisClient, err := redis.NewClient(&redis.Config{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Set up listener
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", addr, err)
	}

	// Initialize gRPC server
	grpcServer := grpc.NewServer()

	// Register auth service
	authServer := server.NewAuthServer(postgresClient.DB, redisClient)
	authv1.RegisterAuthServiceServer(grpcServer, authServer)

	// Register reflection service for debugging
	reflection.Register(grpcServer)

	// Start server in a goroutine
	go func() {
		log.Printf("Auth service starting on %s", addr)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	// Gracefully stop the server
	log.Println("Shutting down auth service...")
	grpcServer.GracefulStop()

	// Close database connections with timeout
	log.Println("Closing database connections...")

	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}

	if err := postgresClient.Close(); err != nil {
		log.Printf("Error closing PostgreSQL connection: %v", err)
	}

	log.Println("Auth service stopped")
}
