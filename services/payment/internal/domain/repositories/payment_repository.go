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
	GetUserPayments(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*models.Payment, int32, error)

	// Subscription operations
	CreateSubscription(ctx context.Context, subscription *models.Subscription) error
	GetUserActiveSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error)
	GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error)
	UpdateSubscription(ctx context.Context, subscription *models.Subscription) error
	DeactivateExpiredSubscriptions(ctx context.Context) error
}
