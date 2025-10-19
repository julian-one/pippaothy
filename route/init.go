package route

import (
	"log/slog"
	"net/http"

	"pippaothy/internal/middleware"

	"github.com/jmoiron/sqlx"
)

// Initialize sets up all routes and returns the configured mux
func Initialize(db *sqlx.DB, logger *slog.Logger) *http.ServeMux {
	mux := http.NewServeMux()

	// Initialize auth middleware
	auth := middleware.NewAuth(db, logger)

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Health
	mux.HandleFunc("GET /health", GetHealth())

	// Pages with optional auth (shows different content based on auth status)
	mux.HandleFunc("GET /", auth.WithAuth(GetHome()))
	mux.HandleFunc("GET /about", auth.WithAuth(GetAbout()))
	mux.HandleFunc("GET /register", auth.WithAuth(GetRegister()))
	mux.HandleFunc("GET /login", auth.WithAuth(GetLogin()))

	// Auth endpoints
	mux.HandleFunc("POST /register", PostRegister(db, logger))
	mux.HandleFunc("POST /login", PostLogin(db, logger))
	mux.HandleFunc("POST /logout", auth.RequireAuth(PostLogout(db, logger)))

	// Password reset endpoints
	mux.HandleFunc("GET /forgot-password", GetForgotPassword())
	mux.HandleFunc("POST /forgot-password", PostForgotPassword(db, logger))
	mux.HandleFunc("GET /reset-password", GetResetPassword(db, logger))
	mux.HandleFunc("POST /reset-password", PostResetPassword(db, logger))

	// Recipes (all require auth)
	mux.HandleFunc("GET /recipes", auth.RequireAuth(GetRecipes(db, logger)))
	mux.HandleFunc("GET /recipes/new", auth.RequireAuth(GetNewRecipe()))
	mux.HandleFunc("POST /recipes", auth.RequireAuth(PostRecipe(db, logger)))
	mux.HandleFunc("GET /recipes/{id}", auth.RequireAuth(GetRecipe(db, logger)))
	mux.HandleFunc("GET /recipes/{id}/edit", auth.RequireAuth(GetEditRecipe(db, logger)))
	mux.HandleFunc("PUT /recipes/{id}", auth.RequireAuth(PutRecipe(db, logger)))
	mux.HandleFunc("DELETE /recipes/{id}", auth.RequireAuth(DeleteRecipe(db, logger)))

	return mux
}