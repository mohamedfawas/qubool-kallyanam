package redis

import (
	"context"
	"time"

	redisdb "github.com/mohamedfawas/qubool-kallyanam/pkg/database/redis"
	"github.com/mohamedfawas/qubool-kallyanam/services/auth/internal/domain/repositories"
)

type OTPRepo struct {
	client *redisdb.Client
}

func NewOTPRepository(client *redisdb.Client) repositories.OTPRepository {
	return &OTPRepo{
		client: client,
	}
}

func (r *OTPRepo) GetOTP(ctx context.Context, key string) (string, error) {
	return r.client.Get(ctx, key)
}

func (r *OTPRepo) StoreOTP(ctx context.Context, key string, otp string, expiry time.Duration) error {
	return r.client.Set(ctx, key, otp, expiry)
}

func (r *OTPRepo) DeleteOTP(ctx context.Context, key string) error {
	return r.client.Del(ctx, key)
}
