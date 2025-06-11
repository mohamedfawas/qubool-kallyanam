package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/chat/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/database/mongodb"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	mongoAdapter "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/adapters/mongodb"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/services"
	v1 "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/handlers/grpc/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/handlers/health"
)

type Server struct {
	config      *config.Config
	logger      logging.Logger
	grpcServer  *grpc.Server
	mongoClient *mongodb.Client
}

func NewServer(cfg *config.Config, logger logging.Logger) (*Server, error) {
	// Create MongoDB client using pkg/database/mongodb
	mongoClient, err := mongodb.NewClient(cfg.GetMongoDBConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB client: %w", err)
	}

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			createLoggingInterceptor(logger),
			createErrorInterceptor(),
		),
	)

	// Register services
	if err := registerServices(grpcServer, mongoClient, cfg, logger); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return &Server{
		config:      cfg,
		logger:      logger,
		grpcServer:  grpcServer,
		mongoClient: mongoClient,
	}, nil
}

func registerServices(
	grpcServer *grpc.Server,
	mongoClient *mongodb.Client,
	cfg *config.Config,
	logger logging.Logger,
) error {
	// Register health service using pkg/health
	health.RegisterHealthService(grpcServer, mongoClient.GetClient())

	// Create repositories
	conversationRepo := mongoAdapter.NewConversationRepository(mongoClient.GetDatabase())
	messageRepo := mongoAdapter.NewMessageRepository(mongoClient.GetDatabase())

	// Create services
	chatService := services.NewChatService(conversationRepo, messageRepo, logger)

	// Create and register gRPC handler
	chatHandler := v1.NewChatHandler(chatService, logger)
	chatpb.RegisterChatServiceServer(grpcServer, chatHandler)

	// TODO: Create indexes for MongoDB collections
	if err := createMongoIndexes(mongoClient); err != nil {
		logger.Error("Failed to create MongoDB indexes", "error", err)
		// Don't fail startup for index creation errors
	}

	return nil
}

func createMongoIndexes(mongoClient *mongodb.Client) error {
	// TODO: Implement MongoDB index creation in Phase 2
	// This will include:
	// - Index on conversations.participants
	// - Index on conversations.updated_at
	// - Index on messages.conversation_id + sent_at
	// - Index on messages.sender_id
	return nil
}

func createLoggingInterceptor(logger logging.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger.Info("gRPC request", "method", info.FullMethod)
		resp, err := handler(ctx, req)
		if err != nil {
			logger.Error("gRPC error", "method", info.FullMethod, "error", err)
		}
		return resp, err
	}
}

func createErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			if _, ok := status.FromError(err); ok {
				return resp, err
			}
			return resp, status.Error(codes.Internal, "Internal server error")
		}
		return resp, nil
	}
}

func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPC.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("Starting gRPC server", "port", s.config.GRPC.Port)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.logger.Info("Stopping gRPC server")

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	if s.mongoClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.mongoClient.Close(ctx)
	}
}
