package grpc

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/discovery"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry"
)

// ClientOption is a function that configures a Client
type ClientOption func(*ClientOptions)

// ClientOptions contains all client configuration
type ClientOptions struct {
	ServiceName        string
	Target             string
	UseDiscovery       bool
	Secure             bool
	Authority          string
	CertFile           string
	ServerNameOverride string
	UserAgent          string
	Timeout            time.Duration
	MaxRetries         int
	KeepaliveTime      time.Duration
	Headers            map[string]string
	Logger             logging.Logger
	Registry           discovery.Registry
	MetricsProvider    telemetry.MetricsProvider
}

// Client is a gRPC client with enhanced functionality
type Client struct {
	options ClientOptions
	conn    *grpc.ClientConn
	logger  logging.Logger
}

// NewClient creates a new gRPC client
func NewClient(opts ...ClientOption) (*Client, error) {
	// Default options
	options := ClientOptions{
		Timeout:       30 * time.Second,
		MaxRetries:    3,
		KeepaliveTime: 60 * time.Second,
		Headers:       make(map[string]string),
		Logger:        logging.Get().Named("grpc-client"),
	}

	// Apply options
	for _, opt := range opts {
		opt(&options)
	}

	client := &Client{
		options: options,
		logger:  options.Logger,
	}

	// Establish connection
	if err := client.connect(); err != nil {
		return nil, err
	}

	return client, nil
}

// WithServiceName sets the service name
func WithServiceName(name string) ClientOption {
	return func(o *ClientOptions) {
		o.ServiceName = name
	}
}

// WithTarget sets the target address
func WithTarget(target string) ClientOption {
	return func(o *ClientOptions) {
		o.Target = target
	}
}

// WithDiscovery enables service discovery
func WithDiscovery(useDiscovery bool) ClientOption {
	return func(o *ClientOptions) {
		o.UseDiscovery = useDiscovery
	}
}

// WithSecure enables TLS
func WithSecure(secure bool) ClientOption {
	return func(o *ClientOptions) {
		o.Secure = secure
	}
}

// WithCertFile sets the certificate file for TLS
func WithCertFile(certFile string) ClientOption {
	return func(o *ClientOptions) {
		o.CertFile = certFile
	}
}

// WithServerNameOverride sets the server name override for TLS
func WithServerNameOverride(serverName string) ClientOption {
	return func(o *ClientOptions) {
		o.ServerNameOverride = serverName
	}
}

// WithAuthority sets the authority for gRPC calls
func WithAuthority(authority string) ClientOption {
	return func(o *ClientOptions) {
		o.Authority = authority
	}
}

// WithUserAgent sets the user agent
func WithUserAgent(userAgent string) ClientOption {
	return func(o *ClientOptions) {
		o.UserAgent = userAgent
	}
}

// WithTimeout sets the timeout for requests
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *ClientOptions) {
		o.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retries
func WithMaxRetries(maxRetries int) ClientOption {
	return func(o *ClientOptions) {
		o.MaxRetries = maxRetries
	}
}

// WithKeepaliveTime sets the keepalive time
func WithKeepaliveTime(keepaliveTime time.Duration) ClientOption {
	return func(o *ClientOptions) {
		o.KeepaliveTime = keepaliveTime
	}
}

// WithHeader adds a header to all requests
func WithHeader(key, value string) ClientOption {
	return func(o *ClientOptions) {
		o.Headers[key] = value
	}
}

// WithLogger sets a custom logger
func WithLogger(logger logging.Logger) ClientOption {
	return func(o *ClientOptions) {
		o.Logger = logger
	}
}

// WithRegistry sets a service discovery registry
func WithRegistry(registry discovery.Registry) ClientOption {
	return func(o *ClientOptions) {
		o.Registry = registry
	}
}

// WithMetricsProvider sets a metrics provider
func WithMetricsProvider(provider telemetry.MetricsProvider) ClientOption {
	return func(o *ClientOptions) {
		o.MetricsProvider = provider
	}
}

// connect establishes a gRPC connection
func (c *Client) connect() error {
	target := c.options.Target

	// Use service discovery if enabled
	if c.options.UseDiscovery && c.options.Registry != nil && c.options.ServiceName != "" {
		query := discovery.ServiceQuery{
			Name:        c.options.ServiceName,
			OnlyHealthy: true,
		}

		services, err := c.options.Registry.GetService(context.Background(), query)
		if err != nil {
			return fmt.Errorf("service discovery failed: %w", err)
		}

		if len(services) == 0 {
			return fmt.Errorf("no healthy instances found for service: %s", c.options.ServiceName)
		}

		// Pick the first service (in a real app, you might want load balancing)
		instance := services[0]
		target = fmt.Sprintf("%s:%d", instance.Address, instance.Port)
	}

	// If no target is specified, we can't connect
	if target == "" {
		return fmt.Errorf("no target specified for gRPC connection")
	}

	// Setup dial options
	dialOpts := []grpc.DialOption{
		// Set backoff config
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  100 * time.Millisecond,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   2 * time.Second,
			},
			MinConnectTimeout: 5 * time.Second,
		}),
		// Add keepalive
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                c.options.KeepaliveTime,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		// Add OpenTelemetry instrumentation
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
		// Add our custom interceptors
		grpc.WithUnaryInterceptor(c.unaryInterceptor),
		grpc.WithStreamInterceptor(c.streamInterceptor),
	}

	// Add authority if specified
	if c.options.Authority != "" {
		dialOpts = append(dialOpts, grpc.WithAuthority(c.options.Authority))
	}

	// Add user agent if specified
	if c.options.UserAgent != "" {
		dialOpts = append(dialOpts, grpc.WithUserAgent(c.options.UserAgent))
	}

	// Setup security
	if c.options.Secure {
		if c.options.CertFile != "" {
			creds, err := credentials.NewClientTLSFromFile(c.options.CertFile, c.options.ServerNameOverride)
			if err != nil {
				return fmt.Errorf("failed to create TLS credentials: %w", err)
			}
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
		} else {
			// Use system certificates
			dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(nil)))
		}
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Connect with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c.logger.Info("Connecting to gRPC service",
		logging.String("target", target),
		logging.String("service", c.options.ServiceName),
	)

	// Establish connection
	conn, err := grpc.DialContext(ctx, target, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	c.conn = conn
	return nil
}

// Client returns the gRPC client connection
func (c *Client) Client() *grpc.ClientConn {
	return c.conn
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ContextWithTimeout returns a context with the client's default timeout
func (c *Client) ContextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.options.Timeout)
}

// ContextWithHeaders returns a context with the client's default headers
func (c *Client) ContextWithHeaders(ctx context.Context) context.Context {
	if len(c.options.Headers) == 0 {
		return ctx
	}

	md := metadata.New(c.options.Headers)
	return metadata.NewOutgoingContext(ctx, md)
}

// unaryInterceptor intercepts unary RPC calls
func (c *Client) unaryInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	startTime := time.Now()

	// Add default headers
	if len(c.options.Headers) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(c.options.Headers))
	}

	// Define retry logic
	var lastErr error
	for attempt := 0; attempt <= c.options.MaxRetries; attempt++ {
		// If this is a retry, add some backoff
		if attempt > 0 {
			backoffTime := time.Duration(attempt*attempt*50) * time.Millisecond
			time.Sleep(backoffTime)

			c.logger.Debug("Retrying gRPC call",
				logging.String("method", method),
				logging.Int("attempt", attempt),
				logging.Duration("backoff", backoffTime),
			)
		}

		// Make the call
		err := invoker(ctx, method, req, reply, cc, opts...)

		// If successful or non-retryable error, return
		if err == nil || !c.isRetryable(err) {
			// Record metrics if provider is available
			if c.options.MetricsProvider != nil {
				latency := time.Since(startTime)
				statusCode := "OK"

				if err != nil {
					statusCode = status.Code(err).String()
				}

				// Record request count and latency
				if counter := c.options.MetricsProvider.Counter("grpc_client_requests_total",
					"Total number of gRPC client requests",
					"method", "status"); counter != nil {
					counter.Inc(method, statusCode)
				}

				if histogram := c.options.MetricsProvider.Histogram("grpc_client_request_duration_seconds",
					"gRPC client request duration in seconds",
					[]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
					"method", "status"); histogram != nil {
					histogram.Observe(latency.Seconds(), method, statusCode)
				}
			}

			return err
		}

		// Save the error for potential retry
		lastErr = err
	}

	// If we get here, we've exhausted retries
	return fmt.Errorf("gRPC call failed after %d attempts: %w", c.options.MaxRetries+1, lastErr)
}

// streamInterceptor intercepts streaming RPC calls
func (c *Client) streamInterceptor(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	startTime := time.Now()

	// Add default headers
	if len(c.options.Headers) > 0 {
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(c.options.Headers))
	}

	// No retry for streaming calls, just invoke
	stream, err := streamer(ctx, desc, cc, method, opts...)

	// Record metrics if provider is available and call completed
	if c.options.MetricsProvider != nil {
		latency := time.Since(startTime)
		statusCode := "OK"

		if err != nil {
			statusCode = status.Code(err).String()
		}

		// Record stream start
		if counter := c.options.MetricsProvider.Counter("grpc_client_streams_started_total",
			"Total number of gRPC client streams started",
			"method", "status"); counter != nil {
			counter.Inc(method, statusCode)
		}

		// For successful starts, also track latency
		if err == nil {
			if histogram := c.options.MetricsProvider.Histogram("grpc_client_stream_start_duration_seconds",
				"gRPC client stream start duration in seconds",
				[]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
				"method"); histogram != nil {
				histogram.Observe(latency.Seconds(), method)
			}
		}
	}

	return stream, err
}

// isRetryable determines if a gRPC error is retryable
func (c *Client) isRetryable(err error) bool {
	// Extract the gRPC status code
	code := status.Code(err)

	// These status codes are generally retryable
	retryableCodes := map[codes.Code]bool{
		codes.Unavailable:       true, // Server is currently unavailable
		codes.ResourceExhausted: true, // Resource has been exhausted
		codes.Aborted:           true, // Operation was aborted
		codes.DeadlineExceeded:  true, // Deadline expired
		codes.Internal:          true, // Internal errors
	}

	return retryableCodes[code]
}
