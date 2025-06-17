package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/payment/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/messaging/rabbitmq"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/payment/razorpay"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/constants"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/models"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/repositories"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/errors"
)

type PaymentDetails struct {
	OrderID   string
	PaymentID string
	Amount    int64
	Currency  string
	PlanName  string
	Status    string
	UserID    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type PaymentService struct {
	paymentRepo     repositories.PaymentRepository
	razorpayService *razorpay.Service
	plansConfig     *config.PlansConfig
	logger          logging.Logger
	messageBroker   *rabbitmq.Client
}

type PaymentServiceConfig struct {
	PaymentRepo     repositories.PaymentRepository
	RazorpayService *razorpay.Service
	PlansConfig     *config.PlansConfig
	Logger          logging.Logger
	MessageBroker   *rabbitmq.Client
}

type OrderDetails struct {
	RazorpayOrderId string
	RazorpayKeyId   string
	Amount          int64
	Currency        string
	PlanName        string
	Status          string
	UserID          string
}

func NewPaymentService(cfg PaymentServiceConfig) *PaymentService {
	return &PaymentService{
		paymentRepo:     cfg.PaymentRepo,
		razorpayService: cfg.RazorpayService,
		plansConfig:     cfg.PlansConfig,
		logger:          cfg.Logger,
		messageBroker:   cfg.MessageBroker,
	}
}

func (s *PaymentService) CreatePaymentOrder(ctx context.Context, userID, planID string) (*paymentpb.CreatePaymentOrderResponse, error) {
	s.logger.Info("Creating payment order", "userID", userID, "planID", planID)

	// Get plan configuration
	plan, exists := s.plansConfig.GetPlan(planID)
	if !exists || !plan.IsActive {
		return &paymentpb.CreatePaymentOrderResponse{
			Success: false,
			Message: "Invalid plan selected",
			Error:   "INVALID_PLAN",
		}, nil
	}

	// Convert amount to paise
	amountInPaise := int64(plan.Amount * constants.PaiseMultiplier)

	// Create Razorpay order
	order, err := s.razorpayService.CreateOrder(amountInPaise, plan.Currency)
	if err != nil {
		s.logger.Error("Failed to create Razorpay order", "error", err)
		return &paymentpb.CreatePaymentOrderResponse{
			Success: false,
			Message: "Failed to create payment order",
			Error:   "ORDER_CREATION_FAILED",
		}, nil
	}

	// Parse user ID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return &paymentpb.CreatePaymentOrderResponse{
			Success: false,
			Message: "Invalid user ID",
			Error:   "INVALID_USER_ID",
		}, nil
	}

	// Create payment record
	payment := &models.Payment{
		UserID:          userUUID,
		RazorpayOrderID: order.ID,
		Amount:          plan.Amount,
		Currency:        plan.Currency,
		Status:          models.PaymentStatusPending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.paymentRepo.CreatePayment(ctx, payment); err != nil {
		s.logger.Error("Failed to save payment record", "error", err)
		return &paymentpb.CreatePaymentOrderResponse{
			Success: false,
			Message: "Failed to save payment record",
			Error:   "PAYMENT_SAVE_FAILED",
		}, nil
	}

	return &paymentpb.CreatePaymentOrderResponse{
		Success: true,
		Message: "Payment order created successfully",
		OrderData: &paymentpb.PaymentOrderData{
			RazorpayOrderId: order.ID,
			RazorpayKeyId:   s.razorpayService.GetKeyID(),
			Amount:          amountInPaise,
			Currency:        plan.Currency,
			PlanName:        plan.Name,
		},
	}, nil
}

func (s *PaymentService) VerifyPayment(ctx context.Context, userID, razorpayOrderID, razorpayPaymentID, razorpaySignature string) (*paymentpb.VerifyPaymentResponse, error) {
	s.logger.Info("Verifying payment", "userID", userID, "orderID", razorpayOrderID)

	// Verify signature with Razorpay
	attributes := map[string]interface{}{
		"razorpay_order_id":   razorpayOrderID,
		"razorpay_payment_id": razorpayPaymentID,
		"razorpay_signature":  razorpaySignature,
	}

	if err := s.razorpayService.VerifyPaymentSignature(attributes); err != nil {
		s.logger.Error("Payment signature verification failed", "error", err)
		return &paymentpb.VerifyPaymentResponse{
			Success: false,
			Message: "Payment verification failed",
			Error:   "SIGNATURE_VERIFICATION_FAILED",
		}, nil
	}

	// Get payment record
	payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, razorpayOrderID)
	if err != nil {
		s.logger.Error("Failed to get payment record", "error", err)
		return &paymentpb.VerifyPaymentResponse{
			Success: false,
			Message: "Payment not found",
			Error:   "PAYMENT_NOT_FOUND",
		}, nil
	}

	// Check if payment is already processed
	if payment.Status == models.PaymentStatusSuccess {
		s.logger.Info("Payment already processed", "orderID", razorpayOrderID)
		return &paymentpb.VerifyPaymentResponse{
			Success: false,
			Message: "Payment already processed",
			Error:   "DUPLICATE_PAYMENT",
		}, nil
	}

	// Verify user ownership
	userUUID, _ := uuid.Parse(userID)
	if payment.UserID != userUUID {
		s.logger.Error("Unauthorized payment access", "userID", userID, "paymentUserID", payment.UserID)
		return &paymentpb.VerifyPaymentResponse{
			Success: false,
			Message: "Unauthorized access",
			Error:   "UNAUTHORIZED_ACCESS",
		}, nil
	}

	// Process payment and create subscription in transaction
	var subscription *models.Subscription
	err = s.paymentRepo.WithTx(ctx, func(txCtx context.Context) error {
		// Update payment record
		paymentIDStr := razorpayPaymentID
		signatureStr := razorpaySignature
		payment.RazorpayPaymentID = &paymentIDStr
		payment.RazorpaySignature = &signatureStr
		payment.Status = models.PaymentStatusSuccess
		payment.UpdatedAt = time.Now()

		if err := s.paymentRepo.UpdatePayment(txCtx, payment); err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}

		// Get plan details
		planID := constants.DefaultPlanID // or extract from payment record if stored
		plan, exists := s.plansConfig.GetPlan(planID)
		if !exists {
			return fmt.Errorf("plan not found: %s", planID)
		}

		// Create subscription
		now := time.Now()
		endDate := now.AddDate(0, 0, plan.DurationDays)

		subscription = &models.Subscription{
			UserID:    userUUID,
			PlanID:    plan.ID,
			Status:    models.SubscriptionStatusActive,
			StartDate: &now,
			EndDate:   &endDate,
			Amount:    plan.Amount,
			Currency:  plan.Currency,
			PaymentID: &payment.ID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		return s.paymentRepo.CreateSubscription(txCtx, subscription)
	})

	if err != nil {
		s.logger.Error("Failed to process payment", "error", err)
		return &paymentpb.VerifyPaymentResponse{
			Success: false,
			Message: "Failed to process payment",
			Error:   "PAYMENT_PROCESSING_FAILED",
		}, nil
	}

	// Publish subscription activated event
	s.publishSubscriptionEvent(userID, "subscription.activated", subscription)

	// Convert subscription to protobuf
	subscriptionData := &paymentpb.SubscriptionData{
		Id:        subscription.ID.String(),
		PlanId:    subscription.PlanID,
		Status:    string(subscription.Status),
		StartDate: timestamppb.New(*subscription.StartDate),
		EndDate:   timestamppb.New(*subscription.EndDate),
		Amount:    subscription.Amount,
		Currency:  subscription.Currency,
		IsActive:  subscription.IsActive(),
	}

	return &paymentpb.VerifyPaymentResponse{
		Success:      true,
		Message:      "Payment verified and subscription activated",
		Subscription: subscriptionData,
	}, nil
}

func (s *PaymentService) GetSubscriptionStatus(ctx context.Context, userID string) (*paymentpb.GetSubscriptionStatusResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return &paymentpb.GetSubscriptionStatusResponse{
			Success: false,
			Message: "Invalid user ID",
			Error:   "INVALID_USER_ID",
		}, nil
	}

	subscription, err := s.paymentRepo.GetActiveSubscriptionByUserID(ctx, userUUID)
	if err != nil {
		if err == errors.ErrSubscriptionNotFound {
			return &paymentpb.GetSubscriptionStatusResponse{
				Success: true,
				Message: "No active subscription found",
			}, nil
		}
		s.logger.Error("Failed to get subscription", "error", err)
		return &paymentpb.GetSubscriptionStatusResponse{
			Success: false,
			Message: "Failed to get subscription",
			Error:   "SUBSCRIPTION_FETCH_FAILED",
		}, nil
	}

	subscriptionData := &paymentpb.SubscriptionData{
		Id:        subscription.ID.String(),
		PlanId:    subscription.PlanID,
		Status:    string(subscription.Status),
		StartDate: timestamppb.New(*subscription.StartDate),
		EndDate:   timestamppb.New(*subscription.EndDate),
		Amount:    subscription.Amount,
		Currency:  subscription.Currency,
		IsActive:  subscription.IsActive(),
	}

	return &paymentpb.GetSubscriptionStatusResponse{
		Success:      true,
		Message:      "Subscription found",
		Subscription: subscriptionData,
	}, nil
}

func (s *PaymentService) GetPaymentHistory(ctx context.Context, userID string, limit, offset int32) (*paymentpb.GetPaymentHistoryResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return &paymentpb.GetPaymentHistoryResponse{
			Success: false,
			Message: "Invalid user ID",
			Error:   "INVALID_USER_ID",
		}, nil
	}

	// Apply default limits
	if limit <= 0 || limit > constants.MaxLimit {
		limit = constants.DefaultLimit
	}
	if offset < 0 {
		offset = constants.DefaultOffset
	}

	payments, total, err := s.paymentRepo.GetPaymentsByUserID(ctx, userUUID, int(limit), int(offset))
	if err != nil {
		s.logger.Error("Failed to get payment history", "error", err)
		return &paymentpb.GetPaymentHistoryResponse{
			Success: false,
			Message: "Failed to get payment history",
			Error:   "PAYMENT_HISTORY_FETCH_FAILED",
		}, nil
	}

	// Convert to protobuf
	paymentData := make([]*paymentpb.PaymentData, len(payments))
	for i, payment := range payments {
		paymentData[i] = &paymentpb.PaymentData{
			Id:                payment.ID.String(),
			RazorpayOrderId:   payment.RazorpayOrderID,
			RazorpayPaymentId: getStringPtr(payment.RazorpayPaymentID),
			Amount:            payment.Amount,
			Currency:          payment.Currency,
			Status:            string(payment.Status),
			PaymentMethod:     getStringPtr(payment.PaymentMethod),
			CreatedAt:         timestamppb.New(payment.CreatedAt),
		}
	}

	pagination := &paymentpb.PaginationData{
		Limit:   limit,
		Offset:  offset,
		Total:   int32(total),
		HasMore: int64(offset+limit) < total,
	}

	return &paymentpb.GetPaymentHistoryResponse{
		Success:    true,
		Message:    "Payment history retrieved",
		Payments:   paymentData,
		Pagination: pagination,
	}, nil
}

func (s *PaymentService) HandleWebhook(ctx context.Context, event, payload, signature string) (*paymentpb.WebhookResponse, error) {
	s.logger.Info("Handling webhook", "event", event)

	// Verify webhook signature
	if err := s.razorpayService.VerifyWebhookSignature(signature, payload); err != nil {
		s.logger.Error("Webhook signature verification failed", "error", err)
		return &paymentpb.WebhookResponse{
			Success: false,
			Message: "Signature verification failed",
		}, nil
	}

	// Process webhook event
	// This is a basic implementation - you can extend based on your needs
	s.logger.Info("Webhook processed successfully", "event", event)

	return &paymentpb.WebhookResponse{
		Success: true,
		Message: "Webhook processed successfully",
	}, nil
}

func (s *PaymentService) CreatePaymentURL(ctx context.Context, userID, planID string) (*paymentpb.CreatePaymentURLResponse, error) {
	// Get plan configuration
	plan, exists := s.plansConfig.GetPlan(planID)
	if !exists || !plan.IsActive {
		return &paymentpb.CreatePaymentURLResponse{
			Success: false,
			Message: "Invalid plan selected",
		}, nil
	}

	// Create a simple payment URL (redirect to payment service)
	paymentURL := fmt.Sprintf("http://localhost:8081/payment/checkout?plan_id=%s", planID)

	return &paymentpb.CreatePaymentURLResponse{
		Success:    true,
		Message:    "Payment URL created",
		PaymentUrl: paymentURL,
	}, nil
}

// Helper functions
func getStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (s *PaymentService) publishSubscriptionEvent(userID, eventType string, subscription *models.Subscription) {
	if s.messageBroker == nil {
		return
	}

	event := map[string]interface{}{
		"user_id":         userID,
		"subscription_id": subscription.ID.String(),
		"plan_id":         subscription.PlanID,
		"status":          string(subscription.Status),
		"start_date":      subscription.StartDate,
		"end_date":        subscription.EndDate,
		"premium_until":   subscription.EndDate, // auth service will use this to check if user is premium
		"amount":          subscription.Amount,
		"currency":        subscription.Currency,
		"timestamp":       time.Now(),
	}

	if err := s.messageBroker.Publish(eventType, event); err != nil {
		s.logger.Error("Failed to publish subscription event", "error", err, "event", eventType)
	}
}

func (s *PaymentService) GetOrderDetails(ctx context.Context, razorpayOrderID string) (*OrderDetails, error) {
	s.logger.Info("Getting order details", "orderID", razorpayOrderID)

	// Get payment record by order ID
	payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, razorpayOrderID)
	if err != nil {
		s.logger.Error("Failed to get payment record", "error", err)
		return nil, fmt.Errorf("order not found")
	}

	// Get plan details
	plan, exists := s.plansConfig.GetPlan("premium_365") // You might want to store plan_id in payment record
	if !exists {
		return nil, fmt.Errorf("plan not found")
	}

	return &OrderDetails{
		RazorpayOrderId: payment.RazorpayOrderID,
		RazorpayKeyId:   s.razorpayService.GetKeyID(),
		Amount:          int64(payment.Amount * 100), // Convert to paise
		Currency:        payment.Currency,
		PlanName:        plan.Name,
		Status:          string(payment.Status),
		UserID:          payment.UserID.String(),
	}, nil
}

func (s *PaymentService) GetPaymentDetailsByOrderID(ctx context.Context, razorpayOrderID string) (*PaymentDetails, error) {
	s.logger.Info("Getting payment details by order ID", "orderID", razorpayOrderID)

	// Get payment record by order ID
	payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, razorpayOrderID)
	if err != nil {
		s.logger.Error("Failed to get payment record", "error", err)
		return nil, fmt.Errorf("payment not found")
	}

	// Get plan details
	plan, exists := s.plansConfig.GetPlan("premium_365") // You might want to store plan_id in payment record
	if !exists {
		return nil, fmt.Errorf("plan not found")
	}

	return &PaymentDetails{
		OrderID:   payment.RazorpayOrderID,
		PaymentID: *payment.RazorpayPaymentID,  // Will be null if not paid yet
		Amount:    int64(payment.Amount * 100), // Convert to paise
		Currency:  payment.Currency,
		PlanName:  plan.Name,
		Status:    string(payment.Status),
		UserID:    payment.UserID.String(),
		CreatedAt: payment.CreatedAt,
		UpdatedAt: payment.UpdatedAt,
	}, nil
}

// VerifyPaymentByOrder verifies payment without requiring user ID - gets user context from payment record
func (s *PaymentService) VerifyPaymentByOrder(ctx context.Context, razorpayOrderID, razorpayPaymentID, razorpaySignature string) (*paymentpb.VerifyPaymentResponse, error) {
	s.logger.Info("Verifying payment by order", "orderID", razorpayOrderID)

	// ✅ Step 1: Verify signature with Razorpay (security check)
	attributes := map[string]interface{}{
		"razorpay_order_id":   razorpayOrderID,
		"razorpay_payment_id": razorpayPaymentID,
		"razorpay_signature":  razorpaySignature,
	}

	if err := s.razorpayService.VerifyPaymentSignature(attributes); err != nil {
		s.logger.Error("Payment signature verification failed", "error", err)
		return &paymentpb.VerifyPaymentResponse{
			Success: false,
			Message: "Payment verification failed",
			Error:   "SIGNATURE_VERIFICATION_FAILED",
		}, nil
	}

	// ✅ Step 2: Get payment record by order ID (contains user info)
	payment, err := s.paymentRepo.GetPaymentByOrderID(ctx, razorpayOrderID)
	if err != nil {
		s.logger.Error("Failed to get payment record", "error", err)
		return &paymentpb.VerifyPaymentResponse{
			Success: false,
			Message: "Payment not found",
			Error:   "PAYMENT_NOT_FOUND",
		}, nil
	}

	// ✅ Step 3: Check if payment is already processed
	if payment.Status == models.PaymentStatusSuccess {
		s.logger.Info("Payment already processed", "orderID", razorpayOrderID)
		return &paymentpb.VerifyPaymentResponse{
			Success: false,
			Message: "Payment already processed",
			Error:   "DUPLICATE_PAYMENT",
		}, nil
	}

	// ✅ Step 4: Extract user ID from payment record (no external user auth needed)
	userUUID := payment.UserID
	userID := userUUID.String()

	s.logger.Info("Processing payment for user", "userID", userID, "orderID", razorpayOrderID)

	// ✅ Step 5: Process payment and create subscription in transaction
	var subscription *models.Subscription
	err = s.paymentRepo.WithTx(ctx, func(txCtx context.Context) error {
		// Update payment record
		paymentIDStr := razorpayPaymentID
		signatureStr := razorpaySignature
		payment.RazorpayPaymentID = &paymentIDStr
		payment.RazorpaySignature = &signatureStr
		payment.Status = models.PaymentStatusSuccess
		payment.UpdatedAt = time.Now()

		if err := s.paymentRepo.UpdatePayment(txCtx, payment); err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}

		// Get plan details
		planID := constants.DefaultPlanID // or extract from payment record if stored
		plan, exists := s.plansConfig.GetPlan(planID)
		if !exists {
			return fmt.Errorf("plan not found: %s", planID)
		}

		// Create subscription
		now := time.Now()
		endDate := now.AddDate(0, 0, plan.DurationDays)

		subscription = &models.Subscription{
			UserID:    userUUID,
			PlanID:    plan.ID,
			Status:    models.SubscriptionStatusActive,
			StartDate: &now,
			EndDate:   &endDate,
			Amount:    plan.Amount,
			Currency:  plan.Currency,
			PaymentID: &payment.ID,
			CreatedAt: now,
			UpdatedAt: now,
		}

		return s.paymentRepo.CreateSubscription(txCtx, subscription)
	})

	if err != nil {
		s.logger.Error("Failed to process payment", "error", err)
		return &paymentpb.VerifyPaymentResponse{
			Success: false,
			Message: "Failed to process payment",
			Error:   "PAYMENT_PROCESSING_FAILED",
		}, nil
	}

	// ✅ Step 6: Publish subscription activated event
	s.publishSubscriptionEvent(userID, "subscription.activated", subscription)

	// ✅ Step 7: Convert subscription to protobuf response
	subscriptionData := &paymentpb.SubscriptionData{
		Id:        subscription.ID.String(),
		PlanId:    subscription.PlanID,
		Status:    string(subscription.Status),
		StartDate: timestamppb.New(*subscription.StartDate),
		EndDate:   timestamppb.New(*subscription.EndDate),
		Amount:    subscription.Amount,
		Currency:  subscription.Currency,
		IsActive:  subscription.IsActive(),
	}

	s.logger.Info("Payment verified and subscription activated", "userID", userID, "subscriptionID", subscription.ID)

	return &paymentpb.VerifyPaymentResponse{
		Success:      true,
		Message:      "Payment verified and subscription activated",
		Subscription: subscriptionData,
	}, nil
}
