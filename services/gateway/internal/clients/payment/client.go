package payment

import (
	"context"
	"fmt"

	paymentpb "github.com/mohamedfawas/qubool-kallyanam/api/proto/payment/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

type Client struct {
	conn   *grpc.ClientConn
	client paymentpb.PaymentServiceClient
}

func NewClient(address string) (*Client, error) {
	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client: %w", err)
	}

	return &Client{
		conn:   conn,
		client: paymentpb.NewPaymentServiceClient(conn),
	}, nil
}

// CreatePaymentOrder creates a payment order
func (c *Client) CreatePaymentOrder(ctx context.Context, planID string) (bool, string, *paymentpb.PaymentOrderData, error) {
	// Extract user ID from context and add to metadata
	var md metadata.MD
	if userID, ok := ctx.Value("user-id").(string); ok {
		md = metadata.New(map[string]string{
			"user-id": userID,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	resp, err := c.client.CreatePaymentOrder(ctx, &paymentpb.CreatePaymentOrderRequest{
		PlanId: planID,
	})
	if err != nil {
		return false, "", nil, err
	}

	return resp.Success, resp.Message, resp.OrderData, nil
}

// GetSubscriptionStatus gets current subscription status
func (c *Client) GetSubscriptionStatus(ctx context.Context) (bool, string, *paymentpb.SubscriptionData, error) {
	// Extract user ID from context and add to metadata
	var md metadata.MD
	if userID, ok := ctx.Value("user-id").(string); ok {
		md = metadata.New(map[string]string{
			"user-id": userID,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	resp, err := c.client.GetSubscriptionStatus(ctx, &paymentpb.GetSubscriptionStatusRequest{})
	if err != nil {
		return false, "", nil, err
	}

	return resp.Success, resp.Message, resp.Subscription, nil
}

// GetPaymentHistory gets payment history
func (c *Client) GetPaymentHistory(ctx context.Context, limit, offset int32) (bool, string, []*paymentpb.PaymentData, *paymentpb.PaginationData, error) {
	// Extract user ID from context and add to metadata
	var md metadata.MD
	if userID, ok := ctx.Value("user-id").(string); ok {
		md = metadata.New(map[string]string{
			"user-id": userID,
		})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	resp, err := c.client.GetPaymentHistory(ctx, &paymentpb.GetPaymentHistoryRequest{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return false, "", nil, nil, err
	}

	return resp.Success, resp.Message, resp.Payments, resp.Pagination, nil
}

// VerifyPayment verifies a payment callback (no auth needed - public callback)
func (c *Client) VerifyPayment(ctx context.Context, razorpayOrderID, razorpayPaymentID, razorpaySignature string) (bool, string, *paymentpb.SubscriptionData, error) {
	resp, err := c.client.VerifyPayment(ctx, &paymentpb.VerifyPaymentRequest{
		RazorpayOrderId:   razorpayOrderID,
		RazorpayPaymentId: razorpayPaymentID,
		RazorpaySignature: razorpaySignature,
	})
	if err != nil {
		return false, "", nil, err
	}

	return resp.Success, resp.Message, resp.Subscription, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}
