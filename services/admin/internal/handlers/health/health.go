package health

import (
	"github.com/mohamedfawas/qubool-kallyanam/pkg/health"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

// RegisterHealthService registers the health service
func RegisterHealthService(grpcServer *grpc.Server, db *gorm.DB) *health.Service {
	return health.RegisterHealthService(grpcServer, db, nil, nil)
}
