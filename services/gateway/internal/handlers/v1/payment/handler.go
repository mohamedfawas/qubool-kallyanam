package payment

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/clients/payment"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

type Handler struct {
	paymentClient *payment.Client
	logger        logging.Logger
}

func NewHandler(paymentClient *payment.Client, logger logging.Logger) *Handler {
	return &Handler{
		paymentClient: paymentClient,
		logger:        logger,
	}
}

type CreateOrderRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

type VerifyPaymentRequest struct {
	RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature" binding:"required"`
}

// CreateOrder - Creates a payment order via payment service
func (h *Handler) CreateOrder(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	success, message, orderData, err := h.paymentClient.CreatePaymentOrder(c.Request.Context(), req.PlanID)
	if err != nil {
		h.logger.Error("Failed to create payment order", "error", err, "userID", userID)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if !success {
		pkghttp.Error(c, pkghttp.NewBadRequest(message, nil))
		return
	}

	response := gin.H{
		"success": success,
		"message": message,
		"order_data": gin.H{
			"razorpay_order_id": orderData.RazorpayOrderId,
			"razorpay_key_id":   orderData.RazorpayKeyId,
			"amount":            orderData.Amount,
			"currency":          orderData.Currency,
			"plan_name":         orderData.PlanName,
		},
		// Add payment URL for easy access
		"payment_url": fmt.Sprintf("http://localhost:8081/payment/checkout?order_id=%s", orderData.RazorpayOrderId),
	}

	pkghttp.Success(c, http.StatusOK, message, response)
}

// VerifyPayment - Verifies payment via payment service
func (h *Handler) VerifyPayment(c *gin.Context) {
	h.logger.Info("VerifyPayment endpoint called")

	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	var req VerifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", "error", err)
		pkghttp.Error(c, pkghttp.NewBadRequest("Invalid request format", err))
		return
	}

	success, message, subscription, err := h.paymentClient.VerifyPayment(
		c.Request.Context(),
		req.RazorpayOrderID,
		req.RazorpayPaymentID,
		req.RazorpaySignature,
	)

	if err != nil {
		h.logger.Error("Payment verification failed", "error", err, "userID", userID)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if !success {
		pkghttp.Error(c, pkghttp.NewBadRequest(message, nil))
		return
	}

	response := gin.H{
		"success": success,
		"message": message,
	}

	if subscription != nil {
		response["subscription"] = gin.H{
			"id":         subscription.Id,
			"plan_id":    subscription.PlanId,
			"status":     subscription.Status,
			"start_date": subscription.StartDate.AsTime(),
			"end_date":   subscription.EndDate.AsTime(),
			"amount":     subscription.Amount,
			"currency":   subscription.Currency,
			"is_active":  subscription.IsActive,
		}
	}

	h.logger.Info("Payment verified successfully", "userID", userID)
	pkghttp.Success(c, http.StatusOK, message, response)
}

// VerifyPaymentQuery - Handle verification via query parameters (for redirects)
func (h *Handler) VerifyPaymentQuery(c *gin.Context) {
	h.logger.Info("VerifyPaymentQuery endpoint called")

	// Extract payment parameters from Razorpay redirect
	orderID := c.Query("razorpay_order_id")
	paymentID := c.Query("razorpay_payment_id")
	signature := c.Query("razorpay_signature")

	if orderID == "" || paymentID == "" || signature == "" {
		h.logger.Error("Missing payment parameters")
		// ✅ Clean redirect - only error info
		c.Redirect(http.StatusFound, "http://localhost:8081/payment/failed?error=missing_parameters")
		return
	}

	// ✅ IMMEDIATE VERIFICATION - no user context needed (order provides context)
	success, _, _, err := h.paymentClient.VerifyPayment(
		c.Request.Context(),
		orderID,
		paymentID,
		signature,
	)

	if err != nil || !success {
		h.logger.Error("Payment verification failed", "error", err, "orderID", orderID)
		// ✅ Clean redirect - only order ID (safe to expose)
		c.Redirect(http.StatusFound, "http://localhost:8081/payment/failed?order_id="+orderID)
		return
	}

	h.logger.Info("Payment verified successfully", "orderID", orderID)

	// ✅ CLEAN SUCCESS REDIRECT - no sensitive payment details in URL
	successURL := "http://localhost:8081/payment/success?order_id=" + orderID
	c.Redirect(http.StatusFound, successURL)
}

// GetSubscription - Gets current subscription status
func (h *Handler) GetSubscription(c *gin.Context) {
	h.logger.Info("GetSubscription endpoint called")

	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	success, message, subscription, err := h.paymentClient.GetSubscriptionStatus(c.Request.Context())
	if err != nil {
		h.logger.Error("Failed to get subscription", "error", err, "userID", userID)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	response := gin.H{
		"success": success,
		"message": message,
	}

	if subscription != nil {
		response["subscription"] = gin.H{
			"id":         subscription.Id,
			"plan_id":    subscription.PlanId,
			"status":     subscription.Status,
			"start_date": subscription.StartDate.AsTime(),
			"end_date":   subscription.EndDate.AsTime(),
			"amount":     subscription.Amount,
			"currency":   subscription.Currency,
			"is_active":  subscription.IsActive,
		}
	}

	h.logger.Info("Subscription status retrieved", "userID", userID)
	pkghttp.Success(c, http.StatusOK, message, response)
}

// GetPaymentHistory - Gets payment history
func (h *Handler) GetPaymentHistory(c *gin.Context) {
	h.logger.Info("GetPaymentHistory endpoint called")

	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		pkghttp.Error(c, pkghttp.NewUnauthorized("Authentication required", nil))
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	success, message, payments, pagination, err := h.paymentClient.GetPaymentHistory(
		c.Request.Context(),
		int32(limit),
		int32(offset),
	)

	if err != nil {
		h.logger.Error("Failed to get payment history", "error", err, "userID", userID)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	responsePayments := make([]gin.H, len(payments))
	for i, payment := range payments {
		responsePayments[i] = gin.H{
			"id":                  payment.Id,
			"razorpay_order_id":   payment.RazorpayOrderId,
			"razorpay_payment_id": payment.RazorpayPaymentId,
			"amount":              payment.Amount,
			"currency":            payment.Currency,
			"status":              payment.Status,
			"payment_method":      payment.PaymentMethod,
			"created_at":          payment.CreatedAt.AsTime(),
		}
	}

	response := gin.H{
		"success":  success,
		"message":  message,
		"payments": responsePayments,
		"pagination": gin.H{
			"limit":    pagination.Limit,
			"offset":   pagination.Offset,
			"total":    pagination.Total,
			"has_more": pagination.HasMore,
		},
	}

	h.logger.Info("Payment history retrieved", "userID", userID, "count", len(payments))
	pkghttp.Success(c, http.StatusOK, message, response)
}

// PaymentStatus - Simple health check for payment service
func (h *Handler) PaymentStatus(c *gin.Context) {
	h.logger.Info("Payment status endpoint called")
	pkghttp.Success(c, http.StatusOK, "Payment service is ready", gin.H{
		"status":  "ready",
		"service": "payment",
	})
}

func (h *Handler) RedirectToPlans(c *gin.Context) {

	h.logger.Info("Redirecting to payment service plans")
	c.Redirect(http.StatusFound, "http://localhost:8081/payment/plans")
}
