package health

import (
	"github.com/mohamedfawas/qubool-kallyanam/pkg/health"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc"
)

func RegisterHealthService(grpcServer *grpc.Server, mongoClient *mongo.Client) *health.Service {
	return health.RegisterHealthService(grpcServer, nil, nil, mongoClient)
}
