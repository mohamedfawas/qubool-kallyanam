package health

import (
	"github.com/mohamedfawas/qubool-kallyanam/pkg/health"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// RegisterHealthService registers the health service
func RegisterHealthService(grpcServer *grpc.Server, db *gorm.DB, redisClient *redis.Client) *health.Service {
	return health.RegisterHealthService(grpcServer, db, redisClient, nil)
}
