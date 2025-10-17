package middleware

import (
	"context"
	"net/http"

	"pippaothy/internal/auth"
	ctxkeys "pippaothy/internal/context"
	"pippaothy/internal/users"

	"github.com/jmoiron/sqlx"
)

// WithAuth adds the authenticated user and flash message to the request context
func WithAuth(db *sqlx.DB) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var user *users.User
			var flashMessage string

			if cookie, err := r.Cookie("session_token"); err == nil {
				user, _ = auth.GetSession(r.Context(), db, cookie.Value)
				flashMessage, _ = auth.GetAndClearFlashMessage(r.Context(), db, cookie.Value)
			}

			ctx := context.WithValue(r.Context(), ctxkeys.UserContextKey, user)
			ctx = context.WithValue(ctx, ctxkeys.FlashMessageContextKey, flashMessage)
			next(w, r.WithContext(ctx))
		}
	}
}

// RequireAuth ensures the user is authenticated, redirecting to login if not
func RequireAuth(db *sqlx.DB) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var user *users.User
			var flashMessage string

			// Check for session cookie
			if cookie, err := r.Cookie("session_token"); err == nil {
				user, _ = auth.GetSession(r.Context(), db, cookie.Value)
				flashMessage, _ = auth.GetAndClearFlashMessage(r.Context(), db, cookie.Value)
			}

			// Redirect if not authenticated
			if user == nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Add user to context and continue
			ctx := context.WithValue(r.Context(), ctxkeys.UserContextKey, user)
			ctx = context.WithValue(ctx, ctxkeys.FlashMessageContextKey, flashMessage)
			next(w, r.WithContext(ctx))
		}
	}
}