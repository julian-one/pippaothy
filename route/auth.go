package route

import (
	"fmt"
	"log/slog"
	"net/http"

	"pippaothy/internal/auth"
	"pippaothy/internal/templates"
	"pippaothy/internal/user"

	"github.com/jmoiron/sqlx"
)

// GetLogin returns a handler for the login page
func GetLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.Login(), "login", false).Render(r.Context(), w)
	}
}

// PostLogin returns a handler for processing login
func PostLogin(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			logger.Error("failed to parse login form", "error", err)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="error">Invalid form data</div>`))
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		// Validate email format
		if err := user.ValidateEmail(email); err != nil {
			logger.Warn("login failed - invalid email format", "email", email)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="error">Invalid email format</div>`))
			return
		}

		// Basic password validation (not as strict as registration)
		if password == "" {
			logger.Warn("login failed - empty password", "email", email)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="error">Password is required</div>`))
			return
		}

		// Get user by email
		u, err := user.ByEmail(r.Context(), db, email)
		if err != nil {
			logger.Warn("login failed - user not found", "email", email)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`<div class="error">Invalid credentials</div>`))
			return
		}

		// Verify password
		if !user.VerifyPassword(u, password) {
			logger.Warn("login failed - invalid password", "email", email)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`<div class="error">Invalid credentials</div>`))
			return
		}

		// Create session
		token, err := auth.CreateSession(r.Context(), db, u.UserId)
		if err != nil {
			logger.Error("failed to create session", "error", err)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`<div class="error">Failed to create session</div>`))
			return
		}

		auth.SetCookie(w, token)

		// Set flash message
		if err := auth.SetFlashMessage(r.Context(), db, token, "Login successful!"); err != nil {
			logger.Error("failed to set flash message", "error", err)
		}

		logger.Info("user logged in successfully", "uid", u.UserId)

		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
	}
}

// PostLogout returns a handler for processing logout
func PostLogout(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get session cookie and destroy session
		if cookie, err := r.Cookie("session_token"); err == nil {
			if err := auth.DestroySession(r.Context(), db, cookie.Value); err != nil {
				logger.Error("failed to destroy session", "error", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

		auth.ResetCookie(w)
		logger.Info("user logged out successfully")

		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
	}
}

// GetRegister returns a handler for the registration page
func GetRegister() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.Register(), "register", false).Render(r.Context(), w)
	}
}

// PostRegister returns a handler for processing registration
func PostRegister(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logger.Error("failed to parse registration form", "error", err)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="error">Invalid form data</div>`))
			return
		}

		request := user.CreateRequest{
			FirstName: r.FormValue("first_name"),
			LastName:  r.FormValue("last_name"),
			Email:     r.FormValue("email"),
			Password:  r.FormValue("password"),
		}

		// Sanitize input
		request.Sanitize()

		// Validate input
		if err := request.Validate(); err != nil {
			logger.Warn("registration validation failed", "error", err)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<div class="error">%s</div>`, err.Error())
			return
		}

		// Check if user already exists
		if user.Exists(r.Context(), db, request.Email) {
			logger.Warn("registration attempt with existing email", "email", request.Email)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(`<div class="error">Email already registered</div>`))
			return
		}

		// Create user
		uid, err := user.Create(r.Context(), db, request)
		if err != nil {
			logger.Error("failed to create user", "error", err)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`<div class="error">Failed to create user</div>`))
			return
		}

		// Create session
		token, err := auth.CreateSession(r.Context(), db, uid)
		if err != nil {
			logger.Error("failed to create session", "error", err)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`<div class="error">Failed to create session</div>`))
			return
		}

		auth.SetCookie(w, token)

		// Set flash message
		if err := auth.SetFlashMessage(r.Context(), db, token, "Registration successful!"); err != nil {
			logger.Error("failed to set flash message", "error", err)
		}

		logger.Info("user registered successfully", "uid", uid)

		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
	}
}