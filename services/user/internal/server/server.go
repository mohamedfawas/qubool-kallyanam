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
	"github.com/mohamedfawas/qubool-kallyanam/pkg/notifications/email"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/adapters/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/adapters/storage"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/domain/services"
	v1 "github.com/mohamedfawas/qubool-kallyanam/services/user/internal/handlers/grpc/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/user/internal/handlers/health"
)

// Server represents the gRPC server
type Server struct {
	config              *config.Config
	logger              logging.Logger
	grpcServer          *grpc.Server
	pgClient            *pgdb.Client
	redisClient         *redisdb.Client
	rabbitClient        *rabbitmq.Client
	profileService      *services.ProfileService
	photoService        *services.PhotoService
	matchmakingService  *services.MatchmakingService
	notificationService *services.NotificationService
	jwtManager          *jwt.Manager
	photoStorage        storage.PhotoStorage
}

// CompositeHandler combines all specialized handlers
type CompositeHandler struct {
	*v1.ProfileHandler
	*v1.PartnerPreferencesHandler
	*v1.PhotoHandler
	*v1.VideoHandler
	*v1.MatchHandler
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

	// Create RabbitMQ client
	rabbitClient, err := rabbitmq.NewClient(cfg.RabbitMQ.DSN, cfg.RabbitMQ.ExchangeName)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	// Create S3 config and photo storage
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

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			createLoggingInterceptor(logger),
			createErrorInterceptor(),
		),
	)

	// Register health service
	health.RegisterHealthService(grpcServer, pgClient.DB, redisClient.GetClient())

	// Create repositories with correct constructors
	profileRepo := postgres.NewProfileRepository(pgClient.DB)
	partnerPreferencesRepo := postgres.NewPartnerPreferencesRepository(pgClient.DB)
	photoRepo := postgres.NewPhotoRepository(pgClient.DB)
	videoRepo := postgres.NewVideoRepository(pgClient.DB)
	matchRepo := postgres.NewMatchRepository(pgClient.DB) // Fixed: Only takes *gorm.DB

	// Create email client
	emailClient, err := email.NewClient(email.Config{
		SMTPHost:     cfg.Email.SMTPHost,
		SMTPPort:     cfg.Email.SMTPPort,
		SMTPUsername: cfg.Email.Username,
		SMTPPassword: cfg.Email.Password,
		FromEmail:    cfg.Email.FromEmail,
		FromName:     cfg.Email.FromName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create email client: %w", err)
	}

	// Create services with correct constructors
	profileService := services.NewProfileService(
		profileRepo,
		partnerPreferencesRepo,
		photoRepo,
		videoRepo,
		logger,
	)

	partnerPreferencesService := services.NewPartnerPreferencesService(
		profileRepo,
		partnerPreferencesRepo,
		logger,
	)

	photoService := services.NewPhotoService(
		profileRepo,
		photoRepo,
		photoStorage,
		logger,
	)

	videoService := services.NewVideoService(
		profileRepo,
		videoRepo,
		photoStorage,
		logger,
	)

	notificationService := services.NewNotificationService(
		profileRepo,
		emailClient,
		logger,
	)

	matchmakingService := services.NewMatchmakingService(
		matchRepo,
		profileRepo,
		partnerPreferencesRepo,
		notificationService,
		logger,
		cfg,
	)

	// Create JWT manager
	jwtManager := jwt.NewManager(jwt.Config{
		SecretKey:       cfg.Auth.JWT.SecretKey,
		AccessTokenTTL:  time.Duration(15) * time.Minute,
		RefreshTokenTTL: time.Duration(7) * 24 * time.Hour,
		Issuer:          cfg.Auth.JWT.Issuer,
	})

	// Create modular handlers
	profileHandler := v1.NewProfileHandler(
		profileService,
		jwtManager,
		logger,
	)

	partnerPreferencesHandler := v1.NewPartnerPreferencesHandler(
		partnerPreferencesService,
		jwtManager,
		logger,
	)

	photoHandler := v1.NewPhotoHandler(
		photoService,
		jwtManager,
		logger,
	)

	videoHandler := v1.NewVideoHandler(
		videoService,
		jwtManager,
		logger,
	)

	matchHandler := v1.NewMatchHandler(
		matchmakingService,
		jwtManager,
		logger,
	)

	// Create composite handler to combine all handlers
	compositeHandler := &CompositeHandler{
		ProfileHandler:            profileHandler,
		PartnerPreferencesHandler: partnerPreferencesHandler,
		PhotoHandler:              photoHandler,
		VideoHandler:              videoHandler,
		MatchHandler:              matchHandler,
	}

	// Register the composite handler
	userpb.RegisterUserServiceServer(grpcServer, compositeHandler)

	// Create server instance
	server := &Server{
		config:              cfg,
		logger:              logger,
		grpcServer:          grpcServer,
		pgClient:            pgClient,
		redisClient:         redisClient,
		rabbitClient:        rabbitClient,
		profileService:      profileService,
		matchmakingService:  matchmakingService,
		photoService:        photoService,
		notificationService: notificationService,
		jwtManager:          jwtManager,
		photoStorage:        photoStorage,
	}

	// Subscribe to events
	if err := server.subscribeToEvents(); err != nil {
		server.Stop()
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}

	// Initialize storage
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

	// Subscribe to user deletion events
	err = s.rabbitClient.Subscribe("user.deleted", s.handleUserDeletion)
	if err != nil {
		return fmt.Errorf("failed to subscribe to user.deleted events: %w", err)
	}

	s.logger.Info("Subscribed to user login and deletion events")
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

func (s *Server) handleUserLogin(message []byte) error {
	var event struct {
		UserID    string    `json:"user_id"`
		Phone     string    `json:"phone"`
		Email     string    `json:"email"`
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
		"email", event.Email,
		"eventType", event.EventType)

	// Process the login event using the service layer
	ctx := context.Background()
	err := s.profileService.HandleUserLogin(ctx, event.UserID, event.Phone, event.Email, event.LastLogin)
	if err != nil {
		s.logger.Error("Failed to process login event", "error", err, "userID", event.UserID)
		return err
	}

	s.logger.Info("Successfully processed login event", "userID", event.UserID)
	return nil
}

func (s *Server) handleUserDeletion(message []byte) error {
	var event struct {
		UserID    string    `json:"user_id"`
		EventType string    `json:"event_type"`
		Timestamp time.Time `json:"timestamp"`
	}

	if err := json.Unmarshal(message, &event); err != nil {
		s.logger.Error("Failed to unmarshal user deletion event", "error", err)
		return err
	}

	s.logger.Info("Received user deletion event",
		"userID", event.UserID,
		"eventType", event.EventType)

	// Delegate business logic to service layer
	ctx := context.Background()
	if err := s.profileService.HandleUserDeletion(ctx, event.UserID); err != nil {
		s.logger.Error("Failed to process user deletion event", "error", err, "userID", event.UserID)
		return err
	}

	s.logger.Info("Successfully processed user deletion event", "userID", event.UserID)
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
			if _, ok := status.FromError(err); ok {
				return resp, err
			}
			return resp, status.Error(codes.Internal, "Internal server error")
		}
		return resp, nil
	}
}
