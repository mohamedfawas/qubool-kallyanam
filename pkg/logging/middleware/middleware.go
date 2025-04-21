package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

const (
	requestIDHeader     = "X-Request-ID"
	correlationIDHeader = "X-Correlation-ID"
	maxBodyLogSize      = 10 * 1024 // 10 KB
)

// HTTP middleware

// HTTPMiddleware returns a middleware that logs HTTP requests and responses
func HTTPMiddleware(skipPaths []string) func(http.Handler) http.Handler {
	skipPathMap := make(map[string]bool, len(skipPaths))
	for _, path := range skipPaths {
		skipPathMap[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for certain paths
			if skipPathMap[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			logger := logging.Get()

			// Ensure we have a request ID
			requestID := r.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = uuid.New().String()
				r.Header.Set(requestIDHeader, requestID)
			}

			// Get correlation ID if present, or use request ID
			correlationID := r.Header.Get(correlationIDHeader)
			if correlationID == "" {
				correlationID = requestID
			}

			// Enhance context with request information
			ctx := r.Context()
			ctx = logging.WithRequestID(ctx, requestID)
			ctx = logging.WithCorrelationID(ctx, correlationID)
			ctx = logging.WithIPAddress(ctx, r.RemoteAddr)
			ctx = logging.WithUserAgent(ctx, r.UserAgent())

			// Update request with enhanced context
			r = r.WithContext(ctx)

			// Create a response wrapper to capture status code
			rw := newResponseWriter(w)

			// Log the request
			logger.WithContext(ctx).Info("HTTP request started",
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.String("query", r.URL.RawQuery),
			)

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Prepare status code for logging
			statusCode := rw.status

			// Log at appropriate level based on status code
			logFn := logger.WithContext(ctx).Info
			if statusCode >= 500 {
				logFn = logger.WithContext(ctx).Error
			} else if statusCode >= 400 {
				logFn = logger.WithContext(ctx).Warn
			}

			// Log the response
			logFn("HTTP request completed",
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.Int("status", statusCode),
				logging.Duration("duration", duration),
				logging.Int("response_size", rw.size),
			)
		})
	}
}

// responseWriter is a wrapper for http.ResponseWriter that captures status code and response size
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		status:         http.StatusOK, // Default status code
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// gRPC middleware

// UnaryServerInterceptor returns a gRPC unary server interceptor for logging
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		startTime := time.Now()
		logger := logging.Get()

		// Extract metadata
		md, _ := metadata.FromIncomingContext(ctx)

		// Get or generate request ID
		requestIDs := md.Get(requestIDHeader)
		requestID := ""
		if len(requestIDs) > 0 {
			requestID = requestIDs[0]
		} else {
			requestID = uuid.New().String()
			md.Set(requestIDHeader, requestID)
			ctx = metadata.NewIncomingContext(ctx, md)
		}

		// Get correlation ID
		correlationIDs := md.Get(correlationIDHeader)
		correlationID := ""
		if len(correlationIDs) > 0 {
			correlationID = correlationIDs[0]
		} else {
			correlationID = requestID
		}

		// Enhance context with request information
		ctx = logging.WithRequestID(ctx, requestID)
		ctx = logging.WithCorrelationID(ctx, correlationID)

		// Log the request
		logger.WithContext(ctx).Info("gRPC request started",
			logging.String("method", info.FullMethod),
			logging.Any("request", sanitizeForLogging(req)),
		)

		// Invoke the handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(startTime)

		// Choose log level based on error
		logFn := logger.WithContext(ctx).Info
		if err != nil {
			st, _ := status.FromError(err)
			logFn = logger.WithContext(ctx).Error
			logger.WithContext(ctx).Error("gRPC request failed",
				logging.String("method", info.FullMethod),
				logging.String("code", st.Code().String()),
				logging.String("message", st.Message()),
				logging.Duration("duration", duration),
			)
		} else {
			// Log the response
			logFn("gRPC request completed",
				logging.String("method", info.FullMethod),
				logging.Duration("duration", duration),
				logging.Any("response", sanitizeForLogging(resp)),
			)
		}

		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for logging
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		startTime := time.Now()
		logger := logging.Get()

		// Extract context and metadata
		ctx := ss.Context()
		md, _ := metadata.FromIncomingContext(ctx)

		// Get or generate request ID
		requestIDs := md.Get(requestIDHeader)
		requestID := ""
		if len(requestIDs) > 0 {
			requestID = requestIDs[0]
		} else {
			requestID = uuid.New().String()
			md.Set(requestIDHeader, requestID)
			ctx = metadata.NewIncomingContext(ctx, md)
		}

		// Get correlation ID
		correlationIDs := md.Get(correlationIDHeader)
		correlationID := ""
		if len(correlationIDs) > 0 {
			correlationID = correlationIDs[0]
		} else {
			correlationID = requestID
		}

		// Enhance context with request information
		ctx = logging.WithRequestID(ctx, requestID)
		ctx = logging.WithCorrelationID(ctx, correlationID)

		// Create a wrapped stream with the enhanced context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Log the stream start
		logger.WithContext(ctx).Info("gRPC stream started",
			logging.String("method", info.FullMethod),
			logging.Bool("client_stream", info.IsClientStream),
			logging.Bool("server_stream", info.IsServerStream),
		)

		// Handle the stream
		err := handler(srv, wrappedStream)

		// Calculate duration
		duration := time.Since(startTime)

		// Choose log level based on error
		logFn := logger.WithContext(ctx).Info
		if err != nil {
			st, _ := status.FromError(err)
			logFn = logger.WithContext(ctx).Error
			logger.WithContext(ctx).Error("gRPC stream failed",
				logging.String("method", info.FullMethod),
				logging.String("code", st.Code().String()),
				logging.String("message", st.Message()),
				logging.Duration("duration", duration),
			)
		} else {
			// Log the stream completion
			logFn("gRPC stream completed",
				logging.String("method", info.FullMethod),
				logging.Duration("duration", duration),
			)
		}

		return err
	}
}

// wrappedServerStream wraps grpc.ServerStream to override context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// sanitizeForLogging prevents sensitive data from being logged
func sanitizeForLogging(obj interface{}) interface{} {
	// This is a placeholder for a more complex implementation
	// that would sanitize passwords, tokens, etc.
	return obj
}
