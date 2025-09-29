package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"pippaothy/internal/auth"
	"pippaothy/internal/users"
)

type contextKey string

const (
	userContextKey         contextKey = "authenticatedUser"
	flashMessageContextKey contextKey = "flashMessage"
	requestIDContextKey    contextKey = "requestID"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.responseSize += int64(size)
	return size, err
}

// generateRequestID creates a unique request ID for tracing
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// getClientIP extracts the real client IP from various headers
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (handles proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		if parts := strings.Split(xff, ","); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fallback to RemoteAddr, stripping port if present
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// isStaticAsset checks if the request is for a static asset
func isStaticAsset(path string) bool {
	staticPrefixes := []string{"/static/", "/favicon.ico", "/robots.txt"}
	for _, prefix := range staticPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// getStatusCategory returns a human-readable status category
func getStatusCategory(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "success"
	case code >= 300 && code < 400:
		return "redirect"
	case code >= 400 && code < 500:
		return "client_error"
	case code >= 500:
		return "server_error"
	default:
		return "unknown"
	}
}

func (s *Server) withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		requestID := generateRequestID()

		// Add request ID to context for downstream handlers
		ctx := context.WithValue(r.Context(), requestIDContextKey, requestID)
		r = r.WithContext(ctx)

		// Extract client information
		clientIP := getClientIP(r)
		userAgent := r.UserAgent()
		referer := r.Referer()
		isStatic := isStaticAsset(r.URL.Path)

		// Get content length for request body size
		contentLength := max(r.ContentLength, 0)

		// Build base log fields
		baseFields := []any{
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"client_ip", clientIP,
			"user_agent", userAgent,
			"referer", referer,
			"request_size", contentLength,
			"is_static", isStatic,
			"protocol", r.Proto,
			"host", r.Host,
		}

		// Create enhanced response writer wrapper
		wrapper := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			responseSize:   0,
		}

		// Call the next handler with panic recovery
		func() {
			defer func() {
				if err := recover(); err != nil {
					wrapper.statusCode = http.StatusInternalServerError
					
					// Get user info after auth middleware has run
					if user := s.getCtxUser(r); user != nil {
						baseFields = append(baseFields, "user_id", user.UserId, "username", user.Email)
					}
					
					s.logger.Error("HTTP request panic recovered",
						append(baseFields, "panic", err, "stack_trace", true)...)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next(wrapper, r)
		}()

		// Get user info after the request has been processed
		if user := s.getCtxUser(r); user != nil {
			baseFields = append(baseFields, "user_id", user.UserId, "username", user.Email)
		}

		// Calculate metrics
		duration := time.Since(start)
		statusCategory := getStatusCategory(wrapper.statusCode)

		// Build response log fields
		responseFields := append(baseFields,
			"status", wrapper.statusCode,
			"status_category", statusCategory,
			"response_size", wrapper.responseSize,
			"duration_ms", duration.Milliseconds(),
			"duration_us", duration.Microseconds(),
			"duration", duration.String(),
		)

		// Determine log level and message based on response
		var logLevel slog.Level
		message := "HTTP request completed"

		switch {
		case wrapper.statusCode >= 500:
			logLevel = slog.LevelError
			message = "HTTP request failed with server error"
		case wrapper.statusCode >= 400:
			logLevel = slog.LevelWarn
			message = "HTTP request failed with client error"
		case duration > 5*time.Second:
			logLevel = slog.LevelWarn
			message = "HTTP request completed slowly"
		case isStatic:
			logLevel = slog.LevelDebug
			message = "Static asset served"
		default:
			logLevel = slog.LevelInfo
		}

		// Log based on level
		switch logLevel {
		case slog.LevelError:
			s.logger.Error(message, responseFields...)
		case slog.LevelWarn:
			s.logger.Warn(message, responseFields...)
		case slog.LevelDebug:
			s.logger.Debug(message, responseFields...)
		default:
			s.logger.Info(message, responseFields...)
		}

		// Add response headers for debugging
		w.Header().Set("X-Request-ID", requestID)
		if duration > time.Second {
			w.Header().Set("X-Response-Time", duration.String())
		}
	}
}

func (s *Server) getCtxUser(r *http.Request) *users.User {
	user, _ := r.Context().Value(userContextKey).(*users.User)
	return user
}

func (s *Server) getRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(requestIDContextKey).(string); ok {
		return id
	}
	return "unknown"
}

func (s *Server) getFlashMessage(r *http.Request) string {
	if msg, ok := r.Context().Value(flashMessageContextKey).(string); ok {
		return msg
	}
	return ""
}

func (s *Server) setFlashMessage(ctx context.Context, token, message string) {
	if err := auth.SetFlashMessage(ctx, s.db, token, message); err != nil {
		s.logger.Error("failed to set flash message",
			"error", err,
			"session_token", token[:8]+"...", // Only log partial token for security
		)
	}
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var user *users.User
		var flashMessage string
		requestID := s.getRequestID(r)

		if cookie, err := r.Cookie("session_token"); err == nil {
			var sessionErr error
			user, sessionErr = auth.GetSession(r.Context(), s.db, cookie.Value)
			if sessionErr != nil {
				s.logger.Debug("session lookup failed",
					"request_id", requestID,
					"error", sessionErr,
					"session_token", cookie.Value[:8]+"...",
				)
			} else if user != nil {
				s.logger.Debug("user authenticated from session",
					"request_id", requestID,
					"user_id", user.UserId,
					"username", user.Email,
				)
			}

			flashMessage, _ = auth.GetAndClearFlashMessage(r.Context(), s.db, cookie.Value)
		} else {
			s.logger.Debug("no session cookie found", "request_id", requestID)
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, flashMessageContextKey, flashMessage)
		next(w, r.WithContext(ctx))
	}
}

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication in development mode
		isDevelopment := os.Getenv("GO_ENV") != "production" && os.Getenv("GO_ENV") != "prod"

		if isDevelopment {
			s.logger.Debug("skipping authentication in development mode",
				"path", r.URL.Path,
				"method", r.Method,
			)
			// Create a mock user for development
			mockUser := &users.User{
				UserId: 1,
				Email:  "dev@localhost",
			}
			ctx := context.WithValue(r.Context(), userContextKey, mockUser)
			next(w, r.WithContext(ctx))
			return
		}

		var user *users.User
		var flashMessage string
		requestID := s.getRequestID(r)
		clientIP := getClientIP(r)

		// Check for session cookie
		if cookie, err := r.Cookie("session_token"); err == nil {
			var sessionErr error
			user, sessionErr = auth.GetSession(r.Context(), s.db, cookie.Value)
			if sessionErr != nil {
				s.logger.Warn("authentication failed - invalid session",
					"request_id", requestID,
					"client_ip", clientIP,
					"path", r.URL.Path,
					"error", sessionErr,
					"session_token", cookie.Value[:8]+"...",
				)
			}
			flashMessage, _ = auth.GetAndClearFlashMessage(r.Context(), s.db, cookie.Value)
		}

		// Redirect if not authenticated
		if user == nil {
			s.logger.Info("authentication required - redirecting to login",
				"request_id", requestID,
				"client_ip", clientIP,
				"path", r.URL.Path,
				"method", r.Method,
				"user_agent", r.UserAgent(),
			)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Log successful authentication
		s.logger.Debug("authentication successful",
			"request_id", requestID,
			"user_id", user.UserId,
			"username", user.Email,
			"path", r.URL.Path,
		)

		// Add user to context and continue
		ctx := context.WithValue(r.Context(), userContextKey, user)
		ctx = context.WithValue(ctx, flashMessageContextKey, flashMessage)
		next(w, r.WithContext(ctx))
	}
}
