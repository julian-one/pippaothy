package route

import (
	"log/slog"
	"net/http"

	"pippaothy/internal/auth"
	"pippaothy/internal/redis"

	"github.com/jmoiron/sqlx"
)

// Initialize sets up all routes and returns the configured mux
func Initialize(db *sqlx.DB, redisClient *redis.Client, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", GetHealth())
	mux.HandleFunc("POST /register", Register(db, redisClient, logger))
	mux.HandleFunc("POST /login", Login(db, redisClient, logger))
	mux.HandleFunc("POST /refresh", RefreshTokenHandler(db, redisClient, logger))

	// Protected
	mux.Handle("GET /me", auth.RequireAuth(redisClient, GetMe()))
	mux.Handle("POST /logout", auth.RequireAuth(redisClient, Logout(redisClient, logger)))
	mux.Handle("GET /users", auth.RequireAuth(redisClient, ListUsers(db, logger)))
	mux.Handle("PATCH /users/{id}", auth.RequireAuth(redisClient, UpdateUser(db, logger)))

	return mux
}
