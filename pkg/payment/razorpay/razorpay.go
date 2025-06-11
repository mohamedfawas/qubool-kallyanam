package razorpay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/razorpay/razorpay-go"
)

type Service struct {
	client    *razorpay.Client
	secretKey string // Razorpay key secret (used for verifying signatures)
	keyID     string // Razorpay key ID (public part, shared with frontend)
}

type Order struct {
	ID       string `json:"id"`       // Razorpay order ID
	Amount   int64  `json:"amount"`   // Amount in paisa
	Currency string `json:"currency"` // Currency code
}

func NewService(keyID, keySecret string) *Service {
	client := razorpay.NewClient(keyID, keySecret)
	return &Service{
		client:    client,
		secretKey: keySecret,
		keyID:     keyID,
	}
}

// GetKeyID returns the Razorpay Key ID (usually sent to frontend to initialize payment)
func (s *Service) GetKeyID() string {
	return s.keyID
}

// CreateOrder creates a new Razorpay order with given amount and currency
func (s *Service) CreateOrder(amount int64, currency string) (*Order, error) {
	// Generate a simple receipt ID
	receipt := fmt.Sprintf("order_%d", time.Now().Unix())

	data := map[string]interface{}{
		"amount":   amount,   // Amount in paise
		"currency": currency, // INR
		"receipt":  receipt,  // Simple receipt ID
	}

	// Send request to Razorpay to create order
	body, err := s.client.Order.Create(data, nil)
	if err != nil {
		fmt.Printf("Razorpay order creation failed: %v\n", err)
		return nil, fmt.Errorf("failed to create razorpay order: %w", err)
	}

	fmt.Printf("Razorpay order response: %+v\n", body)

	// Extract order ID from response
	id, ok := body["id"].(string)
	if !ok {
		return nil, errors.New("invalid order id in response")
	}

	// Extract amount from response
	responseAmount, ok := body["amount"].(float64)
	if !ok {
		return nil, errors.New("invalid amount in response")
	}

	// Extract currency from response
	responseCurrency, ok := body["currency"].(string)
	if !ok {
		return nil, errors.New("invalid currency in response")
	}

	order := &Order{
		ID:       id,
		Amount:   int64(responseAmount),
		Currency: responseCurrency,
	}

	return order, nil
}

// VerifyPaymentSignature validates Razorpay payment signature to ensure payment integrity
func (s *Service) VerifyPaymentSignature(attributes map[string]interface{}) error {
	orderID, ok := attributes["razorpay_order_id"].(string)
	if !ok {
		return errors.New("invalid order id")
	}

	paymentID, ok := attributes["razorpay_payment_id"].(string)
	if !ok {
		return errors.New("invalid payment id")
	}

	signature, ok := attributes["razorpay_signature"].(string)
	if !ok {
		return errors.New("invalid signature")
	}

	// Construct the message for HMAC (as per Razorpay: order_id|payment_id)
	message := orderID + "|" + paymentID

	// Compute HMAC-SHA256 using secret key and the message
	h := hmac.New(sha256.New, []byte(s.secretKey))
	h.Write([]byte(message))
	expectedSignature := hex.EncodeToString(h.Sum(nil)) // Convert to hex string

	// Compare computed signature with the one received from Razorpay
	if expectedSignature != signature {
		return errors.New("signature verification failed")
	}

	return nil
}

// Add to existing Service struct
func (s *Service) VerifyWebhookSignature(signature, payload string) error {
	// Implement Razorpay webhook signature verification
	expectedSignature := s.generateWebhookSignature(payload)
	if expectedSignature != signature {
		return errors.New("webhook signature verification failed")
	}
	return nil
}

func (s *Service) generateWebhookSignature(payload string) string {
	h := hmac.New(sha256.New, []byte(s.secretKey))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

// Create a simpler order creation that works better with redirects
func (s *Service) CreateRedirectOrder(amount int64, currency, callbackURL string) (*Order, error) {
	receipt := fmt.Sprintf("order_%d", time.Now().Unix())

	data := map[string]interface{}{
		"amount":       amount,
		"currency":     currency,
		"receipt":      receipt,
		"callback_url": callbackURL, // Add callback URL for redirect flow
	}

	body, err := s.client.Order.Create(data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create razorpay order: %w", err)
	}

	id, ok := body["id"].(string)
	if !ok {
		return nil, errors.New("invalid order id in response")
	}

	return &Order{
		ID:       id,
		Amount:   amount,
		Currency: currency,
	}, nil
}
