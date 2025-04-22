package ratelimit

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/mohamedfawas/qubool-kallyanam/pkg/auth/jwt"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// Limiter handles rate limiting logic
type Limiter struct {
	limiter   *redis_rate.Limiter
	keyPrefix string
	logger    logging.Logger
}

// Options configures the rate limiter
type Options struct {
	Limit     int           // Max requests per period
	Period    time.Duration // Time window for rate limiting
	KeyPrefix string        // Prefix for Redis keys
}

// DefaultOptions returns sensible default options
func DefaultOptions() Options {
	return Options{
		Limit:     100,
		Period:    time.Minute,
		KeyPrefix: "ratelimit:",
	}
}

// New creates a new Redis-based rate limiter
func New(redisClient *redis.Client, opts Options, logger logging.Logger) *Limiter {
	if logger == nil {
		logger = logging.Get().Named("ratelimit")
	}

	if opts.Limit <= 0 || opts.Period <= 0 {
		opts = DefaultOptions()
	}

	return &Limiter{
		limiter:   redis_rate.NewLimiter(redisClient),
		keyPrefix: opts.KeyPrefix,
		logger:    logger,
	}
}

// Allow checks if a request is allowed based on the key
func (l *Limiter) Allow(ctx context.Context, key string, limit redis_rate.Limit) (*redis_rate.Result, error) {
	fullKey := l.keyPrefix + key
	return l.limiter.Allow(ctx, fullKey, limit)
}

// Reset clears rate limit data for a specific key
func (l *Limiter) Reset(ctx context.Context, key string) error {
	fullKey := l.keyPrefix + key
	return l.limiter.Reset(ctx, fullKey)
}

// Strategy defines how to generate keys for rate limiting
type Strategy func(ctx context.Context) (string, error)

// HTTP middleware for rate limiting
func (l *Limiter) HTTPMiddleware(strategy Strategy, limit redis_rate.Limit) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			key, err := strategy(ctx)
			if err != nil {
				l.logger.Warn("Error generating rate limit key", logging.Error(err))
				next.ServeHTTP(w, r)
				return
			}

			res, err := l.Allow(ctx, key, limit)
			if err != nil {
				l.logger.Error("Rate limiting error", logging.Error(err))
				next.ServeHTTP(w, r)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", res.Limit.Rate))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", res.Remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", res.ResetAfter/time.Second))

			if res.Allowed == 0 {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", res.RetryAfter/time.Second))
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)

				l.logger.Warn("Rate limit exceeded",
					logging.String("key", key),
					logging.String("path", r.URL.Path),
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UnaryServerInterceptor returns a gRPC interceptor for rate limiting
func (l *Limiter) UnaryServerInterceptor(strategy Strategy, limit redis_rate.Limit) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		key, err := strategy(ctx)
		if err != nil {
			l.logger.Warn("Error generating rate limit key", logging.Error(err))
			return handler(ctx, req)
		}

		res, err := l.Allow(ctx, key, limit)
		if err != nil {
			l.logger.Error("Rate limiting error", logging.Error(err))
			return handler(ctx, req)
		}

		if res.Allowed == 0 {
			l.logger.Warn("Rate limit exceeded",
				logging.String("key", key),
				logging.String("method", info.FullMethod),
			)

			return nil, status.Errorf(
				codes.ResourceExhausted,
				"Rate limit exceeded. Try again in %d seconds",
				int(res.RetryAfter/time.Second),
			)
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream interceptor for rate limiting
func (l *Limiter) StreamServerInterceptor(strategy Strategy, limit redis_rate.Limit) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := stream.Context()
		key, err := strategy(ctx)
		if err != nil {
			l.logger.Warn("Error generating rate limit key", logging.Error(err))
			return handler(srv, stream)
		}

		res, err := l.Allow(ctx, key, limit)
		if err != nil {
			l.logger.Error("Rate limiting error", logging.Error(err))
			return handler(srv, stream)
		}

		if res.Allowed == 0 {
			l.logger.Warn("Rate limit exceeded",
				logging.String("key", key),
				logging.String("method", info.FullMethod),
			)

			return status.Errorf(
				codes.ResourceExhausted,
				"Rate limit exceeded. Try again in %d seconds",
				int(res.RetryAfter/time.Second),
			)
		}

		return handler(srv, stream)
	}
}

// Common strategies

// IPStrategy returns a rate limiting strategy based on client IP
func IPStrategy() Strategy {
	return func(ctx context.Context) (string, error) {
		// For HTTP requests
		if r, ok := ctx.Value(HTTPRequestCtxKey{}).(*http.Request); ok {
			return "ip:" + extractIP(r), nil
		}

		// For gRPC
		return "ip:" + extractGRPCIP(ctx), nil
	}
}

// UserStrategy returns a rate limiting strategy based on user ID
func UserStrategy() Strategy {
	return func(ctx context.Context) (string, error) {
		claims, ok := jwt.ClaimsFromContext(ctx)
		if !ok || claims == nil {
			// Fall back to IP if no user ID is available
			if r, ok := ctx.Value(HTTPRequestCtxKey{}).(*http.Request); ok {
				return "ip:" + extractIP(r), nil
			}
			return "ip:" + extractGRPCIP(ctx), nil
		}

		return "user:" + claims.UserID, nil
	}
}

// Helper functions

// HTTPRequestCtxKey is the key used to store HTTP request in context
type HTTPRequestCtxKey struct{}

// extractIP extracts client IP from HTTP request
func extractIP(r *http.Request) string {
	// Try X-Forwarded-For
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	// Try X-Real-IP
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// extractGRPCIP extracts client IP from gRPC context
func extractGRPCIP(ctx context.Context) string {
	// Try to get peer info
	p, ok := peer.FromContext(ctx)
	if ok {
		ip, _, err := net.SplitHostPort(p.Addr.String())
		if err != nil {
			return p.Addr.String()
		}
		return ip
	}

	// Try to get from metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if values := md.Get("x-forwarded-for"); len(values) > 0 {
			ips := strings.Split(values[0], ",")
			return strings.TrimSpace(ips[0])
		}

		if values := md.Get("x-real-ip"); len(values) > 0 {
			return values[0]
		}
	}

	return "unknown"
}
