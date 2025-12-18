package route

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"pippaothy/internal/auth"
	"pippaothy/internal/middleware"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: message})
}

type Config struct {
	Db     *sqlx.DB
	Redis  *redis.Client
	Issuer *auth.Issuer
	Logger *slog.Logger
}

func Initialize(config Config) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", GetHealth())
	mux.HandleFunc("POST /register", Register(config.Db, config.Redis, config.Issuer, config.Logger))
	mux.HandleFunc("POST /login", Login(config.Db, config.Redis, config.Issuer, config.Logger))
	mux.HandleFunc(
		"POST /refresh",
		RefreshToken(config.Db, config.Redis, config.Issuer, config.Logger),
	)

	// Protected routes
	mux.Handle("GET /me", middleware.RequireAuth(config.Issuer, config.Redis, GetMe()))
	mux.Handle(
		"POST /logout",
		middleware.RequireAuth(config.Issuer, config.Redis, Logout(config.Redis, config.Logger)),
	)
	mux.Handle(
		"GET /users",
		middleware.RequireAuth(config.Issuer, config.Redis, ListUsers(config.Db, config.Logger)),
	)
	mux.Handle(
		"PATCH /users/{id}",
		middleware.RequireAuth(config.Issuer, config.Redis, UpdateUser(config.Db, config.Logger)),
	)

	return middleware.CORS(mux)
}
