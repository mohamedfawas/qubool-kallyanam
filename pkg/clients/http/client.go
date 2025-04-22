package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/discovery"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/telemetry"
)

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// Client is an HTTP client with enhanced functionality
type Client struct {
	baseClient      *http.Client
	baseURL         string
	headers         map[string]string
	errorHandler    ErrorHandler
	logger          logging.Logger
	retryPolicy     RetryPolicy
	resolver        discovery.Registry
	metricsProvider telemetry.MetricsProvider
}

// ErrorHandler processes HTTP errors
type ErrorHandler func(resp *http.Response, body []byte) error

// RetryPolicy defines when to retry requests
type RetryPolicy struct {
	MaxRetries  int
	MaxDuration time.Duration
	Predicate   func(error, *http.Response) bool
}

// DefaultRetryPolicy returns a standard retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:  3,
		MaxDuration: 10 * time.Second,
		Predicate: func(err error, resp *http.Response) bool {
			// Retry on connection errors or 5xx server errors
			if err != nil {
				return true
			}
			return resp != nil && resp.StatusCode >= 500
		},
	}
}

// NewClient creates a new HTTP client
func NewClient(opts ...ClientOption) *Client {
	// Create transport with OpenTelemetry instrumentation
	transport := otelhttp.NewTransport(http.DefaultTransport)

	// Create default client
	client := &Client{
		baseClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		headers:     make(map[string]string),
		retryPolicy: DefaultRetryPolicy(),
		logger:      logging.Get().Named("http-client"),
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// WithBaseURL sets the base URL for the client
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithTimeout sets the timeout for requests
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.baseClient.Timeout = timeout
	}
}

// WithHeader adds a header to all requests
func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers[key] = value
	}
}

// WithAuth adds authentication to all requests
func WithAuth(scheme, token string) ClientOption {
	return func(c *Client) {
		c.headers["Authorization"] = fmt.Sprintf("%s %s", scheme, token)
	}
}

// WithErrorHandler sets a custom error handler
func WithErrorHandler(handler ErrorHandler) ClientOption {
	return func(c *Client) {
		c.errorHandler = handler
	}
}

// WithRetryPolicy sets a custom retry policy
func WithRetryPolicy(policy RetryPolicy) ClientOption {
	return func(c *Client) {
		c.retryPolicy = policy
	}
}

// WithLogger sets a custom logger
func WithLogger(logger logging.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithServiceDiscovery sets a service discovery registry
func WithServiceDiscovery(resolver discovery.Registry) ClientOption {
	return func(c *Client) {
		c.resolver = resolver
	}
}

// WithMetricsProvider sets a metrics provider
func WithMetricsProvider(provider telemetry.MetricsProvider) ClientOption {
	return func(c *Client) {
		c.metricsProvider = provider
	}
}

// Request represents an HTTP request
type Request struct {
	Method      string
	Path        string
	QueryParams map[string]string
	Headers     map[string]string
	Body        interface{}
	// Service discovery options
	ServiceName string
	// If true, will resolve the URL using service discovery
	UseDiscovery bool
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// Do executes an HTTP request with retries
func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	startTime := time.Now()
	path := req.Path
	method := req.Method

	// Build the URL
	var fullURL string
	if req.UseDiscovery && req.ServiceName != "" && c.resolver != nil {
		// Use service discovery to find the service
		query := discovery.ServiceQuery{
			Name:        req.ServiceName,
			OnlyHealthy: true,
		}

		services, err := c.resolver.GetService(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("service discovery failed: %w", err)
		}

		if len(services) == 0 {
			return nil, fmt.Errorf("no healthy instances found for service: %s", req.ServiceName)
		}

		// Pick the first service (in a real app, you might want load balancing)
		instance := services[0]

		protocol := "http"
		if instance.Secure {
			protocol = "https"
		}

		fullURL = fmt.Sprintf("%s://%s:%d%s", protocol, instance.Address, instance.Port, path)
	} else {
		// Use the baseURL
		fullURL = c.baseURL + path
	}

	// Add query parameters
	if len(req.QueryParams) > 0 {
		fullURL += "?"
		first := true
		for k, v := range req.QueryParams {
			if !first {
				fullURL += "&"
			}
			fullURL += fmt.Sprintf("%s=%s", k, v)
			first = false
		}
	}

	// Create retry backoff
	expBackoff := backoff.NewExponentialBackOff()
	expBackoff.MaxElapsedTime = c.retryPolicy.MaxDuration

	// Define the operation to retry
	operation := func() (*Response, error) {
		// Prepare request body
		var bodyReader io.Reader
		if req.Body != nil {
			bodyBytes, err := json.Marshal(req.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewReader(bodyBytes)
		}

		// Create HTTP request
		httpReq, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Add default headers
		for k, v := range c.headers {
			httpReq.Header.Set(k, v)
		}

		// Add request-specific headers
		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}

		// Set content type for requests with body
		if req.Body != nil && httpReq.Header.Get("Content-Type") == "" {
			httpReq.Header.Set("Content-Type", "application/json")
		}

		// Set accept header if not specified
		if httpReq.Header.Get("Accept") == "" {
			httpReq.Header.Set("Accept", "application/json")
		}

		// Execute request
		httpResp, err := c.baseClient.Do(httpReq)

		// Check if we should retry
		if c.shouldRetry(err, httpResp) {
			c.logger.Debug("Retrying request",
				logging.String("method", method),
				logging.String("url", fullURL),
				logging.Error(err),
			)

			if httpResp != nil {
				httpResp.Body.Close()
			}

			return nil, fmt.Errorf("request failed, will retry: %w", err)
		}

		if err != nil {
			return nil, err
		}

		// Read response body
		defer httpResp.Body.Close()
		respBody, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		response := &Response{
			StatusCode: httpResp.StatusCode,
			Headers:    httpResp.Header,
			Body:       respBody,
		}

		// Handle errors based on status code
		if httpResp.StatusCode >= 400 && c.errorHandler != nil {
			err = c.errorHandler(httpResp, respBody)
			if err != nil {
				return response, err
			}
		}

		// Record metrics if provider is available
		if c.metricsProvider != nil {
			latency := time.Since(startTime)
			status := fmt.Sprintf("%d", httpResp.StatusCode)

			// Record request count and latency
			if counter := c.metricsProvider.Counter("http_client_requests_total",
				"Total number of HTTP client requests",
				"method", "path", "status"); counter != nil {
				counter.Inc(method, path, status)
			}

			if histogram := c.metricsProvider.Histogram("http_client_request_duration_seconds",
				"HTTP client request duration in seconds",
				[]float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
				"method", "path", "status"); histogram != nil {
				histogram.Observe(latency.Seconds(), method, path, status)
			}
		}

		return response, nil
	}

	// Execute with retries
	var resp *Response
	var err error

	if c.retryPolicy.MaxRetries > 0 {
		// Define retry notification function
		notify := func(err error, duration time.Duration) {
			c.logger.Debug("Retrying request after error",
				logging.String("method", method),
				logging.String("url", fullURL),
				logging.Error(err),
				logging.Duration("retry_after", duration),
			)
		}

		// Setup retry logic
		b := backoff.WithMaxRetries(expBackoff, uint64(c.retryPolicy.MaxRetries))
		bCtx := backoff.WithContext(b, ctx)

		// Execute with retries
		err = backoff.RetryNotify(func() error {
			var opErr error
			resp, opErr = operation()
			return opErr
		}, bCtx, notify)
	} else {
		// Execute without retries
		resp, err = operation()
	}

	if err != nil {
		return nil, fmt.Errorf("request failed after retries: %w", err)
	}

	return resp, nil
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string, queryParams map[string]string) (*Response, error) {
	return c.Do(ctx, Request{
		Method:      http.MethodGet,
		Path:        path,
		QueryParams: queryParams,
	})
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, Request{
		Method: http.MethodPost,
		Path:   path,
		Body:   body,
	})
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}) (*Response, error) {
	return c.Do(ctx, Request{
		Method: http.MethodPut,
		Path:   path,
		Body:   body,
	})
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return c.Do(ctx, Request{
		Method: http.MethodDelete,
		Path:   path,
	})
}

// Decode decodes the response body into the provided target
func (c *Client) Decode(resp *Response, target interface{}) error {
	return json.Unmarshal(resp.Body, target)
}

// shouldRetry determines if a request should be retried
func (c *Client) shouldRetry(err error, resp *http.Response) bool {
	if c.retryPolicy.Predicate != nil {
		return c.retryPolicy.Predicate(err, resp)
	}

	// Default retry logic
	if err != nil {
		return true
	}

	return resp != nil && resp.StatusCode >= 500
}

// DefaultErrorHandler provides a standard HTTP error handler
func DefaultErrorHandler(resp *http.Response, body []byte) error {
	// Handle common status codes
	switch resp.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("resource not found: %s", resp.Request.URL)
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: invalid credentials")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: insufficient permissions")
	case http.StatusBadRequest:
		// Try to parse error message from body
		var errorResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != "" {
			return fmt.Errorf("bad request: %s", errorResp.Error)
		}
		return fmt.Errorf("bad request: %s", string(body))
	}

	// Handle 5xx errors
	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error: %s (status=%d)", resp.Status, resp.StatusCode)
	}

	// Generic error for other cases
	return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
}
