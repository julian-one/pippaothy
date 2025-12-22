package route

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"pippaothy/internal/auth"
	"pippaothy/internal/logstream"
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
	Db         *sqlx.DB
	Redis      *redis.Client
	Issuer     *auth.Issuer
	Logger     *slog.Logger
	FileLogger *logstream.FileLogger
}

func Initialize(config Config) http.Handler {
	// Base chain for all routes (without CORS - applied at top level)
	baseChain := middleware.New(
		middleware.RequestLogger(config.Logger),
	)

	// Protected chain extends base with auth
	protectedChain := baseChain.Use(middleware.RequireAuth(config.Issuer, config.Redis))

	mux := http.NewServeMux()

	// Public routes - use base chain
	mux.Handle("GET /health", baseChain.ThenFunc(GetHealth()))
	mux.Handle("POST /register", baseChain.ThenFunc(Register(config.Db, config.Redis, config.Issuer)))
	mux.Handle("POST /login", baseChain.ThenFunc(Login(config.Db, config.Redis, config.Issuer)))
	mux.Handle("POST /refresh", baseChain.ThenFunc(RefreshToken(config.Db, config.Redis, config.Issuer)))

	// Protected routes - use protected chain
	mux.Handle("GET /me", protectedChain.ThenFunc(GetMe()))
	mux.Handle("POST /logout", protectedChain.ThenFunc(Logout(config.Redis)))
	mux.Handle("GET /users", protectedChain.ThenFunc(ListUsers(config.Db)))
	mux.Handle("PATCH /users/{id}", protectedChain.ThenFunc(UpdateUser(config.Db)))
	mux.Handle("GET /logs/stream", protectedChain.ThenFunc(StreamLogs(config.FileLogger)))
	mux.Handle("GET /logs/history", protectedChain.ThenFunc(GetLogHistory(config.FileLogger)))

	// Wrap entire mux with CORS to handle OPTIONS preflight
	// before Go's method-based routing returns 405
	return middleware.CORS(mux)
}
