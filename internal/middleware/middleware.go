package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"pippaothy/internal/auth"
	"pippaothy/internal/users"

	"github.com/jmoiron/sqlx"
)

// contextKey is a type for context keys used within this package
type contextKey string

const (
	// userContextKey is the key for storing the authenticated user in the request context
	userContextKey contextKey = "authenticatedUser"

	// flashMessageContextKey is the key for storing flash messages in the request context
	flashMessageContextKey contextKey = "flashMessage"
)

// AuthMiddleware holds dependencies for authentication middleware
type AuthMiddleware struct {
	db     *sqlx.DB
	logger *slog.Logger
}

// NewAuth creates a new AuthMiddleware instance
func NewAuth(db *sqlx.DB, logger *slog.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		db:     db,
		logger: logger,
	}
}

// WithAuth adds the authenticated user and flash message to the request context
func (am *AuthMiddleware) WithAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user *users.User
		var flashMessage string

		if cookie, err := r.Cookie("session_token"); err == nil {
			user, err = auth.GetSession(r.Context(), am.db, cookie.Value)
			if err != nil {
				am.logger.Debug("failed to get session", "error", err)
			}

			flashMessage, err = auth.GetAndClearFlashMessage(r.Context(), am.db, cookie.Value)
			if err != nil {
				am.logger.Debug("failed to get flash message", "error", err)
			}
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, flashMessageContextKey, flashMessage)
		next(w, r.WithContext(ctx))
	}
}

// RequireAuth ensures the user is authenticated, redirecting to login if not
func (am *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user *users.User
		var flashMessage string

		// Check for session cookie
		if cookie, err := r.Cookie("session_token"); err == nil {
			user, err = auth.GetSession(r.Context(), am.db, cookie.Value)
			if err != nil {
				am.logger.Debug("failed to get session", "error", err)
			}

			flashMessage, err = auth.GetAndClearFlashMessage(r.Context(), am.db, cookie.Value)
			if err != nil {
				am.logger.Debug("failed to get flash message", "error", err)
			}
		}

		// Redirect if not authenticated
		if user == nil {
			am.logger.Debug("unauthenticated access attempt", "path", r.URL.Path)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user to context and continue
		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, flashMessageContextKey, flashMessage)
		next(w, r.WithContext(ctx))
	}
}

// GetUserFromContext retrieves the authenticated user from the request context
// Returns nil if no user is found or if the type assertion fails
func GetUserFromContext(r *http.Request) *users.User {
	if user, ok := r.Context().Value(userContextKey).(*users.User); ok {
		return user
	}
	return nil
}

// GetFlashMessageFromContext retrieves the flash message from the request context
// Returns an empty string if no message is found or if the type assertion fails
func GetFlashMessageFromContext(r *http.Request) string {
	if msg, ok := r.Context().Value(flashMessageContextKey).(string); ok {
		return msg
	}
	return ""
}