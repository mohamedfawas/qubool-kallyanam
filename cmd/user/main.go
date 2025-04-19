package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	userConfig "github.com/mohamedfawas/qubool-kallyanam/internal/user/config"
	userHandlers "github.com/mohamedfawas/qubool-kallyanam/internal/user/handlers"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/postgres"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/log"
)

func main() {
	// Load configuration
	cfg, err := userConfig.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Set Gin mode based on environment
	if cfg.Common.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize logger
	logger, err := log.NewLogger("user", cfg.Common.Debug)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize Postgres database connection
	pgConfig := postgres.Config{
		Host:     cfg.Database.Postgres.Host,
		Port:     cfg.Database.Postgres.Port,
		Username: cfg.Database.Postgres.Username,
		Password: cfg.Database.Postgres.Password,
		Database: cfg.Database.Postgres.Database,
		SSLMode:  cfg.Database.Postgres.SSLMode,
		MaxConns: cfg.Database.Postgres.MaxConns,
		MaxIdle:  cfg.Database.Postgres.MaxIdle,
		Timeout:  cfg.Database.Postgres.Timeout,
	}
	pgClient, err := postgres.NewClient(pgConfig, "user", logger)
	if err != nil {
		logger.Fatal("Failed to connect to Postgres", zap.Error(err))
	}
	// Ensure database is closed on exit
	defer func() {
		if err := pgClient.Close(); err != nil {
			logger.Error("Error closing Postgres connection", zap.Error(err))
		}
	}()

	// Initialize Redis client
	redisConfig := redis.Config{
		Host:     cfg.Database.Redis.Host,
		Port:     cfg.Database.Redis.Port,
		Password: cfg.Database.Redis.Password,
		DB:       cfg.Database.Redis.DB,
		MaxConns: cfg.Database.Redis.MaxConns,
		MinIdle:  cfg.Database.Redis.MinIdle,
		Timeout:  cfg.Database.Redis.Timeout,
	}
	redisClient, err := redis.NewClient(context.Background(), redisConfig, "user", logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	// Ensure Redis client is closed on exit
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("Error closing Redis connection", zap.Error(err))
		}
	}()

	// Initialize router
	router := gin.New()
	router.Use(gin.Recovery())

	// Register handlers
	healthHandler := userHandlers.NewHealthHandler(logger, pgClient, redisClient)
	healthHandler.Register(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting user service",
			zap.String("host", cfg.Server.Host),
			zap.Int("port", cfg.Server.Port))

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down user service...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("User service exited")
}
