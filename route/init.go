package route

import (
	"log/slog"
	"net/http"

	"citadel/internal/auth"
	"citadel/internal/logging"
	"citadel/internal/middleware"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Db          *sqlx.DB
	Redis       *redis.Client
	Issuer      *auth.Issuer
	Logger      *slog.Logger
	LogManager  *logging.Manager
	Broadcaster *logging.Broadcaster
}

func Initialize(config Config) http.Handler {
	// Base chain for all routes
	baseChain := middleware.New(
		middleware.CORS,
		middleware.RequestLogger(config.Logger),
	)

	// Protected chain extends base with auth
	protectedChain := baseChain.Use(middleware.RequireAuth(config.Issuer, config.Redis))

	mux := http.NewServeMux()

	// Handle OPTIONS preflight for all routes
	mux.Handle("OPTIONS /", baseChain.ThenFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Public routes - use base chain
	mux.Handle("GET /health", baseChain.ThenFunc(GetHealth()))
	mux.Handle(
		"POST /register",
		baseChain.ThenFunc(Register(config.Db, config.Redis, config.Issuer)),
	)
	mux.Handle("POST /login", baseChain.ThenFunc(Login(config.Db, config.Redis, config.Issuer)))
	mux.Handle(
		"POST /refresh",
		baseChain.ThenFunc(RefreshToken(config.Db, config.Redis, config.Issuer)),
	)

	// Protected routes - use protected chain
	mux.Handle("GET /me", protectedChain.ThenFunc(GetMe()))
	mux.Handle("POST /logout", protectedChain.ThenFunc(Logout(config.Redis)))
	mux.Handle("GET /users", protectedChain.ThenFunc(ListUsers(config.Db)))
	mux.Handle("PATCH /users/{id}", protectedChain.ThenFunc(UpdateUser(config.Db)))

	// SSE log streaming - protected route
	mux.Handle("GET /logs/stream", protectedChain.ThenFunc(
		LogsStream(config.LogManager, config.Broadcaster),
	))

	return mux
}
