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
	"gorm.io/gorm"

	authpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/auth/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	pgdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	redisdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/messaging/rabbitmq"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/notifications/email"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/security/otp"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/adapters/postgres"
	redisAdapter "github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/adapters/redis"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/services"
	v1 "github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/handlers/grpc/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/handlers/health"
)

type Server struct {
	config       *config.Config
	logger       logging.Logger
	grpcServer   *grpc.Server
	pgClient     *pgdb.Client
	redisClient  *redisdb.Client
	rabbitClient *rabbitmq.Client
	authService  *services.AuthService
}

func NewServer(cfg *config.Config, logger logging.Logger) (*Server, error) {

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

	redisClient, err := redisdb.NewClient(&redisdb.Config{
		Host:     cfg.Database.Redis.Host,
		Port:     fmt.Sprintf("%d", cfg.Database.Redis.Port),
		Password: cfg.Database.Redis.Password,
		DB:       cfg.Database.Redis.DB,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create redis client: %w", err)
	}

	rabbitClient, err := rabbitmq.NewClient(cfg.RabbitMQ.DSN, cfg.RabbitMQ.ExchangeName)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			createLoggingInterceptor(logger),
			createErrorInterceptor(),
		),
	)

	authService, err := registerServices(grpcServer,
		pgClient.DB,
		redisClient,
		cfg,
		logger,
		rabbitClient)
	if err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	server := &Server{
		config:       cfg,
		logger:       logger,
		grpcServer:   grpcServer,
		pgClient:     pgClient,
		redisClient:  redisClient,
		rabbitClient: rabbitClient,
		authService:  authService,
	}

	if err := server.subscribeToSubscriptionEvents(); err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %w", err)
	}

	return server, nil
}

func registerServices(
	grpcServer *grpc.Server,
	db *gorm.DB,
	redisClient *redisdb.Client,
	cfg *config.Config,
	logger logging.Logger,
	rabbitClient *rabbitmq.Client,
) (*services.AuthService, error) {

	health.RegisterHealthService(grpcServer, db, redisClient.GetClient())

	jwtManager := jwt.NewManager(jwt.Config{
		SecretKey:       cfg.Auth.JWT.SecretKey,
		AccessTokenTTL:  time.Duration(cfg.Auth.JWT.AccessTokenMinutes) * time.Minute,
		RefreshTokenTTL: time.Duration(cfg.Auth.JWT.RefreshTokenDays) * 24 * time.Hour,
		Issuer:          cfg.Auth.JWT.Issuer,
	})

	otpConfig := otp.DefaultConfig()
	otpGenerator := otp.NewGenerator(otpConfig)

	userRepo := postgres.NewUserRepository(db)
	registrationRepo := postgres.NewRegistrationRepository(db)
	adminRepo := postgres.NewAdminRepository(db)

	otpRepo := redisAdapter.NewOTPRepository(redisClient)
	tokenRepo := redisAdapter.NewTokenRepository(redisClient)

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

	registrationService := services.NewRegistrationService(
		registrationRepo,
		userRepo,
		otpRepo,
		otpGenerator,
		otpConfig.ExpiryTime,
		emailClient,
		logger,
	)

	adminService := services.NewAdminService(adminRepo, logger)

	if err := adminService.InitializeDefaultAdmin(
		context.Background(),
		cfg.Admin.DefaultEmail,
		cfg.Admin.DefaultPassword,
	); err != nil {
		logger.Error("Failed to initialize default admin", "error", err)

	}

	authService := services.NewAuthService(
		userRepo,
		tokenRepo,
		adminRepo,
		jwtManager,
		logger,
		time.Duration(cfg.Auth.JWT.AccessTokenMinutes)*time.Minute,
		time.Duration(cfg.Auth.JWT.RefreshTokenDays)*24*time.Hour,
		rabbitClient,
	)

	authHandler := v1.NewAuthHandler(
		registrationService,
		authService,
		logger,
	)
	authpb.RegisterAuthServiceServer(grpcServer, authHandler)

	return authService, nil
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

func (s *Server) subscribeToSubscriptionEvents() error {
	err := s.rabbitClient.Subscribe("subscription.activated", s.handleSubscriptionEvent)
	if err != nil {
		return fmt.Errorf("failed to subscribe to subscription.activated: %w", err)
	}

	err = s.rabbitClient.Subscribe("subscription.extended", s.handleSubscriptionEvent)
	if err != nil {
		return fmt.Errorf("failed to subscribe to subscription.extended: %w", err)
	}

	s.logger.Info("Subscribed to subscription events")
	return nil
}

func (s *Server) handleSubscriptionEvent(message []byte) error {
	var event struct {
		UserID       string    `json:"user_id"`
		PremiumUntil time.Time `json:"premium_until"`
		EventType    string    `json:"event_type"`
		PlanID       string    `json:"plan_id"`
		Timestamp    time.Time `json:"timestamp"`
	}

	if err := json.Unmarshal(message, &event); err != nil {
		s.logger.Error("Failed to unmarshal subscription event", "error", err)
		return err
	}

	s.logger.Info("Received subscription event",
		"userID", event.UserID,
		"eventType", event.EventType,
		"premiumUntil", event.PremiumUntil)

	ctx := context.Background()
	err := s.authService.UpdateUserPremiumStatus(ctx, event.UserID, event.PremiumUntil)
	if err != nil {
		s.logger.Error("Failed to update premium status", "userID", event.UserID, "error", err)
		return err
	}

	s.logger.Info("Successfully updated premium status", "userID", event.UserID)
	return nil
}
