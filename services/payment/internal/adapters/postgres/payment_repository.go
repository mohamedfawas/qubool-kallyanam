package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/repositories"
	"gorm.io/gorm"
)

type PaymentRepo struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) repositories.PaymentRepository {
	return &PaymentRepo{
		db: db,
	}
}

// Payment operations
func (r *PaymentRepo) CreatePayment(ctx context.Context, payment *models.Payment) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

func (r *PaymentRepo) GetPaymentByOrderID(ctx context.Context, orderID string) (*models.Payment, error) {
	var payment models.Payment
	result := r.db.WithContext(ctx).Where("razorpay_order_id = ?", orderID).First(&payment)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &payment, nil
}

func (r *PaymentRepo) GetPaymentByID(ctx context.Context, id uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&payment)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &payment, nil
}

func (r *PaymentRepo) UpdatePayment(ctx context.Context, payment *models.Payment) error {
	return r.db.WithContext(ctx).Save(payment).Error
}

func (r *PaymentRepo) GetUserPayments(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*models.Payment, int32, error) {
	var payments []*models.Payment
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.Payment{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get payments with pagination
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(int(limit)).
		Offset(int(offset)).
		Find(&payments).Error; err != nil {
		return nil, 0, err
	}

	return payments, int32(total), nil
}

// Subscription operations
func (r *PaymentRepo) CreateSubscription(ctx context.Context, subscription *models.Subscription) error {
	return r.db.WithContext(ctx).Create(subscription).Error
}

func (r *PaymentRepo) GetUserActiveSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription
	result := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ? AND end_date > ?", userID, models.SubscriptionStatusActive, time.Now()).
		First(&subscription)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &subscription, nil
}

func (r *PaymentRepo) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*models.Subscription, error) {
	var subscription models.Subscription
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&subscription)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return &subscription, nil
}

func (r *PaymentRepo) UpdateSubscription(ctx context.Context, subscription *models.Subscription) error {
	return r.db.WithContext(ctx).Save(subscription).Error
}

func (r *PaymentRepo) DeactivateExpiredSubscriptions(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("status = ? AND end_date < ?", models.SubscriptionStatusActive, time.Now()).
		Update("status", models.SubscriptionStatusExpired).
		Error
}
