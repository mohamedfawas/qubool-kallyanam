package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/payment/razorpay"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/repositories"
)

var (
	ErrInvalidPlan      = errors.New("invalid subscription plan")
	ErrPaymentNotFound  = errors.New("payment not found")
	ErrInvalidSignature = errors.New("invalid payment signature")
	ErrDuplicatePayment = errors.New("payment already processed")
)

type PaymentService struct {
	paymentRepo     repositories.PaymentRepository
	razorpayService *razorpay.Service
	logger          logging.Logger
}

func NewPaymentService(
	paymentRepo repositories.PaymentRepository,
	razorpayService *razorpay.Service,
	logger logging.Logger,
) *PaymentService {
	return &PaymentService{
		paymentRepo:     paymentRepo,
		razorpayService: razorpayService,
		logger:          logger,
	}
}

func (s *PaymentService) CreatePaymentOrder(ctx context.Context, userID uuid.UUID, planID string) (*models.Payment, *models.SubscriptionPlan, error) {
	s.logger.Info("Creating payment order", "userID", userID, "planID", planID)

	// Validate plan
	plan, exists := models.DefaultPlans[planID]
	if !exists {
		s.logger.Error("Invalid plan requested", "planID", planID, "userID", userID)
		return nil, nil, ErrInvalidPlan
	}

	// Create Razorpay order
	amountInPaise := int64(plan.Amount * 100) // Convert to paise
	razorpayOrder, err := s.razorpayService.CreateOrder(amountInPaise, plan.Currency)
	if err != nil {
		s.logger.Error("Failed to create Razorpay order", "error", err)
		return nil, nil, fmt.Errorf("failed to create payment order: %w", err)
	}

	now := indianstandardtime.Now()
	// Create payment record
	payment := &models.Payment{
		UserID:          userID,
		RazorpayOrderID: razorpayOrder.ID,
		Amount:          plan.Amount,
		Currency:        plan.Currency,
		Status:          models.PaymentStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Save to database
	if err := s.paymentRepo.CreatePayment(ctx, payment); err != nil {
		s.logger.Error("Failed to save payment record", "error", err)
		return nil, nil, fmt.Errorf("failed to save payment record: %w", err)
	}

	s.logger.Info("Payment order created successfully",
		"userID", userID,
		"planID", planID,
		"amount", plan.Amount,
		"orderID", payment.RazorpayOrderID)

	return payment, &plan, nil
}

func (s *PaymentService) VerifyPayment(ctx context.Context, userID uuid.UUID, razorpayOrderID, paymentID, signature string) (*models.Subscription, error) {
	s.logger.Info("Verifying payment",
		"userID", userID,
		"orderID", razorpayOrderID,
		"paymentID", paymentID)

	// Get payment record
	payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, razorpayOrderID)
	if err != nil {
		s.logger.Error("Failed to get payment", "error", err)
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	if payment == nil {
		s.logger.Error("Payment not found", "orderID", razorpayOrderID)
		return nil, ErrPaymentNotFound
	}

	// Verify user ownership
	if payment.UserID != userID {
		s.logger.Error("Payment user mismatch", "paymentUserID", payment.UserID, "requestUserID", userID)
		return nil, ErrPaymentNotFound
	}

	// Check if payment already processed
	if payment.Status == models.PaymentStatusSuccess {
		s.logger.Info("Payment already processed", "orderID", razorpayOrderID)
		return nil, ErrDuplicatePayment
	}

	// Verify signature
	attributes := map[string]interface{}{
		"razorpay_order_id":   razorpayOrderID,
		"razorpay_payment_id": paymentID,
		"razorpay_signature":  signature,
	}

	if err := s.razorpayService.VerifyPaymentSignature(attributes); err != nil {
		s.logger.Error("Payment signature verification failed", "error", err)

		// Update payment status to failed
		payment.Status = models.PaymentStatusFailed
		payment.UpdatedAt = indianstandardtime.Now()
		if updateErr := s.paymentRepo.UpdatePayment(ctx, payment); updateErr != nil {
			s.logger.Error("Failed to update payment status to failed", "error", updateErr)
		}

		return nil, ErrInvalidSignature
	}

	// Update payment record
	payment.RazorpayPaymentID = &paymentID
	payment.RazorpaySignature = &signature
	payment.Status = models.PaymentStatusSuccess
	payment.UpdatedAt = indianstandardtime.Now()

	if err := s.paymentRepo.UpdatePayment(ctx, payment); err != nil {
		s.logger.Error("Failed to update payment", "error", err)
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	// Check if user already has an active subscription
	existingSubscription, err := s.paymentRepo.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to check existing subscription", "error", err)
		return nil, fmt.Errorf("failed to check existing subscription: %w", err)
	}

	if existingSubscription != nil {
		s.logger.Info("User already has active subscription, extending it", "userID", userID, "existingSubID", existingSubscription.ID)
		// Extend existing subscription
		if existingSubscription.EndDate != nil {
			newEndDate := existingSubscription.EndDate.AddDate(1, 0, 0) // Extend by 1 year
			existingSubscription.EndDate = &newEndDate
		} else {
			now := indianstandardtime.Now()
			endDate := now.AddDate(1, 0, 0) // Extend by 1 year
			existingSubscription.EndDate = &endDate
		}
		existingSubscription.UpdatedAt = indianstandardtime.Now()

		if err := s.paymentRepo.UpdateSubscription(ctx, existingSubscription); err != nil {
			s.logger.Error("Failed to extend subscription", "error", err)
			return nil, fmt.Errorf("failed to extend subscription: %w", err)
		}

		return existingSubscription, nil
	}

	now := indianstandardtime.Now()
	// Create new subscription
	subscription := &models.Subscription{
		UserID:    userID,
		PlanID:    "premium_365", // Default plan
		Status:    models.SubscriptionStatusActive,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		PaymentID: &payment.ID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Set subscription dates
	subscription.StartDate = &now
	endDate := now.AddDate(1, 0, 0) // 1 year from now
	subscription.EndDate = &endDate

	if err := s.paymentRepo.CreateSubscription(ctx, subscription); err != nil {
		s.logger.Error("Failed to create subscription", "error", err)
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	s.logger.Info("Payment verified and subscription created",
		"userID", userID,
		"orderID", razorpayOrderID,
		"subscriptionID", subscription.ID)

	return subscription, nil
}

func (s *PaymentService) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	s.logger.Info("Getting user subscription", "userID", userID)

	subscription, err := s.paymentRepo.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user subscription", "error", err, "userID", userID)
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return subscription, nil
}

func (s *PaymentService) GetUserPaymentHistory(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*models.Payment, int32, error) {
	s.logger.Info("Getting payment history", "userID", userID, "limit", limit, "offset", offset)

	payments, total, err := s.paymentRepo.GetUserPayments(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get payment history", "error", err, "userID", userID)
		return nil, 0, fmt.Errorf("failed to get payment history: %w", err)
	}

	return payments, total, nil
}

func (s *PaymentService) GetRazorpayKeyID() string {
	return s.razorpayService.GetKeyID()
}
