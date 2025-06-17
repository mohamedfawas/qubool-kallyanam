package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/repositories"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/errors"
)

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) repositories.PaymentRepository {
	return &paymentRepository{db: db}
}

// Payment operations
func (r *paymentRepository) CreatePayment(ctx context.Context, payment *models.Payment) error {
	if err := r.db.WithContext(ctx).Create(payment).Error; err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

func (r *paymentRepository) GetPaymentByOrderID(ctx context.Context, orderID string) (*models.Payment, error) {
	var payment models.Payment
	if err := r.db.WithContext(ctx).Where("razorpay_order_id = ?", orderID).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by order ID: %w", err)
	}
	return &payment, nil
}

func (r *paymentRepository) GetPaymentByID(ctx context.Context, id uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&payment).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get payment by ID: %w", err)
	}
	return &payment, nil
}

func (r *paymentRepository) UpdatePayment(ctx context.Context, payment *models.Payment) error {
	if err := r.db.WithContext(ctx).Save(payment).Error; err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}
	return nil
}

func (r *paymentRepository) GetPaymentsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.Payment, int64, error) {
	var payments []*models.Payment
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.Payment{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	// Get payments with pagination
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&payments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get payments: %w", err)
	}

	return payments, total, nil
}

// Subscription operations
func (r *paymentRepository) CreateSubscription(ctx context.Context, subscription *models.Subscription) error {
	if err := r.db.WithContext(ctx).Create(subscription).Error; err != nil {
		return fmt.Errorf("failed to create subscription: %w", err)
	}
	return nil
}

func (r *paymentRepository) GetActiveSubscriptionByUserID(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, models.SubscriptionStatusActive).
		Where("end_date > NOW()").
		First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("failed to get active subscription: %w", err)
	}
	return &subscription, nil
}

func (r *paymentRepository) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrSubscriptionNotFound
		}
		return nil, fmt.Errorf("failed to get subscription by ID: %w", err)
	}
	return &subscription, nil
}

func (r *paymentRepository) UpdateSubscription(ctx context.Context, subscription *models.Subscription) error {
	if err := r.db.WithContext(ctx).Save(subscription).Error; err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}
	return nil
}

func (r *paymentRepository) GetSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.Subscription, error) {
	var subscriptions []*models.Subscription
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&subscriptions).Error; err != nil {
		return nil, fmt.Errorf("failed to get subscriptions: %w", err)
	}
	return subscriptions, nil
}

// Transaction support
func (r *paymentRepository) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create new repository instance with the transaction
		txRepo := &paymentRepository{db: tx}
		// Create new context with the transaction repository
		txCtx := context.WithValue(ctx, "tx_repo", txRepo)
		return fn(txCtx)
	})
}
