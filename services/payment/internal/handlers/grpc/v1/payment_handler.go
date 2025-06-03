package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"
	paymentpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/payment/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/services"
)

type PaymentHandler struct {
	paymentpb.UnimplementedPaymentServiceServer
	paymentService *services.PaymentService
	logger         logging.Logger
}

func NewPaymentHandler(
	paymentService *services.PaymentService,
	logger logging.Logger,
) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		logger:         logger,
	}
}

func (h *PaymentHandler) extractUserID(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		h.logger.Debug("Metadata missing from context")
		return uuid.Nil, status.Error(codes.Unauthenticated, "Missing authorization")
	}

	userIDs := md.Get("user-id")
	if len(userIDs) == 0 {
		h.logger.Debug("User ID missing from metadata")
		return uuid.Nil, status.Error(codes.Unauthenticated, "Authentication required")
	}

	userID, err := uuid.Parse(userIDs[0])
	if err != nil {
		h.logger.Error("Invalid user ID format", "error", err)
		return uuid.Nil, status.Error(codes.Unauthenticated, "Invalid user ID")
	}

	return userID, nil
}

func (h *PaymentHandler) CreatePaymentOrder(ctx context.Context, req *paymentpb.CreatePaymentOrderRequest) (*paymentpb.CreatePaymentOrderResponse, error) {
	h.logger.Info("Received create payment order request", "planID", req.PlanId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	payment, plan, err := h.paymentService.CreatePaymentOrder(ctx, userID, req.PlanId)
	if err != nil {
		h.logger.Error("Failed to create payment order", "error", err)
		switch err {
		case services.ErrInvalidPlan:
			return nil, status.Error(codes.InvalidArgument, "Invalid subscription plan")
		default:
			return nil, status.Error(codes.Internal, "Failed to create payment order")
		}
	}

	orderData := &paymentpb.PaymentOrderData{
		RazorpayOrderId: payment.RazorpayOrderID,
		RazorpayKeyId:   h.paymentService.GetRazorpayKeyID(),
		Amount:          int64(payment.Amount * 100), // Convert to paise
		Currency:        payment.Currency,
		PlanName:        plan.Name,
	}

	h.logger.Info("Payment order created successfully", "userID", userID, "planID", req.PlanId)
	return &paymentpb.CreatePaymentOrderResponse{
		Success:   true,
		Message:   "Payment order created successfully",
		OrderData: orderData,
	}, nil
}

func (h *PaymentHandler) VerifyPayment(ctx context.Context, req *paymentpb.VerifyPaymentRequest) (*paymentpb.VerifyPaymentResponse, error) {
	h.logger.Info("Received verify payment request", "orderID", req.RazorpayOrderId)

	userID, err := h.extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	subscription, err := h.paymentService.VerifyPayment(ctx, userID, req.RazorpayOrderId, req.RazorpayPaymentId, req.RazorpaySignature)
	if err != nil {
		h.logger.Error("Payment verification failed", "error", err)
		switch err {
		case services.ErrInvalidSignature:
			return nil, status.Error(codes.InvalidArgument, "Invalid payment signature")
		case services.ErrPaymentNotFound:
			return nil, status.Error(codes.NotFound, "Payment not found")
		case services.ErrDuplicatePayment:
			return nil, status.Error(codes.AlreadyExists, "Payment already processed")
		default:
			return nil, status.Error(codes.Internal, "Payment verification failed")
		}
	}

	subscriptionData := &paymentpb.SubscriptionData{
		Id:       subscription.ID.String(),
		PlanId:   subscription.PlanID,
		Status:   string(subscription.Status),
		Amount:   subscription.Amount,
		Currency: subscription.Currency,
		IsActive: subscription.IsActive(),
	}

	if subscription.StartDate != nil {
		subscriptionData.StartDate = timestamppb.New(*subscription.StartDate)
	}
	if subscription.EndDate != nil {
		subscriptionData.EndDate = timestamppb.New(*subscription.EndDate)
	}

	h.logger.Info("Payment verified successfully", "userID", userID, "orderID", req.RazorpayOrderId)
	return &paymentpb.VerifyPaymentResponse{
		Success:      true,
		Message:      "Payment verified and subscription activated",
		Subscription: subscriptionData,
	}, nil
}

func (h *PaymentHandler) GetSubscriptionStatus(ctx context.Context, req *paymentpb.GetSubscriptionStatusRequest) (*paymentpb.GetSubscriptionStatusResponse, error) {
	h.logger.Info("Received get subscription status request")

	userID, err := h.extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	subscription, err := h.paymentService.GetUserSubscription(ctx, userID)
	if err != nil {
		h.logger.Error("Failed to get subscription", "error", err)
		return nil, status.Error(codes.Internal, "Failed to get subscription status")
	}

	if subscription == nil {
		h.logger.Info("No active subscription found", "userID", userID)
		return &paymentpb.GetSubscriptionStatusResponse{
			Success:      true,
			Message:      "No active subscription found",
			Subscription: nil,
		}, nil
	}

	subscriptionData := &paymentpb.SubscriptionData{
		Id:       subscription.ID.String(),
		PlanId:   subscription.PlanID,
		Status:   string(subscription.Status),
		Amount:   subscription.Amount,
		Currency: subscription.Currency,
		IsActive: subscription.IsActive(),
	}

	if subscription.StartDate != nil {
		subscriptionData.StartDate = timestamppb.New(*subscription.StartDate)
	}
	if subscription.EndDate != nil {
		subscriptionData.EndDate = timestamppb.New(*subscription.EndDate)
	}

	h.logger.Info("Subscription status retrieved", "userID", userID)
	return &paymentpb.GetSubscriptionStatusResponse{
		Success:      true,
		Message:      "Subscription status retrieved successfully",
		Subscription: subscriptionData,
	}, nil
}

func (h *PaymentHandler) GetPaymentHistory(ctx context.Context, req *paymentpb.GetPaymentHistoryRequest) (*paymentpb.GetPaymentHistoryResponse, error) {
	h.logger.Info("Received get payment history request")

	userID, err := h.extractUserID(ctx)
	if err != nil {
		return nil, err
	}

	// Set default values if not provided
	limit := req.Limit
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	payments, total, err := h.paymentService.GetUserPaymentHistory(ctx, userID, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get payment history", "error", err)
		return nil, status.Error(codes.Internal, "Failed to get payment history")
	}

	paymentData := make([]*paymentpb.PaymentData, len(payments))
	for i, payment := range payments {
		paymentData[i] = &paymentpb.PaymentData{
			Id:              payment.ID.String(),
			RazorpayOrderId: payment.RazorpayOrderID,
			Amount:          payment.Amount,
			Currency:        payment.Currency,
			Status:          string(payment.Status),
			CreatedAt:       timestamppb.New(payment.CreatedAt),
		}

		if payment.RazorpayPaymentID != nil {
			paymentData[i].RazorpayPaymentId = *payment.RazorpayPaymentID
		}

		if payment.PaymentMethod != nil {
			paymentData[i].PaymentMethod = *payment.PaymentMethod
		}
	}

	pagination := &paymentpb.PaginationData{
		Limit:   limit,
		Offset:  offset,
		Total:   total,
		HasMore: offset+limit < total,
	}

	h.logger.Info("Payment history retrieved", "userID", userID, "count", len(payments))
	return &paymentpb.GetPaymentHistoryResponse{
		Success:    true,
		Message:    "Payment history retrieved successfully",
		Payments:   paymentData,
		Pagination: pagination,
	}, nil
}
