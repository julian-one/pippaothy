package auth

import (
	"context"
	"net/http"
	"strings"

	"pippaothy/internal/redis"
)

type contextKey string

const ClaimsKey contextKey = "claims"

// RequireAuth validates Bearer token from Authorization header per RFC 6750
// and checks if the token has been blacklisted in Redis
func RequireAuth(redisClient *redis.Client, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// RFC 6750 Section 2.1: Authorization: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(
				w,
				"Invalid Authorization header format. Expected: Bearer <token>",
				http.StatusUnauthorized,
			)
			return
		}

		token := parts[1]
		claims, err := ValidateJWT(token)
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Check if token is blacklisted (logged out)
		isBlacklisted, err := redisClient.IsTokenBlacklisted(r.Context(), claims.ID)
		if err != nil {
			// Log error but don't fail the request if Redis is down
			// In production, you might want to fail-closed instead
			http.Error(w, "Authentication service temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		if isBlacklisted {
			http.Error(w, "Token has been revoked", http.StatusUnauthorized)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
