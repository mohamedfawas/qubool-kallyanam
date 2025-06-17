package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/models"
)

type PaymentRepository interface {
	// Payment operations
	CreatePayment(ctx context.Context, payment *models.Payment) error
	GetPaymentByOrderID(ctx context.Context, orderID string) (*models.Payment, error)
	GetPaymentByID(ctx context.Context, id uuid.UUID) (*models.Payment, error)
	UpdatePayment(ctx context.Context, payment *models.Payment) error
	GetPaymentsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Payment, int64, error)

	// Subscription operations
	CreateSubscription(ctx context.Context, subscription *models.Subscription) error
	GetActiveSubscriptionByUserID(ctx context.Context, userID uuid.UUID) (*models.Subscription, error)
	GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, subscription *models.Subscription) error
	GetSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error)

	// Transaction support
	WithTx(ctx context.Context, fn func(context.Context) error) error
}
