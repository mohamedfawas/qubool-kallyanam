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
	firestoreClient "github.com/mohamedfawas/qubool-kallyanam/pkg/database/firestore"
	mongoClient "github.com/mohamedfawas/qubool-kallyanam/pkg/database/mongodb"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	firestoreAdapter "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/adapters/firestore"
	mongoAdapter "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/adapters/mongodb"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/config"
	repositories "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/repository" // ADD THIS LINE
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/domain/services"
	v1 "github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/handlers/grpc/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/chat/internal/handlers/health"
)

type Server struct {
	config     *config.Config
	logger     logging.Logger
	grpcServer *grpc.Server

	// Simple approach - store actual clients directly
	mongoClient     *mongoClient.Client     // Will be nil if using Firestore
	firestoreClient *firestoreClient.Client // Will be nil if using MongoDB
}

func NewServer(cfg *config.Config, logger logging.Logger) (*Server, error) {
	server := &Server{
		config: cfg,
		logger: logger,
	}

	// Step 1: Create database clients based on configuration
	if err := server.createDatabaseClients(); err != nil {
		return nil, err
	}

	// Step 2: Create gRPC server with interceptors
	server.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			createLoggingInterceptor(logger),
			createErrorInterceptor(),
		),
	)

	// Step 3: Register services
	if err := server.registerServices(); err != nil {
		return nil, err
	}

	return server, nil
}

// Simple helper function to create database clients
func (s *Server) createDatabaseClients() error {
	dbType := s.config.GetDatabaseType()

	if dbType == "mongodb" {
		// Create MongoDB client
		client, err := mongoClient.NewClient(s.config.GetMongoDBConfig())
		if err != nil {
			return fmt.Errorf("failed to create MongoDB client: %w", err)
		}
		s.mongoClient = client
		s.logger.Info("Using MongoDB database")

	} else if dbType == "firestore" {
		// Create Firestore client
		client, err := firestoreClient.NewClient(s.config.GetFirestoreConfig())
		if err != nil {
			return fmt.Errorf("failed to create Firestore client: %w", err)
		}
		s.firestoreClient = client
		s.logger.Info("Using Firestore database")

	} else {
		return fmt.Errorf("unsupported database type: %s. Use 'mongodb' or 'firestore'", dbType)
	}

	return nil
}

// Simple helper function to register services
func (s *Server) registerServices() error {
	// Step 1: Create repositories based on which database client we have
	var conversationRepo repositories.ConversationRepository
	var messageRepo repositories.MessageRepository

	if s.mongoClient != nil {
		// We're using MongoDB
		conversationRepo = mongoAdapter.NewConversationRepository(s.mongoClient.GetDatabase())
		messageRepo = mongoAdapter.NewMessageRepository(s.mongoClient.GetDatabase())

		// Register health service for MongoDB
		health.RegisterHealthService(s.grpcServer, s.mongoClient.GetClient())

	} else if s.firestoreClient != nil {
		// We're using Firestore
		conversationRepo = firestoreAdapter.NewConversationRepository(s.firestoreClient.GetClient())
		messageRepo = firestoreAdapter.NewMessageRepository(s.firestoreClient.GetClient())

		// Note: Health service for Firestore can be added later if needed

	} else {
		return fmt.Errorf("no database client available")
	}

	// Step 2: Create business service (same regardless of database)
	chatService := services.NewChatService(conversationRepo, messageRepo, s.logger)

	// Step 3: Create and register gRPC handler (same regardless of database)
	chatHandler := v1.NewChatHandler(chatService, s.logger)
	chatpb.RegisterChatServiceServer(s.grpcServer, chatHandler)

	s.logger.Info("Services registered successfully")
	return nil
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

	// Stop gRPC server gracefully
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	// Close database connections
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.mongoClient != nil {
		s.logger.Info("Closing MongoDB connection")
		s.mongoClient.Close(ctx)
	}

	if s.firestoreClient != nil {
		s.logger.Info("Closing Firestore connection")
		s.firestoreClient.Close()
	}
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
