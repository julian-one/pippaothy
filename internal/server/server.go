package server

import (
	"context"
	"log/slog"
	"net/http"
	"pippaothy/internal/ipinfo"

	"github.com/jmoiron/sqlx"
)

type Server struct {
	db      *sqlx.DB
	logger  *slog.Logger
	ipInfo  *ipinfo.Service
	server  *http.Server
}

func New(db *sqlx.DB, logger *slog.Logger) *Server {
	return &Server{
		db:     db,
		logger: logger,
		ipInfo: ipinfo.NewService(),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Health check endpoints
	mux.HandleFunc("GET /health", s.getHealth)
	mux.HandleFunc("GET /ready", s.getReady)

	mux.HandleFunc("GET /", s.withAuth(s.getHome))
	mux.HandleFunc("GET /register", s.withAuth(s.getRegister))
	mux.HandleFunc("POST /register", s.requireCSRF(s.postRegister))
	mux.HandleFunc("GET /login", s.withAuth(s.getLogin))
	mux.HandleFunc("POST /login", s.requireCSRF(s.postLogin))
	mux.HandleFunc("POST /logout", s.requireAuth(s.requireCSRF(s.postLogout)))
	mux.HandleFunc("GET /logs", s.requireAuth(s.getSimpleLogs))

	// Wrap the entire mux with logging middleware
	handler := s.withLogging(mux.ServeHTTP)

	// Create the HTTP server
	s.server = &http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Initiating graceful server shutdown")
	return s.server.Shutdown(ctx)
}
