package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"pippaothy/internal/auth"
	rdb "pippaothy/internal/redis"

	"github.com/redis/go-redis/v9"
)

type contextKey string

const ClaimsKey contextKey = "claims"

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: message})
}

func RequireAuth(
	issuer *auth.Issuer,
	client *redis.Client,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "Invalid Authorization header format")
			return
		}

		claims, err := issuer.Validate(parts[1])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		isBlacklisted, err := rdb.IsBlacklisted(ctx, client, claims.ID)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "Authentication service unavailable")
			return
		}
		if isBlacklisted {
			writeError(w, http.StatusUnauthorized, "Token has been revoked")
			return
		}

		ctx = context.WithValue(ctx, ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
