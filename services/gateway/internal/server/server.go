package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/metrics"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/tracing"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/admin" // ✅ Move here
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/auth"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/chat"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/payment"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/user"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/config"
	adminHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/admin" // ✅ Move here
	authHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/auth"
	chatHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/chat"
	paymentHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/payment"
	userHandler "github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/handlers/v1/user"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

// Server represents the HTTP server
type Server struct {
	config        *config.Config
	logger        logging.Logger
	httpServer    *http.Server
	router        *gin.Engine
	authClient    *auth.Client
	userClient    *user.Client
	chatClient    *chat.Client
	paymentClient *payment.Client
	adminClient   *admin.Client
	jwtManager    *jwt.Manager
	auth          *middleware.Auth
	metrics       *metrics.Metrics
	tracer        tracing.Tracer
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, logger logging.Logger) (*Server, error) {
	// Set Gin mode to "release" if running in production to disable debug logs
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create a new Gin router instance for handling incoming HTTP requests
	router := gin.New()

	// Initialize metrics registry
	metricsRegistry := metrics.New("gateway")

	// Initialize tracing
	tracingConfig := tracing.Config{
		Enabled:     cfg.Tracing.Enabled,
		ServiceName: "qubool-gateway",
		Environment: cfg.Environment,
		JaegerURL:   cfg.Tracing.JaegerURL,
		SampleRate:  cfg.Tracing.SampleRate,
	}

	tracer, err := tracing.NewTracer(tracingConfig)
	if err != nil {
		logger.Error("Failed to initialize tracer", "error", err)
		// Continue without tracing instead of failing
		tracer, _ = tracing.NewTracer(tracing.Config{Enabled: false})
	}

	// Initialize the authentication service client using the Auth service address from config
	authClient, err := auth.NewClient(cfg.Services.Auth.Address)
	if err != nil {
		// Example: If Auth service is down or the address is incorrect, this error is returned
		return nil, fmt.Errorf("failed to create auth client: %w", err)
	}

	// Initialize the user service client using the User service address from config
	userClient, err := user.NewClient(cfg.Services.User.Address)
	if err != nil {
		// Example: If User service is unreachable, this error helps catch that early
		return nil, fmt.Errorf("failed to create user client: %w", err)
	}

	chatClient, err := chat.NewClient(cfg.Services.Chat.Address) // Add this block
	if err != nil {
		return nil, fmt.Errorf("failed to create chat client: %w", err)
	}

	paymentClient, err := payment.NewClient(cfg.Services.Payment.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment client: %w", err)
	}

	adminClient, err := admin.NewClient(cfg.Services.Admin.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin client: %w", err)
	}

	// Create JWT Manager for token validation
	jwtManager := jwt.NewManager(jwt.Config{
		SecretKey:       cfg.Auth.JWT.SecretKey,
		AccessTokenTTL:  time.Duration(cfg.Auth.JWT.AccessTokenMinutes) * time.Minute,
		RefreshTokenTTL: time.Duration(cfg.Auth.JWT.RefreshTokenDays) * 24 * time.Hour,
		Issuer:          cfg.Auth.JWT.Issuer,
	})

	// Create auth middleware
	auth := middleware.NewAuth(jwtManager)

	// Define the HTTP server with necessary configurations (port, timeouts, router)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTP.Port), // Sets the port to run on, e.g., ":8080"
		Handler: router,                            // Connect the Gin router to the server
		// ReadTimeout is the maximum duration for reading the entire request (including body).
		ReadTimeout: time.Second * time.Duration(cfg.HTTP.ReadTimeoutSecs),
		// WriteTimeout is the maximum duration for writing the response to the client.
		WriteTimeout: time.Second * time.Duration(cfg.HTTP.WriteTimeoutSecs), // Max time to write a response
		// IdleTimeout is the maximum time to wait for the next request when keep-alives are enabled.
		// This applies to persistent connections where the client can reuse the same connection.
		// Example: If a client stays idle for more than IdleTimeoutSecs = 30, the server closes the connection.
		IdleTimeout: time.Second * time.Duration(cfg.HTTP.IdleTimeoutSecs), // Timeout when idle
	}

	// Create the Server instance with all components
	server := &Server{
		config:        cfg,
		logger:        logger,
		httpServer:    httpServer,
		router:        router,
		authClient:    authClient,
		userClient:    userClient,
		chatClient:    chatClient,
		paymentClient: paymentClient,
		adminClient:   adminClient,
		jwtManager:    jwtManager,
		auth:          auth,
		metrics:       metricsRegistry,
		tracer:        tracer,
	}

	// Register all API routes
	server.setupRoutes()

	return server, nil
}

// setupRoutes configures all routes
func (s *Server) setupRoutes() {
	s.router.Static("/static", "./static")

	authHandler := authHandler.NewHandler(s.authClient, s.logger, s.metrics)
	userHandler := userHandler.NewHandler(s.userClient, s.logger, s.metrics)
	chatHandler := chatHandler.NewHandler(s.chatClient, s.userClient, s.logger, s.metrics)
	paymentHandler := paymentHandler.NewHandler(s.paymentClient, s.logger, s.metrics)
	adminHandler := adminHandler.NewHandler(s.adminClient, s.logger)
	SetupRouter(s.router,
		authHandler,
		userHandler,
		chatHandler,
		paymentHandler,
		adminHandler,
		s.auth,
		s.metrics,
		s.logger)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", "port", s.config.HTTP.Port)
	return s.httpServer.ListenAndServe()
}

// Stop stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping HTTP server")
	if s.authClient != nil {
		s.authClient.Close()
	}
	if s.userClient != nil {
		s.userClient.Close()
	}
	if s.chatClient != nil {
		s.chatClient.Close()
	}

	if s.paymentClient != nil {
		s.paymentClient.Close()
	}
	if s.adminClient != nil {
		s.adminClient.Close()
	}
	// Shutdown tracer
	if s.tracer != nil {
		s.tracer.Shutdown(ctx)
	}
	return s.httpServer.Shutdown(ctx)
}
