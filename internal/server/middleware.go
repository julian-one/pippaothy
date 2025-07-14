package server

import (
	"context"
	"net/http"
	"pippaothy/internal/auth"
	"pippaothy/internal/users"
	"time"
)

type contextKey string

const (
	userContextKey         contextKey = "authenticatedUser"
	flashMessageContextKey contextKey = "flashMessage"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (s *Server) withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log the request
		s.logger.Info("request started",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		// Create a response writer wrapper to capture status code
		wrapper := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next(wrapper, r)

		// Log the response
		duration := time.Since(start)
		s.logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapper.statusCode,
			"duration", duration.String(),
			"duration_ms", duration.Milliseconds(),
		)
	}
}

func (s *Server) getCtxUser(r *http.Request) *users.User {
	user, _ := r.Context().Value(userContextKey).(*users.User)
	return user
}

func (s *Server) getFlashMessage(r *http.Request) string {
	if msg, ok := r.Context().Value(flashMessageContextKey).(string); ok {
		return msg
	}
	return ""
}

func (s *Server) setFlashMessage(token, message string) {
	if err := auth.SetFlashMessage(s.db, token, message); err != nil {
		s.logger.Error("failed to set flash message", "error", err)
	}
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user *users.User
		var flashMessage string

		if cookie, err := r.Cookie("session_token"); err == nil {
			user, _ = auth.GetSession(s.db, cookie.Value)
			flashMessage, _ = auth.GetAndClearFlashMessage(s.db, cookie.Value)
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, flashMessageContextKey, flashMessage)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user *users.User
		var flashMessage string

		// Check for session cookie
		if cookie, err := r.Cookie("session_token"); err == nil {
			user, _ = auth.GetSession(s.db, cookie.Value)
			flashMessage, _ = auth.GetAndClearFlashMessage(s.db, cookie.Value)
		}

		// Redirect if not authenticated
		if user == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user to context and continue
		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, flashMessageContextKey, flashMessage)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
