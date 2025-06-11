package payment

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	pkghttp "github.com/mohamedfawas/qubool-kallyanam/pkg/http"
	"github.com/mohamedfawas/qubool-kallyanam/services/gateway/internal/middleware"
)

type PaymentPageData struct {
	PlanName        string
	DisplayAmount   string // For display in rupees (e.g., "1000")
	Amount          string // For Razorpay in paise as string (e.g., "100000")
	RazorpayKeyID   string
	RazorpayOrderID string
	Token           string
	UserName        string
	UserEmail       string
	UserPhone       string
}

type SuccessPageData struct {
	Amount     string
	OrderID    string
	PaymentID  string
	ValidUntil string
}

type FailedPageData struct {
	OrderID string
	Error   string
}

type PlansPageData struct {
	HasActiveSubscription bool
}

func (h *Handler) ShowPlans(c *gin.Context) {
	h.logger.Info("ShowPlans page requested")

	// Check if user has active subscription
	hasActiveSubscription := false
	_, exists := c.Get(middleware.UserIDKey)
	if exists {
		_, _, subscription, err := h.paymentClient.GetSubscriptionStatus(c.Request.Context())
		if err == nil && subscription != nil && subscription.IsActive {
			hasActiveSubscription = true
		}
	}

	data := PlansPageData{
		HasActiveSubscription: hasActiveSubscription,
	}

	tmpl, err := template.ParseFiles("static/html/subscription_plans.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		pkghttp.Error(c, pkghttp.NewInternalServerError("Template error", err))
		return
	}

	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		h.logger.Error("Failed to execute template", "error", err)
		pkghttp.Error(c, pkghttp.NewInternalServerError("Template execution error", err))
		return
	}
}

func (h *Handler) ShowPaymentPage(c *gin.Context) {
	h.logger.Info("ShowPaymentPage requested")

	_, exists := c.Get(middleware.UserIDKey)
	if !exists {
		h.logger.Debug("Missing user ID in context")
		c.Redirect(http.StatusFound, "/login")
		return
	}

	// Create payment order
	success, message, orderData, err := h.paymentClient.CreatePaymentOrder(c.Request.Context(), "premium_365")
	if err != nil {
		h.logger.Error("Failed to create payment order", "error", err)
		pkghttp.Error(c, pkghttp.FromGRPCError(err))
		return
	}

	if !success {
		h.logger.Error("Payment order creation failed", "message", message)
		pkghttp.Error(c, pkghttp.NewInternalServerError("Failed to create payment order", nil))
		return
	}

	// Get user token for API calls
	token, _ := c.Get("token")
	tokenStr := ""
	if token != nil {
		tokenStr = token.(string)
	}

	// Convert amount from paise to rupees for display
	displayAmount := fmt.Sprintf("%.0f", float64(orderData.Amount)/100)

	data := PaymentPageData{
		PlanName:        orderData.PlanName,
		DisplayAmount:   displayAmount,
		Amount:          fmt.Sprintf("%d", orderData.Amount), // Convert to string for template
		RazorpayKeyID:   orderData.RazorpayKeyId,
		RazorpayOrderID: orderData.RazorpayOrderId,
		Token:           tokenStr,
		UserName:        "User", // You can get this from user service if needed
		UserEmail:       "",     // You can get this from user service if needed
		UserPhone:       "",     // You can get this from user service if needed
	}

	tmpl, err := template.ParseFiles("static/html/payment.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		pkghttp.Error(c, pkghttp.NewInternalServerError("Template error", err))
		return
	}

	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		h.logger.Error("Failed to execute template", "error", err)
		pkghttp.Error(c, pkghttp.NewInternalServerError("Template execution error", err))
		return
	}
}

func (h *Handler) ShowSuccessPage(c *gin.Context) {
	h.logger.Info("ShowSuccessPage requested")

	orderID := c.Query("order_id")
	if orderID == "" {
		c.Redirect(http.StatusFound, "/payment/plans")
		return
	}

	// You can fetch payment details here if needed
	data := SuccessPageData{
		Amount:     "1000",
		OrderID:    orderID,
		PaymentID:  c.Query("payment_id"),
		ValidUntil: time.Now().AddDate(1, 0, 0).Format("Jan 2, 2006"),
	}

	tmpl, err := template.ParseFiles("static/html/payment_success.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		pkghttp.Error(c, pkghttp.NewInternalServerError("Template error", err))
		return
	}

	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		h.logger.Error("Failed to execute template", "error", err)
		pkghttp.Error(c, pkghttp.NewInternalServerError("Template execution error", err))
		return
	}
}

func (h *Handler) ShowFailedPage(c *gin.Context) {
	h.logger.Info("ShowFailedPage requested")

	data := FailedPageData{
		OrderID: c.Query("order_id"),
		Error:   c.DefaultQuery("error", "Payment could not be processed"),
	}

	tmpl, err := template.ParseFiles("static/html/payment_failed.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		pkghttp.Error(c, pkghttp.NewInternalServerError("Template error", err))
		return
	}

	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		h.logger.Error("Failed to execute template", "error", err)
		pkghttp.Error(c, pkghttp.NewInternalServerError("Template execution error", err))
		return
	}
}
