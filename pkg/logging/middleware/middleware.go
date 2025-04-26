// middleware/middleware.go
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
)

// HTTP middleware

// HTTPLogger returns middleware that logs HTTP requests and responses
func HTTPLogger(skipPaths []string) func(http.Handler) http.Handler {
	skipPathMap := make(map[string]bool, len(skipPaths))
	for _, path := range skipPaths {
		skipPathMap[path] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for specified paths
			if skipPathMap[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			logger := logging.Get()

			// Ensure request ID exists
			requestID := r.Header.Get(requestIDHeader)
			if requestID == "" {
				requestID = uuid.New().String()
				r.Header.Set(requestIDHeader, requestID)
			}

			// Get correlation ID or use request ID
			correlationID := r.Header.Get(correlationIDHeader)
			if correlationID == "" {
				correlationID = requestID
			}

			// Add information to context
			ctx := r.Context()
			ctx = logging.WithRequestID(ctx, requestID)
			ctx = logging.WithCorrelationID(ctx, correlationID)
			ctx = logging.WithIPAddress(ctx, r.RemoteAddr)
			ctx = logging.WithUserAgent(ctx, r.UserAgent())
			r = r.WithContext(ctx)

			// Create response wrapper to capture status code and size
			rw := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			// Log request start
			logger.WithContext(ctx).Info("HTTP request started",
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
			)

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Log request completion at appropriate level
			logFn := logger.WithContext(ctx).Info
			if rw.status >= 500 {
				logFn = logger.WithContext(ctx).Error
			} else if rw.status >= 400 {
				logFn = logger.WithContext(ctx).Warn
			}

			logFn("HTTP request completed",
				logging.String("method", r.Method),
				logging.String("path", r.URL.Path),
				logging.Int("status", rw.status),
				logging.Duration("duration", duration),
				logging.Int("response_size", rw.size),
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
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

// UnaryServerLogger returns a gRPC unary server interceptor for logging
func UnaryServerLogger() grpc.UnaryServerInterceptor {
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
		requestID := getFirstMetadataValue(md, requestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
			md = metadata.Join(md, metadata.Pairs(requestIDHeader, requestID))
			ctx = metadata.NewIncomingContext(ctx, md)
		}

		// Get correlation ID or use request ID
		correlationID := getFirstMetadataValue(md, correlationIDHeader)
		if correlationID == "" {
			correlationID = requestID
		}

		// Enhance context
		ctx = logging.WithRequestID(ctx, requestID)
		ctx = logging.WithCorrelationID(ctx, correlationID)

		// Log request start
		logger.WithContext(ctx).Info("gRPC request started",
			logging.String("method", info.FullMethod),
		)

		// Handle the request
		resp, err := handler(ctx, req)

		// Log completion
		duration := time.Since(startTime)
		if err != nil {
			st, _ := status.FromError(err)
			logger.WithContext(ctx).Error("gRPC request failed",
				logging.String("method", info.FullMethod),
				logging.String("code", st.Code().String()),
				logging.String("message", st.Message()),
				logging.Duration("duration", duration),
			)
		} else {
			logger.WithContext(ctx).Info("gRPC request completed",
				logging.String("method", info.FullMethod),
				logging.Duration("duration", duration),
			)
		}

		return resp, err
	}
}

// StreamServerLogger returns a gRPC stream server interceptor for logging
func StreamServerLogger() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		startTime := time.Now()
		logger := logging.Get()
		ctx := ss.Context()

		// Extract metadata
		md, _ := metadata.FromIncomingContext(ctx)

		// Get or generate request ID
		requestID := getFirstMetadataValue(md, requestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
			md = metadata.Join(md, metadata.Pairs(requestIDHeader, requestID))
			ctx = metadata.NewIncomingContext(ctx, md)
		}

		// Get correlation ID or use request ID
		correlationID := getFirstMetadataValue(md, correlationIDHeader)
		if correlationID == "" {
			correlationID = requestID
		}

		// Enhance context
		ctx = logging.WithRequestID(ctx, requestID)
		ctx = logging.WithCorrelationID(ctx, correlationID)

		// Create wrapped stream with enhanced context
		wrappedStream := &serverStreamWithContext{
			ServerStream: ss,
			ctx:          ctx,
		}

		// Log stream start
		logger.WithContext(ctx).Info("gRPC stream started",
			logging.String("method", info.FullMethod),
		)

		// Handle the stream
		err := handler(srv, wrappedStream)

		// Log completion
		duration := time.Since(startTime)
		if err != nil {
			st, _ := status.FromError(err)
			logger.WithContext(ctx).Error("gRPC stream failed",
				logging.String("method", info.FullMethod),
				logging.String("code", st.Code().String()),
				logging.Duration("duration", duration),
			)
		} else {
			logger.WithContext(ctx).Info("gRPC stream completed",
				logging.String("method", info.FullMethod),
				logging.Duration("duration", duration),
			)
		}

		return err
	}
}

// Helper to get first value from metadata
func getFirstMetadataValue(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) > 0 {
		return values[0]
	}
	return ""
}

// serverStreamWithContext wraps grpc.ServerStream to provide custom context
type serverStreamWithContext struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *serverStreamWithContext) Context() context.Context {
	return w.ctx
}
