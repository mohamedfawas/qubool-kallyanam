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

	chatConfig "github.com/mohamedfawas/qubool-kallyanam/internal/chat/config"
	chatHandlers "github.com/mohamedfawas/qubool-kallyanam/internal/chat/handlers"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/mongodb"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/db/redis"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/log"
)

func main() {
	// Create a background context for the application
	ctx := context.Background()

	// Load configuration
	cfg, err := chatConfig.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Set Gin mode based on environment
	if cfg.Common.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize logger
	logger, err := log.NewLogger("chat", cfg.Common.Debug)
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize MongoDB client
	mongoConfig := mongodb.Config{
		URI:      cfg.Database.MongoDB.URI,
		Database: cfg.Database.MongoDB.Database,
		Username: cfg.Database.MongoDB.Username,
		Password: cfg.Database.MongoDB.Password,
		MaxConns: cfg.Database.MongoDB.MaxConns,
		MinConns: cfg.Database.MongoDB.MinConns,
		Timeout:  cfg.Database.MongoDB.Timeout,
	}
	mongoClient, err := mongodb.NewClient(ctx, mongoConfig, "chat", logger)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	// Ensure MongoDB is closed on exit
	defer func() {
		if err := mongoClient.Close(ctx); err != nil {
			logger.Error("Error closing MongoDB connection", zap.Error(err))
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
	redisClient, err := redis.NewClient(ctx, redisConfig, "chat", logger)
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
	healthHandler := chatHandlers.NewHealthHandler(logger, mongoClient, redisClient)
	healthHandler.Register(router)

	// Create HTTP server
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting chat service",
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

	logger.Info("Shutting down chat service...")

	// Create a deadline for graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Chat service exited")
}
