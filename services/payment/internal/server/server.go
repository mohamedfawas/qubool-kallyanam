package server

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	paymentpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/payment/v1"
	pgdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/payment/razorpay"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/adapters/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/services"
	v1 "github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/handlers/grpc/v1"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/handlers/health"
)

type Server struct {
	config     *config.Config
	logger     logging.Logger
	grpcServer *grpc.Server
	pgClient   *pgdb.Client
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

	// Auto-migrate the models
	if err := autoMigrate(pgClient.DB); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			createLoggingInterceptor(logger),
			createErrorInterceptor(),
		),
	)

	if err := registerServices(grpcServer, pgClient.DB, cfg, logger); err != nil {
		return nil, fmt.Errorf("failed to register services: %w", err)
	}

	return &Server{
		config:     cfg,
		logger:     logger,
		grpcServer: grpcServer,
		pgClient:   pgClient,
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
	db *gorm.DB,
	cfg *config.Config,
	logger logging.Logger,
) error {
	// Register health service
	health.RegisterHealthService(grpcServer, db)

	// Create repositories
	paymentRepo := postgres.NewPaymentRepository(db)

	// Create Razorpay service
	razorpayService := razorpay.NewService(cfg.Razorpay.KeyID, cfg.Razorpay.KeySecret)

	// Create payment service
	paymentService := services.NewPaymentService(paymentRepo, razorpayService, logger)

	// Create payment handler
	paymentHandler := v1.NewPaymentHandler(paymentService, logger)

	// Register services
	paymentpb.RegisterPaymentServiceServer(grpcServer, paymentHandler)

	logger.Info("Payment services registered successfully")
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
	s.grpcServer.GracefulStop()
	if s.pgClient != nil {
		s.pgClient.Close()
	}
}
