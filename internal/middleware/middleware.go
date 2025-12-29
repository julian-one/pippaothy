package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"citadel/internal/auth"
	"citadel/internal/cache"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type contextKey string

const (
	ClaimsKey    contextKey = "claims"
	RequestIdKey contextKey = "requestId"
	LoggerKey    contextKey = "logger"
)

// statusRecorder wraps http.ResponseWriter to capture the status code.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher for SSE support.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// RequestLogger returns logging middleware that logs each incoming request.
// It also stores a child logger with request_id in context for use by handlers.
func RequestLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			requestID := uuid.New().String()

			// Create child logger with request ID attached
			reqLogger := logger.With("request_id", requestID)

			// Store request ID and logger in context
			ctx := context.WithValue(r.Context(), RequestIdKey, requestID)
			ctx = context.WithValue(ctx, LoggerKey, reqLogger)
			r = r.WithContext(ctx)

			// Add request ID to response header
			w.Header().Set("X-Request-ID", requestID)

			// Wrap response writer to capture status
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			// Log request start
			reqLogger.Info("request started",
				"method", r.Method,
				"path", r.URL.Path,
				"client_ip", getClientIP(r),
			)

			next.ServeHTTP(rec, r)

			// Log request completion
			reqLogger.Info("request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"client_ip", getClientIP(r),
			)
		})
	}
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// GetRequestID extracts the request ID from the context.
func GetRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(RequestIdKey).(string); ok {
		return id
	}
	return ""
}

// GetLogger extracts the request-scoped logger from the context.
// The returned logger has request_id already attached.
// Falls back to a no-op logger if not found (shouldn't happen if middleware is used).
func GetLogger(r *http.Request) *slog.Logger {
	if logger, ok := r.Context().Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// RequireAuth returns authentication middleware that validates JWT tokens.
func RequireAuth(issuer *auth.Issuer, client *redis.Client) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			var token string

			// Try Authorization header first
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					token = parts[1]
				}
			}

			// Fall back to query param (for SSE which doesn't support headers)
			if token == "" {
				token = r.URL.Query().Get("token")
			}

			if token == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Missing authentication token"})
				return
			}

			claims, err := issuer.Validate(token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired token"})
				return
			}

			isBlacklisted, err := cache.IsBlacklisted(ctx, client, claims.ID)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication service unavailable"})
				return
			}
			if isBlacklisted {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Token has been revoked"})
				return
			}

			ctx = context.WithValue(ctx, ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
