package v1

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	paymentpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/payment/v1"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/services"
)

type PaymentHandler struct {
	paymentpb.UnimplementedPaymentServiceServer
	paymentService *services.PaymentService
	logger         logging.Logger
}

func NewPaymentHandler(paymentService *services.PaymentService, logger logging.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
		logger:         logger,
	}
}

func (h *PaymentHandler) CreatePaymentOrder(ctx context.Context, req *paymentpb.CreatePaymentOrderRequest) (*paymentpb.CreatePaymentOrderResponse, error) {
	userID := h.getUserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "User ID required")
	}

	h.logger.Info("CreatePaymentOrder request", "userID", userID, "planID", req.PlanId)

	return h.paymentService.CreatePaymentOrder(ctx, userID, req.PlanId)
}

func (h *PaymentHandler) GetSubscriptionStatus(ctx context.Context, req *paymentpb.GetSubscriptionStatusRequest) (*paymentpb.GetSubscriptionStatusResponse, error) {
	userID := h.getUserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "User ID required")
	}

	h.logger.Info("GetSubscriptionStatus request", "userID", userID)

	return h.paymentService.GetSubscriptionStatus(ctx, userID)
}

func (h *PaymentHandler) GetPaymentHistory(ctx context.Context, req *paymentpb.GetPaymentHistoryRequest) (*paymentpb.GetPaymentHistoryResponse, error) {
	userID := h.getUserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "User ID required")
	}

	h.logger.Info("GetPaymentHistory request", "userID", userID, "limit", req.Limit, "offset", req.Offset)

	return h.paymentService.GetPaymentHistory(ctx, userID, req.Limit, req.Offset)
}

func (h *PaymentHandler) HandleWebhook(ctx context.Context, req *paymentpb.WebhookRequest) (*paymentpb.WebhookResponse, error) {
	h.logger.Info("HandleWebhook request", "event", req.Event)

	return h.paymentService.HandleWebhook(ctx, req.Event, req.Payload, req.Signature)
}

func (h *PaymentHandler) CreatePaymentURL(ctx context.Context, req *paymentpb.CreatePaymentURLRequest) (*paymentpb.CreatePaymentURLResponse, error) {
	userID := h.getUserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "User ID required")
	}

	h.logger.Info("CreatePaymentURL request", "userID", userID, "planID", req.PlanId)

	return h.paymentService.CreatePaymentURL(ctx, userID, req.PlanId)
}

func (h *PaymentHandler) getUserIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	userIDs := md.Get("user-id")
	if len(userIDs) == 0 {
		return ""
	}

	return userIDs[0]
}

func (h *PaymentHandler) VerifyPayment(ctx context.Context, req *paymentpb.VerifyPaymentRequest) (*paymentpb.VerifyPaymentResponse, error) {
	h.logger.Info("VerifyPayment gRPC request",
		"orderID", req.RazorpayOrderId,
		"paymentID", req.RazorpayPaymentId)

	// Call the existing service method (no user ID needed - extracted from payment record)
	return h.paymentService.VerifyPayment(
		ctx,
		req.RazorpayOrderId,
		req.RazorpayPaymentId,
		req.RazorpaySignature,
	)
}
