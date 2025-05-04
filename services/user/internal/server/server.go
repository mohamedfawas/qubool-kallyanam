package server

import (
	"fmt"
	"net"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/handlers/health"
	"google.golang.org/grpc"
)

// Server represents the gRPC server
type Server struct {
	config      *config.Config
	grpcServer  *grpc.Server
	pgClient    *postgres.Client
	redisClient *redis.Client
}

// NewServer creates a new gRPC server
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize PostgreSQL client
	pgClient, err := postgres.NewClient(&postgres.Config{
		Host:     cfg.Database.Postgres.Host,
		Port:     fmt.Sprintf("%d", cfg.Database.Postgres.Port),
		User:     cfg.Database.Postgres.User,
		Password: cfg.Database.Postgres.Password,
		DBName:   cfg.Database.Postgres.DBName,
		SSLMode:  cfg.Database.Postgres.SSLMode,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres client: %w", err)
	}

	// Initialize Redis client
	redisClient, err := redis.NewClient(&redis.Config{
		Host:     cfg.Database.Redis.Host,
		Port:     fmt.Sprintf("%d", cfg.Database.Redis.Port),
		Password: cfg.Database.Redis.Password,
		DB:       cfg.Database.Redis.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register health service
	health.RegisterHealthService(grpcServer, pgClient.DB, redisClient.GetClient())

	return &Server{
		config:      cfg,
		grpcServer:  grpcServer,
		pgClient:    pgClient,
		redisClient: redisClient,
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
	if s.pgClient != nil {
		s.pgClient.Close()
	}

	if s.redisClient != nil {
		s.redisClient.Close()
	}
}
