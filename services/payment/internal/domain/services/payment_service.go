package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/messaging/rabbitmq"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/payment/razorpay"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/utils/indianstandardtime"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/repositories"
	paymentErrors "github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/errors"
)

// PaymentService handles payment-related business logic
type PaymentService struct {
	paymentRepo     repositories.PaymentRepository
	razorpayService *razorpay.Service
	plansConfig     *config.PlansConfig
	logger          logging.Logger
	messageBroker   *rabbitmq.Client
}

// PaymentServiceConfig holds service configuration
type PaymentServiceConfig struct {
	PaymentRepo     repositories.PaymentRepository
	RazorpayService *razorpay.Service
	PlansConfig     *config.PlansConfig
	Logger          logging.Logger
	MessageBroker   *rabbitmq.Client
}

// NewPaymentService creates a new payment service instance
func NewPaymentService(cfg PaymentServiceConfig) *PaymentService {
	return &PaymentService{
		paymentRepo:     cfg.PaymentRepo,
		razorpayService: cfg.RazorpayService,
		plansConfig:     cfg.PlansConfig,
		logger:          cfg.Logger,
		messageBroker:   cfg.MessageBroker,
	}
}

// CreatePaymentOrder creates a new payment order for subscription
func (s *PaymentService) CreatePaymentOrder(ctx context.Context, userID uuid.UUID, planID string) (*models.Payment, *config.SubscriptionPlan, error) {
	s.logger.Info("Creating payment order", "userID", userID, "planID", planID)

	// Validate plan
	plan, err := s.validatePlan(planID)
	if err != nil {
		s.logger.Error("Invalid plan requested", "planID", planID, "userID", userID, "error", err)
		return nil, nil, err
	}

	// Create Razorpay order
	payment, err := s.createRazorpayOrder(ctx, userID, plan)
	if err != nil {
		s.logger.Error("Failed to create payment order", "error", err, "userID", userID, "planID", planID)
		return nil, nil, err
	}

	s.logger.Info("Payment order created successfully",
		"userID", userID,
		"planID", planID,
		"amount", plan.Amount,
		"orderID", payment.RazorpayOrderID)

	return payment, &plan, nil
}

// VerifyPayment verifies payment signature and creates/extends subscription
func (s *PaymentService) VerifyPayment(ctx context.Context, userID uuid.UUID, razorpayOrderID, paymentID, signature string) (*models.Subscription, error) {
	s.logger.Info("Verifying payment",
		"userID", userID,
		"orderID", razorpayOrderID,
		"paymentID", paymentID)

	// Get and validate payment
	payment, err := s.getAndValidatePayment(ctx, userID, razorpayOrderID)
	if err != nil {
		return nil, err
	}

	// Verify signature
	if err := s.verifyPaymentSignature(razorpayOrderID, paymentID, signature); err != nil {
		s.updatePaymentStatus(ctx, payment, models.PaymentStatusFailed)
		return nil, err
	}

	// Update payment status
	if err := s.updatePaymentStatus(ctx, payment, models.PaymentStatusSuccess); err != nil {
		return nil, err
	}

	// Handle subscription
	subscription, err := s.handleSubscription(ctx, userID, payment, paymentID, signature)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Payment verified and subscription processed",
		"userID", userID,
		"orderID", razorpayOrderID,
		"subscriptionID", subscription.ID)

	return subscription, nil
}

// GetUserSubscription retrieves user's active subscription
func (s *PaymentService) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	s.logger.Info("Getting user subscription", "userID", userID)

	subscription, err := s.paymentRepo.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user subscription", "error", err, "userID", userID)
		return nil, paymentErrors.NewInternalError(
			paymentErrors.CodePaymentNotFound,
			"failed to get subscription",
			err,
		)
	}

	return subscription, nil
}

// GetUserPaymentHistory retrieves user's payment history with pagination
func (s *PaymentService) GetUserPaymentHistory(ctx context.Context, userID uuid.UUID, limit, offset int32) ([]*models.Payment, int32, error) {
	s.logger.Info("Getting payment history", "userID", userID, "limit", limit, "offset", offset)

	// Validate pagination parameters
	limit, offset = s.validatePaginationParams(limit, offset)

	payments, total, err := s.paymentRepo.GetUserPayments(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get payment history", "error", err, "userID", userID)
		return nil, 0, paymentErrors.NewInternalError(
			"PAYMENT_HISTORY_FETCH_FAILED",
			"failed to get payment history",
			err,
		)
	}

	return payments, total, nil
}

// GetRazorpayKeyID returns the Razorpay key ID for frontend integration
func (s *PaymentService) GetRazorpayKeyID() string {
	return s.razorpayService.GetKeyID()
}

// GetActivePlans returns all active subscription plans
func (s *PaymentService) GetActivePlans() map[string]config.SubscriptionPlan {
	return s.plansConfig.GetActivePlans()
}

// Private helper methods
func (s *PaymentService) validatePlan(planID string) (config.SubscriptionPlan, error) {
	// Debug: Log all available plans
	activePlans := s.plansConfig.GetActivePlans()
	s.logger.Info("Validating plan",
		"requested_plan", planID,
		"available_plans_count", len(activePlans))

	for id, plan := range activePlans {
		s.logger.Info("Available plan",
			"id", id,
			"name", plan.Name,
			"active", plan.IsActive)
	}

	plan, exists := s.plansConfig.GetPlan(planID)
	if !exists {
		s.logger.Error("Plan not found",
			"requested_plan", planID,
			"available_plans", func() []string {
				var plans []string
				for id := range activePlans {
					plans = append(plans, id)
				}
				return plans
			}())
		return config.SubscriptionPlan{}, paymentErrors.NewValidationError(
			paymentErrors.CodeInvalidPlan,
			constants.ErrMsgInvalidPlan,
			nil,
		)
	}

	if !plan.IsActive {
		s.logger.Error("Plan is not active",
			"plan_id", planID,
			"active", plan.IsActive)
		return config.SubscriptionPlan{}, paymentErrors.NewValidationError(
			paymentErrors.CodeInvalidPlan,
			constants.ErrMsgInvalidPlan,
			nil,
		)
	}

	s.logger.Info("Plan validation successful",
		"plan_id", planID,
		"plan_name", plan.Name)
	return plan, nil
}

func (s *PaymentService) createRazorpayOrder(ctx context.Context, userID uuid.UUID, plan config.SubscriptionPlan) (*models.Payment, error) {
	// Create Razorpay order
	amountInPaise := int64(plan.Amount * constants.PaiseMultiplier)
	razorpayOrder, err := s.razorpayService.CreateOrder(amountInPaise, plan.Currency)
	if err != nil {
		return nil, paymentErrors.NewExternalError(
			paymentErrors.CodeOrderCreation,
			constants.ErrMsgOrderCreationFailed,
			err,
		)
	}

	// Create payment record
	now := indianstandardtime.Now()
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
		return nil, paymentErrors.NewInternalError(
			paymentErrors.CodePaymentSave,
			constants.ErrMsgPaymentSaveFailed,
			err,
		)
	}

	return payment, nil
}

func (s *PaymentService) getAndValidatePayment(ctx context.Context, userID uuid.UUID, razorpayOrderID string) (*models.Payment, error) {
	payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, razorpayOrderID)
	if err != nil {
		return nil, paymentErrors.NewInternalError(
			paymentErrors.CodePaymentNotFound,
			"failed to get payment",
			err,
		)
	}

	if payment == nil {
		return nil, paymentErrors.NewNotFoundError(
			paymentErrors.CodePaymentNotFound,
			constants.ErrMsgPaymentNotFound,
			nil,
		)
	}

	// Verify user ownership
	if payment.UserID != userID {
		return nil, paymentErrors.NewValidationError(
			paymentErrors.CodeUnauthorized,
			"unauthorized access to payment",
			nil,
		)
	}

	// Check if payment already processed
	if payment.Status == models.PaymentStatusSuccess {
		return nil, paymentErrors.NewConflictError(
			paymentErrors.CodeDuplicatePayment,
			constants.ErrMsgDuplicatePayment,
			nil,
		)
	}

	return payment, nil
}

func (s *PaymentService) verifyPaymentSignature(razorpayOrderID, paymentID, signature string) error {
	attributes := map[string]interface{}{
		"razorpay_order_id":   razorpayOrderID,
		"razorpay_payment_id": paymentID,
		"razorpay_signature":  signature,
	}

	if err := s.razorpayService.VerifyPaymentSignature(attributes); err != nil {
		return paymentErrors.NewValidationError(
			paymentErrors.CodeInvalidSignature,
			constants.ErrMsgInvalidSignature,
			err,
		)
	}

	return nil
}

func (s *PaymentService) updatePaymentStatus(ctx context.Context, payment *models.Payment, status models.PaymentStatus) error {
	payment.Status = status
	payment.UpdatedAt = indianstandardtime.Now()

	if err := s.paymentRepo.UpdatePayment(ctx, payment); err != nil {
		s.logger.Error("Failed to update payment status", "error", err, "paymentID", payment.ID, "status", status)
		return paymentErrors.NewInternalError(
			"PAYMENT_UPDATE_FAILED",
			"failed to update payment",
			err,
		)
	}

	return nil
}

func (s *PaymentService) handleSubscription(ctx context.Context, userID uuid.UUID, payment *models.Payment, paymentID, signature string) (*models.Subscription, error) {
	// Update payment with signature details
	payment.RazorpayPaymentID = &paymentID
	payment.RazorpaySignature = &signature
	if err := s.paymentRepo.UpdatePayment(ctx, payment); err != nil {
		return nil, paymentErrors.NewInternalError(
			"PAYMENT_UPDATE_FAILED",
			"failed to update payment with signature",
			err,
		)
	}

	// Check for existing subscription
	existingSubscription, err := s.paymentRepo.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		return nil, paymentErrors.NewInternalError(
			"SUBSCRIPTION_CHECK_FAILED",
			"failed to check existing subscription",
			err,
		)
	}

	if existingSubscription != nil {
		return s.extendSubscription(ctx, existingSubscription)
	}

	return s.createNewSubscription(ctx, userID, payment)
}

func (s *PaymentService) extendSubscription(ctx context.Context, subscription *models.Subscription) (*models.Subscription, error) {
	s.logger.Info("Extending existing subscription", "subscriptionID", subscription.ID)

	now := indianstandardtime.Now()
	var newEndDate time.Time

	if subscription.EndDate != nil {
		newEndDate = subscription.EndDate.AddDate(constants.DefaultPlanDurationYears, 0, 0)
	} else {
		newEndDate = now.AddDate(constants.DefaultPlanDurationYears, 0, 0)
	}

	subscription.EndDate = &newEndDate
	subscription.UpdatedAt = now

	if err := s.paymentRepo.UpdateSubscription(ctx, subscription); err != nil {
		return nil, paymentErrors.NewInternalError(
			"SUBSCRIPTION_EXTEND_FAILED",
			"failed to extend subscription",
			err,
		)
	}

	// Publish subscription extension event
	if s.messageBroker != nil {
		subscriptionEvent := map[string]interface{}{
			"user_id":         subscription.UserID.String(),
			"premium_until":   newEndDate,
			"event_type":      "subscription.extended",
			"subscription_id": subscription.ID.String(),
			"plan_id":         subscription.PlanID,
			"timestamp":       now,
		}

		if err := s.messageBroker.Publish("subscription.extended", subscriptionEvent); err != nil {
			s.logger.Error("Failed to publish subscription extension event", "userID", subscription.UserID, "error", err)
			// Don't return error as subscription is already updated
		} else {
			s.logger.Info("Subscription extension event published", "userID", subscription.UserID)
		}
	}

	return subscription, nil
}

func (s *PaymentService) createNewSubscription(ctx context.Context, userID uuid.UUID, payment *models.Payment) (*models.Subscription, error) {
	s.logger.Info("Creating new subscription", "userID", userID, "paymentID", payment.ID)

	now := indianstandardtime.Now()
	endDate := now.AddDate(constants.DefaultPlanDurationYears, 0, 0)

	subscription := &models.Subscription{
		UserID:    userID,
		PlanID:    constants.DefaultPlanID,
		Status:    models.SubscriptionStatusActive,
		Amount:    payment.Amount,
		Currency:  payment.Currency,
		PaymentID: &payment.ID,
		StartDate: &now,
		EndDate:   &endDate,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.paymentRepo.CreateSubscription(ctx, subscription); err != nil {
		return nil, paymentErrors.NewInternalError(
			"SUBSCRIPTION_CREATE_FAILED",
			"failed to create subscription",
			err,
		)
	}

	// Publish subscription activation event
	if s.messageBroker != nil {
		subscriptionEvent := map[string]interface{}{
			"user_id":         userID.String(),
			"premium_until":   endDate,
			"event_type":      "subscription.activated",
			"subscription_id": subscription.ID.String(),
			"plan_id":         subscription.PlanID,
			"amount":          subscription.Amount,
			"timestamp":       now,
		}

		if err := s.messageBroker.Publish("subscription.activated", subscriptionEvent); err != nil {
			s.logger.Error("Failed to publish subscription activation event", "userID", userID, "error", err)
			// Don't return error as subscription is already created
		} else {
			s.logger.Info("Subscription activation event published", "userID", userID)
		}
	}

	return subscription, nil
}

func (s *PaymentService) validatePaginationParams(limit, offset int32) (int32, int32) {
	if limit <= 0 || limit > constants.MaxLimit {
		limit = constants.DefaultLimit
	}
	if offset < 0 {
		offset = constants.DefaultOffset
	}
	return limit, offset
}

// Add to existing PaymentService
func (s *PaymentService) VerifyWebhookSignature(signature, payload string) error {
	// Implement Razorpay webhook signature verification
	return s.razorpayService.VerifyWebhookSignature(signature, payload)
}

func (s *PaymentService) HandleWebhookEvent(ctx context.Context, event, payload string) error {
	switch event {
	case "payment.captured":
		return s.handlePaymentCaptured(ctx, payload)
	case "payment.failed":
		return s.handlePaymentFailed(ctx, payload)
	default:
		s.logger.Info("Unhandled webhook event", "event", event)
		return nil
	}
}

func (s *PaymentService) CreatePaymentURL(ctx context.Context, userID uuid.UUID, planID string) (string, error) {
	// Create payment order and return a simple redirect URL
	payment, plan, err := s.CreatePaymentOrder(ctx, userID, planID)
	if err != nil {
		return "", err
	}

	// Return a simple URL that redirects to Razorpay
	return fmt.Sprintf("/api/v1/payment/checkout?order_id=%s", payment.RazorpayOrderID), nil
}
