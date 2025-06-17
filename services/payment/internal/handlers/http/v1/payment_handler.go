package v1

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/config"
	"github.com/mohamedfawas/qubool-kallyanam/services/payment/internal/domain/services"
)

type HTTPHandler struct {
	paymentService *services.PaymentService
	config         *config.Config
	logger         logging.Logger
}

func NewHTTPHandler(paymentService *services.PaymentService, config *config.Config, logger logging.Logger) *HTTPHandler {
	return &HTTPHandler{
		paymentService: paymentService,
		config:         config,
		logger:         logger,
	}
}

type PaymentPageData struct {
	PlanName        string
	DisplayAmount   string
	Amount          int64
	RazorpayKeyID   string
	RazorpayOrderID string
	GatewayURL      string
}

type PlansPageData struct {
	HasActiveSubscription bool
	Plans                 []PlanData
}

type PlanData struct {
	ID           string
	Name         string
	Amount       float64
	Currency     string
	Description  string
	Features     []string
	DurationDays int
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

func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "payment",
		"time":    time.Now().Unix(),
	})
}

func (h *HTTPHandler) ShowPlans(c *gin.Context) {
	h.logger.Info("ShowPlans page requested")

	// Convert config plans to display format
	plans := make([]PlanData, 0)
	for _, plan := range h.config.Plans.GetActivePlans() {
		plans = append(plans, PlanData{
			ID:           plan.ID,
			Name:         plan.Name,
			Amount:       plan.Amount,
			Currency:     plan.Currency,
			Description:  plan.Description,
			Features:     plan.Features,
			DurationDays: plan.DurationDays,
		})
	}

	data := PlansPageData{
		HasActiveSubscription: false,
		Plans:                 plans,
	}

	tmpl, err := template.ParseFiles("templates/plans.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Template error"})
		return
	}

	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		h.logger.Error("Failed to execute template", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Template execution error"})
	}
}

func (h *HTTPHandler) ShowPaymentPage(c *gin.Context) {
	h.logger.Info("ShowPaymentPage requested")

	// Get order ID from URL parameter
	orderID := c.Query("order_id")
	if orderID == "" {
		h.logger.Error("Missing order ID parameter")
		c.Redirect(http.StatusFound, "/payment/failed?error=missing_order_id")
		return
	}

	// Get order details using order ID (no user auth needed)
	orderDetails, err := h.paymentService.GetOrderDetails(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order details", "error", err, "orderID", orderID)
		c.Redirect(http.StatusFound, "/payment/failed?error=order_not_found")
		return
	}

	// Check if order is still valid for payment
	if orderDetails.Status != "created" && orderDetails.Status != "pending" {
		h.logger.Error("Order not available for payment", "orderID", orderID, "status", orderDetails.Status)
		c.Redirect(http.StatusFound, "/payment/failed?error=order_not_available")
		return
	}

	data := PaymentPageData{
		PlanName:        orderDetails.PlanName,
		DisplayAmount:   formatAmount(orderDetails.Amount),
		Amount:          orderDetails.Amount,
		RazorpayKeyID:   orderDetails.RazorpayKeyId,
		RazorpayOrderID: orderDetails.RazorpayOrderId,
		GatewayURL:      h.config.Gateway.Address,
	}

	tmpl, err := template.ParseFiles("templates/payment.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Template error"})
		return
	}

	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		h.logger.Error("Failed to execute template", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Template execution error"})
	}
}

func (h *HTTPHandler) ShowSuccessPage(c *gin.Context) {
	h.logger.Info("ShowSuccessPage requested")

	orderID := c.Query("order_id")
	if orderID == "" {
		h.logger.Error("Missing order ID")
		c.Redirect(http.StatusFound, "/payment/plans")
		return
	}

	// ✅ GET PAYMENT DETAILS FROM ORDER ID (no sensitive data from URL)
	paymentDetails, err := h.paymentService.GetPaymentDetailsByOrderID(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get payment details", "error", err, "orderID", orderID)
		c.Redirect(http.StatusFound, "/payment/failed?order_id="+orderID+"&error=payment_not_found")
		return
	}

	// Check if payment was actually successful
	if paymentDetails.Status != "success" {
		h.logger.Error("Payment not successful", "orderID", orderID, "status", paymentDetails.Status)
		c.Redirect(http.StatusFound, "/payment/failed?order_id="+orderID+"&error=payment_not_completed")
		return
	}

	data := SuccessPageData{
		Amount:     formatAmount(paymentDetails.Amount),
		OrderID:    orderID,
		PaymentID:  paymentDetails.PaymentID,
		ValidUntil: time.Now().AddDate(1, 0, 0).Format("Jan 2, 2006"),
	}

	tmpl, err := template.ParseFiles("templates/success.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Template error"})
		return
	}

	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		h.logger.Error("Failed to execute template", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Template execution error"})
	}
}
func (h *HTTPHandler) ShowFailedPage(c *gin.Context) {
	h.logger.Info("ShowFailedPage requested")

	data := FailedPageData{
		OrderID: c.Query("order_id"),
		Error:   c.DefaultQuery("error", "Payment could not be processed"),
	}

	tmpl, err := template.ParseFiles("templates/failed.html")
	if err != nil {
		h.logger.Error("Failed to parse template", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Template error"})
		return
	}

	c.Header("Content-Type", "text/html")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		h.logger.Error("Failed to execute template", "error", err)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": "Template execution error"})
	}
}

func formatAmount(amountInPaise int64) string {
	return fmt.Sprintf("%.0f", float64(amountInPaise)/100)
}
