package server

import (
	"log/slog"
	"net/http"

	"github.com/jmoiron/sqlx"
)

type Server struct {
	db     *sqlx.DB
	logger *slog.Logger
}

func New(db *sqlx.DB, logger *slog.Logger) *Server {
	return &Server{
		db:     db,
		logger: logger,
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("GET /", s.withAuth(s.getHome))
	mux.HandleFunc("GET /register", s.getRegister)
	mux.HandleFunc("POST /register", s.postRegister)
	mux.HandleFunc("GET /login", s.getLogin)
	mux.HandleFunc("POST /login", s.postLogin)
	mux.HandleFunc("POST /logout", s.requireAuth(s.postLogout))

	// Wrap the entire mux with logging middleware
	handler := s.withLogging(mux.ServeHTTP)

	s.logger.Info("Server starting on :8080")
	return http.ListenAndServe(":8080", handler)
}
