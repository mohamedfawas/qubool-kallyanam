package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/mongodb"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/handlers/health"
	"google.golang.org/grpc"
)

// Server represents the gRPC server
type Server struct {
	config      *config.Config
	grpcServer  *grpc.Server
	mongoClient *mongodb.Client
}

// NewServer creates a new gRPC server
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize MongoDB client
	mongoClient, err := mongodb.NewClient(&mongodb.Config{
		URI:      cfg.Database.MongoDB.URI,
		Database: cfg.Database.MongoDB.Database,
		Timeout:  time.Duration(cfg.Database.MongoDB.TimeoutSeconds) * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register health service
	health.RegisterHealthService(grpcServer, mongoClient.GetClient())

	return &Server{
		config:      cfg,
		grpcServer:  grpcServer,
		mongoClient: mongoClient,
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPC.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	return s.grpcServer.Serve(lis)
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()

	// Close database connections
	if s.mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.mongoClient.Close(ctx)
	}
}
