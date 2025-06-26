package server

import (
	"fmt"
	"net"

	adminpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/admin/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/health" // Add this line
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/clients/auth"
	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/clients/user"
	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/domain/services"
	v1 "github.com/mohamedfawas/qubool-kallyanam/services/admin/internal/handlers/grpc/v1"
	"google.golang.org/grpc"
)

type Server struct {
	config     *config.Config
	grpcServer *grpc.Server
	authClient *auth.Client
	userClient *user.Client
	logger     logging.Logger
}

func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize logger
	logger := logging.Default()

	// Initialize service clients
	authClient, err := auth.NewClient(cfg.Services.Auth.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth client: %w", err)
	}

	userClient, err := user.NewClient(cfg.Services.User.Address)
	if err != nil {
		authClient.Close()
		return nil, fmt.Errorf("failed to create user client: %w", err)
	}

	// Create admin service with logger
	adminService := services.NewAdminService(authClient, userClient, logger)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register handlers
	adminHandler := v1.NewAdminHandler(adminService, logger)
	adminpb.RegisterAdminServiceServer(grpcServer, adminHandler)

	health.RegisterHealthService(grpcServer, nil, nil, nil)
	return &Server{
		config:     cfg,
		grpcServer: grpcServer,
		authClient: authClient,
		userClient: userClient,
		logger:     logger,
	}, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPC.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.config.GRPC.Port, err)
	}

	s.logger.Info("Admin gRPC server starting", "port", s.config.GRPC.Port)
	return s.grpcServer.Serve(listener)
}

func (s *Server) Stop() {
	s.logger.Info("Stopping admin server")
	if s.authClient != nil {
		s.authClient.Close()
	}
	if s.userClient != nil {
		s.userClient.Close()
	}
	s.grpcServer.GracefulStop()
}
