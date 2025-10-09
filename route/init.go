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

	// Static files
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Health
	mux.HandleFunc("GET /health", GetHealth())

	// Pages with optional auth (shows different content based on auth status)
	mux.HandleFunc("GET /", middleware.WithAuth(db)(GetHome()))
	mux.HandleFunc("GET /about", middleware.WithAuth(db)(GetAbout()))
	mux.HandleFunc("GET /register", middleware.WithAuth(db)(GetRegister()))
	mux.HandleFunc("GET /login", middleware.WithAuth(db)(GetLogin()))

	// Auth endpoints
	mux.HandleFunc("POST /register", PostRegister(db, logger))
	mux.HandleFunc("POST /login", PostLogin(db, logger))
	mux.HandleFunc("POST /logout", middleware.RequireAuth(db)(PostLogout(db, logger)))

	// Password reset endpoints
	mux.HandleFunc("GET /forgot-password", GetForgotPassword())
	mux.HandleFunc("POST /forgot-password", PostForgotPassword(db, logger))
	mux.HandleFunc("GET /reset-password", GetResetPassword(db, logger))
	mux.HandleFunc("POST /reset-password", PostResetPassword(db, logger))

	// Recipes (all require auth)
	mux.HandleFunc("GET /recipes", middleware.RequireAuth(db)(GetRecipes(db, logger)))
	mux.HandleFunc("GET /recipes/new", middleware.RequireAuth(db)(GetNewRecipe()))
	mux.HandleFunc("POST /recipes", middleware.RequireAuth(db)(PostRecipe(db, logger)))
	mux.HandleFunc("GET /recipes/{id}", middleware.RequireAuth(db)(GetRecipe(db, logger)))
	mux.HandleFunc("GET /recipes/{id}/edit", middleware.RequireAuth(db)(GetEditRecipe(db, logger)))
	mux.HandleFunc("PUT /recipes/{id}", middleware.RequireAuth(db)(PutRecipe(db, logger)))
	mux.HandleFunc("DELETE /recipes/{id}", middleware.RequireAuth(db)(DeleteRecipe(db, logger)))

	return mux
}