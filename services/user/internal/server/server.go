// user/internal/server/server.go
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	userpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/user/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	s3config "github.com/mohamedfawas/qubool-kallyanam/pkg/cdn/s3"
	pgdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	redisdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/messaging/rabbitmq"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/adapters/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/adapters/storage"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/services"
	v1 "github.com/mohamedfawas/qubool-kallyanam/services/user/internal/handlers/grpc/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/handlers/health"
)

// Server represents the gRPC server
type Server struct {
	config         *config.Config
	logger         logging.Logger
	grpcServer     *grpc.Server
	pgClient       *pgdb.Client
	redisClient    *redisdb.Client
	rabbitClient   *rabbitmq.Client
	profileService *services.ProfileService
	photoService   *services.PhotoService
	jwtManager     *jwt.Manager
	photoStorage   storage.PhotoStorage
}

// NewServer creates a new gRPC server
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize logger
	logger := logging.Default()

	// Initialize PostgreSQL client
	pgClient, err := pgdb.NewClient(&pgdb.Config{
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
	redisClient, err := redisdb.NewClient(&redisdb.Config{
		Host:     cfg.Database.Redis.Host,
		Port:     fmt.Sprintf("%d", cfg.Database.Redis.Port),
		Password: cfg.Database.Redis.Password,
		DB:       cfg.Database.Redis.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	// Initialize RabbitMQ client
	rabbitClient, err := rabbitmq.NewClient(cfg.RabbitMQ.DSN, cfg.RabbitMQ.ExchangeName)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	// Initialize S3 storage - MOVED THIS BEFORE USING IT
	s3Cfg := s3config.NewConfig(
		cfg.Storage.S3.Endpoint,
		cfg.Storage.S3.Region,
		cfg.Storage.S3.AccessKeyID,
		cfg.Storage.S3.SecretAccessKey,
		cfg.Storage.S3.BucketName,
		cfg.Storage.S3.UseSSL,
	)

	photoStorage, err := storage.NewS3PhotoStorage(s3Cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create photo storage: %w", err)
	}

	// Create gRPC server with options for better error handling
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// Add logging interceptor
			createLoggingInterceptor(logger),
			// Add error interceptor
			createErrorInterceptor(),
		),
	)

	// Register health service
	health.RegisterHealthService(grpcServer, pgClient.DB, redisClient.GetClient())

	// Initialize repository layer
	profileRepo := postgres.NewProfileRepository(pgClient.DB)

	// Initialize service layer
	profileService := services.NewProfileService(profileRepo, logger)
	photoService := services.NewPhotoService(profileRepo, photoStorage, logger)

	jwtManager := jwt.NewManager(jwt.Config{
		SecretKey:       cfg.Auth.JWT.SecretKey,
		AccessTokenTTL:  time.Duration(15) * time.Minute,
		RefreshTokenTTL: time.Duration(7) * 24 * time.Hour,
		Issuer:          cfg.Auth.JWT.Issuer,
	})

	// Create profile handler
	profileHandler := v1.NewProfileHandler(profileService, photoService, jwtManager, logger)

	// Register the profile handler
	userpb.RegisterUserServiceServer(grpcServer, profileHandler)

	// Create server instance
	server := &Server{
		config:         cfg,
		logger:         logger,
		grpcServer:     grpcServer,
		pgClient:       pgClient,
		redisClient:    redisClient,
		rabbitClient:   rabbitClient,
		profileService: profileService,
		photoService:   photoService,
		jwtManager:     jwtManager,
		photoStorage:   photoStorage,
	}

	// Subscribe to events
	if err := server.subscribeToEvents(); err != nil {
		server.Stop() // Clean up resources if subscription fails
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Initialize S3 bucket
	if err := server.initializeStorage(); err != nil {
		server.Stop()
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	return server, nil
}

// subscribeToEvents sets up event subscriptions
func (s *Server) subscribeToEvents() error {
	// Subscribe to user login events
	err := s.rabbitClient.Subscribe("user.login", s.handleUserLogin)
	if err != nil {
		return fmt.Errorf("failed to subscribe to user.login events: %w", err)
	}

	s.logger.Info("Subscribed to user login events")
	return nil
}

func (s *Server) initializeStorage() error {
	s.logger.Info("Initializing storage...")
	ctx := context.Background()

	if err := s.photoStorage.EnsureBucketExists(ctx); err != nil {
		return fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	s.logger.Info("Storage initialized successfully")
	return nil
}

// handleUserLogin processes user login events
func (s *Server) handleUserLogin(message []byte) error {
	var event struct {
		UserID    string    `json:"user_id"`
		Phone     string    `json:"phone"`
		LastLogin time.Time `json:"last_login"`
		EventType string    `json:"event_type"`
	}

	if err := json.Unmarshal(message, &event); err != nil {
		s.logger.Error("Failed to unmarshal login event", "error", err)
		return err
	}

	s.logger.Info("Received login event",
		"userID", event.UserID,
		"phone", event.Phone,
		"eventType", event.EventType)

	// Process the login event using the service layer
	ctx := context.Background()
	err := s.profileService.HandleUserLogin(ctx, event.UserID, event.Phone, event.LastLogin)
	if err != nil {
		s.logger.Error("Failed to process login event", "error", err, "userID", event.UserID)
		return err
	}

	s.logger.Info("Successfully processed login event", "userID", event.UserID)
	return nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPC.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("Starting gRPC server", "port", s.config.GRPC.Port)
	return s.grpcServer.Serve(lis)
}

// Stop stops the gRPC server and cleans up resources
func (s *Server) Stop() {
	s.logger.Info("Stopping gRPC server")

	// Graceful shutdown of gRPC server
	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
	}

	// Close database connections
	if s.pgClient != nil {
		s.pgClient.Close()
	}

	if s.redisClient != nil {
		s.redisClient.Close()
	}

	// Close message broker connection
	if s.rabbitClient != nil {
		s.rabbitClient.Close()
	}

	s.logger.Info("Server stopped")
}

// Create a logging interceptor for gRPC
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

// Create an error interceptor for gRPC
func createErrorInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			// If the error is already a gRPC status error, return it as is
			if _, ok := status.FromError(err); ok {
				return resp, err
			}

			// Otherwise, convert it to an internal server error
			return resp, status.Error(codes.Internal, "Internal server error")
		}
		return resp, nil
	}
}
