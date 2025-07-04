package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	paymentpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/payment/v1"
	pgdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/messaging/rabbitmq"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/payment/razorpay"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/adapters/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/services"
	grpcv1 "github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/handlers/grpc/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/handlers/health"
	httpv1 "github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/handlers/http/v1"
)

type Server struct {
	config       *config.Config
	logger       logging.Logger
	grpcServer   *grpc.Server
	httpServer   *http.Server
	pgClient     *pgdb.Client
	rabbitClient *rabbitmq.Client
}

func NewServer(cfg *config.Config, logger logging.Logger) (*Server, error) {
	// Database connection
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

	// RabbitMQ connection
	rabbitClient, err := rabbitmq.NewClient(cfg.RabbitMQ.DSN, cfg.RabbitMQ.ExchangeName)
	if err != nil {
		return nil, fmt.Errorf("failed to create RabbitMQ client: %w", err)
	}

	// Auto-migrate database
	if err := autoMigrate(pgClient.DB); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Setup gRPC server
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			createLoggingInterceptor(logger),
			createErrorInterceptor(),
		),
	)

	// Setup HTTP server
	gin.SetMode(gin.ReleaseMode)
	httpRouter := gin.New()
	httpRouter.Use(gin.Recovery())

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      httpRouter,
		ReadTimeout:  time.Duration(cfg.HTTP.ReadTimeoutSecs) * time.Second,
		WriteTimeout: time.Duration(cfg.HTTP.WriteTimeoutSecs) * time.Second,
		IdleTimeout:  time.Duration(cfg.HTTP.IdleTimeoutSecs) * time.Second,
	}

	// Register services
	if err := registerServices(grpcServer, httpRouter, pgClient.DB, cfg, logger, rabbitClient); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return &Server{
		config:       cfg,
		logger:       logger,
		grpcServer:   grpcServer,
		httpServer:   httpServer,
		pgClient:     pgClient,
		rabbitClient: rabbitClient,
	}, nil
}

func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Payment{},
		&models.Subscription{},
	)
}

func registerServices(
	grpcServer *grpc.Server,
	httpRouter *gin.Engine,
	db *gorm.DB,
	cfg *config.Config,
	logger logging.Logger,
	rabbitClient *rabbitmq.Client,
) error {
	// Register health service for gRPC
	health.RegisterHealthService(grpcServer, db)

	// Initialize dependencies
	paymentRepo := postgres.NewPaymentRepository(db)
	razorpayService := razorpay.NewService(cfg.Razorpay.KeyID, cfg.Razorpay.KeySecret)

	paymentService := services.NewPaymentService(services.PaymentServiceConfig{
		PaymentRepo:     paymentRepo,
		RazorpayService: razorpayService,
		PlansConfig:     &cfg.Plans,
		Logger:          logger,
		MessageBroker:   rabbitClient,
	})

	// Register gRPC handlers
	grpcHandler := grpcv1.NewPaymentHandler(paymentService, logger)
	paymentpb.RegisterPaymentServiceServer(grpcServer, grpcHandler)

	// Register HTTP handlers
	httpHandler := httpv1.NewHTTPHandler(paymentService, cfg, logger)
	setupHTTPRoutes(httpRouter, httpHandler)

	logger.Info("Payment services registered successfully")
	return nil
}

func setupHTTPRoutes(router *gin.Engine, handler *httpv1.HTTPHandler) {
	// Health check
	router.GET("/health", handler.HealthCheck)

	// Payment UI routes
	paymentGroup := router.Group("/payment")
	{
		paymentGroup.GET("/plans", handler.ShowPlans)
		paymentGroup.GET("/checkout", handler.ShowPaymentPage)
		paymentGroup.GET("/success", handler.ShowSuccessPage)
		paymentGroup.GET("/failed", handler.ShowFailedPage)
		paymentGroup.GET("/verify", handler.VerifyPaymentCallback)
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

func (s *Server) Start() error {
	// Start HTTP server in a goroutine
	go func() {
		s.logger.Info("Starting HTTP server", "port", s.config.HTTP.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("Failed to start HTTP server", "error", err)
		}
	}()

	// Start gRPC server (blocking)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.config.GRPC.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("Starting gRPC server", "port", s.config.GRPC.Port)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop() {
	s.logger.Info("Stopping servers")

	// Stop gRPC server
	s.grpcServer.GracefulStop()

	// Stop HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server forced to shutdown", "error", err)
	}

	// Close database and messaging connections
	if s.pgClient != nil {
		s.pgClient.Close()
	}
	if s.rabbitClient != nil {
		s.rabbitClient.Close()
	}
}
