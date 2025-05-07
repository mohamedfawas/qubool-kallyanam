package server

import (
	"context"
	"fmt"
	"net"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	authpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
	pgdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/messaging/rabbitmq"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/adapters/postgres"

	"time"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	redisdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/notifications/email"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/otp"
	redisAdapter "github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/adapters/redis"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/services"
	v1 "github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/handlers/grpc/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/handlers/health"
)

// Server represents the gRPC server
type Server struct {
	config       *config.Config
	logger       logging.Logger
	grpcServer   *grpc.Server
	pgClient     *pgdb.Client
	redisClient  *redisdb.Client
	rabbitClient *rabbitmq.Client
}

// NewServer creates a new gRPC server
func NewServer(cfg *config.Config, logger logging.Logger) (*Server, error) {
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

	// Create gRPC server with options for better error handling
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			// Add logging interceptor
			createLoggingInterceptor(logger),
			// Add error interceptor
			createErrorInterceptor(),
		),
	)

	// Configure and register all services
	if err := registerServices(grpcServer,
		pgClient.DB,
		redisClient.GetClient(),
		cfg,
		logger,
		rabbitClient); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return &Server{
		config:       cfg,
		logger:       logger,
		grpcServer:   grpcServer,
		pgClient:     pgClient,
		redisClient:  redisClient,
		rabbitClient: rabbitClient,
	}, nil
}

// registerServices sets up and registers all gRPC services
func registerServices(
	grpcServer *grpc.Server,
	db *gorm.DB,
	redisClient *redis.Client,
	cfg *config.Config,
	logger logging.Logger,
	rabbitClient *rabbitmq.Client,
) error {
	// Register health service
	health.RegisterHealthService(grpcServer, db, redisClient)

	// Create repository
	registrationRepo := postgres.NewRegistrationRepository(db)
	tokenRepo := redisAdapter.NewTokenRepository(redisClient)
	adminRepo := postgres.NewAdminRepository(db)

	// Create JWT Manager
	jwtManager := jwt.NewManager(jwt.Config{
		SecretKey:       cfg.Auth.JWT.SecretKey,
		AccessTokenTTL:  time.Duration(cfg.Auth.JWT.AccessTokenMinutes) * time.Minute,
		RefreshTokenTTL: time.Duration(cfg.Auth.JWT.RefreshTokenDays) * 24 * time.Hour,
		Issuer:          cfg.Auth.JWT.Issuer,
	})

	// Set up OTP components
	otpConfig := otp.DefaultConfig()
	otpGenerator := otp.NewGenerator(otpConfig)
	otpStore := otp.NewStore(redisClient, otpConfig)

	// Set up email client
	emailClient, err := email.NewClient(email.Config{
		SMTPHost:     cfg.Email.SMTPHost,
		SMTPPort:     cfg.Email.SMTPPort,
		SMTPUsername: cfg.Email.Username,
		SMTPPassword: cfg.Email.Password,
		FromEmail:    cfg.Email.FromEmail,
		FromName:     cfg.Email.FromName,
	})
	if err != nil {
		return fmt.Errorf("failed to create email client: %w", err)
	}

	// Create service
	registrationService := services.NewRegistrationService(
		registrationRepo,
		otpGenerator,
		otpStore,
		emailClient,
		logger,
	)

	// Create admin service
	adminService := services.NewAdminService(adminRepo, logger)

	// Initialize default admin
	if err := adminService.InitializeDefaultAdmin(
		context.Background(),
		cfg.Admin.DefaultEmail,
		cfg.Admin.DefaultPassword,
	); err != nil {
		logger.Error("Failed to initialize default admin", "error", err)
		// Continue execution, don't fail startup
	}

	// Create auth service with admin repository
	authService := services.NewAuthService(
		registrationRepo,
		tokenRepo,
		adminRepo, // Add this parameter
		jwtManager,
		logger,
		time.Duration(cfg.Auth.JWT.AccessTokenMinutes)*time.Minute,
		time.Duration(cfg.Auth.JWT.RefreshTokenDays)*24*time.Hour,
		rabbitClient,
	)

	// Create and register auth handler
	authHandler := v1.NewAuthHandler(registrationService,
		authService,
		logger)
	authpb.RegisterAuthServiceServer(grpcServer, authHandler)

	return nil
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

// Start starts the gRPC server
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPC.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("Starting gRPC server", "port", s.config.GRPC.Port)
	return s.grpcServer.Serve(lis)
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	s.logger.Info("Stopping gRPC server")
	s.grpcServer.GracefulStop()

	// Close database connections
	if s.pgClient != nil {
		s.pgClient.Close()
	}

	if s.redisClient != nil {
		s.redisClient.Close()
	}

	if s.rabbitClient != nil {
		s.rabbitClient.Close()
	}
}
