package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jmoiron/sqlx"
)

type Server struct {
	db      *sqlx.DB
	logger  *slog.Logger
	server  *http.Server
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
	
	// Recipe routes
	mux.HandleFunc("GET /recipes", s.requireAuth(s.handleRecipesList))
	mux.HandleFunc("GET /recipes/public", s.withAuth(s.handlePublicRecipes))
	mux.HandleFunc("GET /recipes/search", s.withAuth(s.handleRecipeSearch))
	mux.HandleFunc("GET /recipes/new", s.requireAuth(s.handleRecipeNew))
	mux.HandleFunc("POST /recipes", s.requireAuth(s.handleRecipeCreate))
	mux.HandleFunc("GET /recipes/{id}", s.withAuth(s.handleRecipeDetail))
	mux.HandleFunc("GET /recipes/{id}/edit", s.requireAuth(s.handleRecipeEdit))
	mux.HandleFunc("PUT /recipes/{id}", s.requireAuth(s.handleRecipeUpdate))
	mux.HandleFunc("DELETE /recipes/{id}", s.requireAuth(s.handleRecipeDelete))

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
