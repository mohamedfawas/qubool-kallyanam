// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: payment/v1/payment.proto

package paymentpb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	PaymentService_CreatePaymentOrder_FullMethodName    = "/payment.v1.PaymentService/CreatePaymentOrder"
	PaymentService_GetSubscriptionStatus_FullMethodName = "/payment.v1.PaymentService/GetSubscriptionStatus"
	PaymentService_GetPaymentHistory_FullMethodName     = "/payment.v1.PaymentService/GetPaymentHistory"
	PaymentService_HandleWebhook_FullMethodName         = "/payment.v1.PaymentService/HandleWebhook"
	PaymentService_CreatePaymentURL_FullMethodName      = "/payment.v1.PaymentService/CreatePaymentURL"
	PaymentService_VerifyPayment_FullMethodName         = "/payment.v1.PaymentService/VerifyPayment"
)

// PaymentServiceClient is the client API for PaymentService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PaymentServiceClient interface {
	// Create a payment order for subscription
	CreatePaymentOrder(ctx context.Context, in *CreatePaymentOrderRequest, opts ...grpc.CallOption) (*CreatePaymentOrderResponse, error)
	// Get subscription status
	GetSubscriptionStatus(ctx context.Context, in *GetSubscriptionStatusRequest, opts ...grpc.CallOption) (*GetSubscriptionStatusResponse, error)
	// Get payment history
	GetPaymentHistory(ctx context.Context, in *GetPaymentHistoryRequest, opts ...grpc.CallOption) (*GetPaymentHistoryResponse, error)
	// Add webhook support
	HandleWebhook(ctx context.Context, in *WebhookRequest, opts ...grpc.CallOption) (*WebhookResponse, error)
	// Add simple redirect-based payment
	CreatePaymentURL(ctx context.Context, in *CreatePaymentURLRequest, opts ...grpc.CallOption) (*CreatePaymentURLResponse, error)
	VerifyPayment(ctx context.Context, in *VerifyPaymentRequest, opts ...grpc.CallOption) (*VerifyPaymentResponse, error)
}

type paymentServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPaymentServiceClient(cc grpc.ClientConnInterface) PaymentServiceClient {
	return &paymentServiceClient{cc}
}

func (c *paymentServiceClient) CreatePaymentOrder(ctx context.Context, in *CreatePaymentOrderRequest, opts ...grpc.CallOption) (*CreatePaymentOrderResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CreatePaymentOrderResponse)
	err := c.cc.Invoke(ctx, PaymentService_CreatePaymentOrder_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *paymentServiceClient) GetSubscriptionStatus(ctx context.Context, in *GetSubscriptionStatusRequest, opts ...grpc.CallOption) (*GetSubscriptionStatusResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetSubscriptionStatusResponse)
	err := c.cc.Invoke(ctx, PaymentService_GetSubscriptionStatus_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *paymentServiceClient) GetPaymentHistory(ctx context.Context, in *GetPaymentHistoryRequest, opts ...grpc.CallOption) (*GetPaymentHistoryResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetPaymentHistoryResponse)
	err := c.cc.Invoke(ctx, PaymentService_GetPaymentHistory_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *paymentServiceClient) HandleWebhook(ctx context.Context, in *WebhookRequest, opts ...grpc.CallOption) (*WebhookResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(WebhookResponse)
	err := c.cc.Invoke(ctx, PaymentService_HandleWebhook_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *paymentServiceClient) CreatePaymentURL(ctx context.Context, in *CreatePaymentURLRequest, opts ...grpc.CallOption) (*CreatePaymentURLResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CreatePaymentURLResponse)
	err := c.cc.Invoke(ctx, PaymentService_CreatePaymentURL_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *paymentServiceClient) VerifyPayment(ctx context.Context, in *VerifyPaymentRequest, opts ...grpc.CallOption) (*VerifyPaymentResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(VerifyPaymentResponse)
	err := c.cc.Invoke(ctx, PaymentService_VerifyPayment_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PaymentServiceServer is the server API for PaymentService service.
// All implementations must embed UnimplementedPaymentServiceServer
// for forward compatibility.
type PaymentServiceServer interface {
	// Create a payment order for subscription
	CreatePaymentOrder(context.Context, *CreatePaymentOrderRequest) (*CreatePaymentOrderResponse, error)
	// Get subscription status
	GetSubscriptionStatus(context.Context, *GetSubscriptionStatusRequest) (*GetSubscriptionStatusResponse, error)
	// Get payment history
	GetPaymentHistory(context.Context, *GetPaymentHistoryRequest) (*GetPaymentHistoryResponse, error)
	// Add webhook support
	HandleWebhook(context.Context, *WebhookRequest) (*WebhookResponse, error)
	// Add simple redirect-based payment
	CreatePaymentURL(context.Context, *CreatePaymentURLRequest) (*CreatePaymentURLResponse, error)
	VerifyPayment(context.Context, *VerifyPaymentRequest) (*VerifyPaymentResponse, error)
	mustEmbedUnimplementedPaymentServiceServer()
}

// UnimplementedPaymentServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedPaymentServiceServer struct{}

func (UnimplementedPaymentServiceServer) CreatePaymentOrder(context.Context, *CreatePaymentOrderRequest) (*CreatePaymentOrderResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreatePaymentOrder not implemented")
}
func (UnimplementedPaymentServiceServer) GetSubscriptionStatus(context.Context, *GetSubscriptionStatusRequest) (*GetSubscriptionStatusResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetSubscriptionStatus not implemented")
}
func (UnimplementedPaymentServiceServer) GetPaymentHistory(context.Context, *GetPaymentHistoryRequest) (*GetPaymentHistoryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetPaymentHistory not implemented")
}
func (UnimplementedPaymentServiceServer) HandleWebhook(context.Context, *WebhookRequest) (*WebhookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HandleWebhook not implemented")
}
func (UnimplementedPaymentServiceServer) CreatePaymentURL(context.Context, *CreatePaymentURLRequest) (*CreatePaymentURLResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreatePaymentURL not implemented")
}
func (UnimplementedPaymentServiceServer) VerifyPayment(context.Context, *VerifyPaymentRequest) (*VerifyPaymentResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method VerifyPayment not implemented")
}
func (UnimplementedPaymentServiceServer) mustEmbedUnimplementedPaymentServiceServer() {}
func (UnimplementedPaymentServiceServer) testEmbeddedByValue()                        {}

// UnsafePaymentServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PaymentServiceServer will
// result in compilation errors.
type UnsafePaymentServiceServer interface {
	mustEmbedUnimplementedPaymentServiceServer()
}

func RegisterPaymentServiceServer(s grpc.ServiceRegistrar, srv PaymentServiceServer) {
	// If the following call pancis, it indicates UnimplementedPaymentServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&PaymentService_ServiceDesc, srv)
}

func _PaymentService_CreatePaymentOrder_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreatePaymentOrderRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PaymentServiceServer).CreatePaymentOrder(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PaymentService_CreatePaymentOrder_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PaymentServiceServer).CreatePaymentOrder(ctx, req.(*CreatePaymentOrderRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PaymentService_GetSubscriptionStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetSubscriptionStatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PaymentServiceServer).GetSubscriptionStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PaymentService_GetSubscriptionStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PaymentServiceServer).GetSubscriptionStatus(ctx, req.(*GetSubscriptionStatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PaymentService_GetPaymentHistory_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetPaymentHistoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PaymentServiceServer).GetPaymentHistory(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PaymentService_GetPaymentHistory_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PaymentServiceServer).GetPaymentHistory(ctx, req.(*GetPaymentHistoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PaymentService_HandleWebhook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(WebhookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PaymentServiceServer).HandleWebhook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PaymentService_HandleWebhook_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PaymentServiceServer).HandleWebhook(ctx, req.(*WebhookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PaymentService_CreatePaymentURL_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreatePaymentURLRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PaymentServiceServer).CreatePaymentURL(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PaymentService_CreatePaymentURL_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PaymentServiceServer).CreatePaymentURL(ctx, req.(*CreatePaymentURLRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PaymentService_VerifyPayment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VerifyPaymentRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PaymentServiceServer).VerifyPayment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PaymentService_VerifyPayment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PaymentServiceServer).VerifyPayment(ctx, req.(*VerifyPaymentRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// PaymentService_ServiceDesc is the grpc.ServiceDesc for PaymentService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var PaymentService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "payment.v1.PaymentService",
	HandlerType: (*PaymentServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreatePaymentOrder",
			Handler:    _PaymentService_CreatePaymentOrder_Handler,
		},
		{
			MethodName: "GetSubscriptionStatus",
			Handler:    _PaymentService_GetSubscriptionStatus_Handler,
		},
		{
			MethodName: "GetPaymentHistory",
			Handler:    _PaymentService_GetPaymentHistory_Handler,
		},
		{
			MethodName: "HandleWebhook",
			Handler:    _PaymentService_HandleWebhook_Handler,
		},
		{
			MethodName: "CreatePaymentURL",
			Handler:    _PaymentService_CreatePaymentURL_Handler,
		},
		{
			MethodName: "VerifyPayment",
			Handler:    _PaymentService_VerifyPayment_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "payment/v1/payment.proto",
}
