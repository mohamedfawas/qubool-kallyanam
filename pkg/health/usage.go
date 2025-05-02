package health

import (
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"gorm.io/gorm"
)

// RegisterHealthService registers the health service with gRPC server
func RegisterHealthService(grpcServer *grpc.Server, db *gorm.DB, redisClient *redis.Client) *Service {
	// Create health service with 3-second timeout for checks
	healthService := NewService(3 * time.Second)

	// Add database checkers
	if db != nil {
		healthService.AddChecker(NewPostgresChecker(db, "postgres"))
	}

	if redisClient != nil {
		healthService.AddChecker(NewRedisChecker(redisClient, "redis"))
	}

	// Register with gRPC server
	grpc_health_v1.RegisterHealthServer(grpcServer, healthService)

	return healthService
}
