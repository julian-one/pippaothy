package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"pippaothy/internal/auth"
	"pippaothy/internal/email"
	"pippaothy/internal/logs"
	"pippaothy/internal/templates"
	"pippaothy/internal/users"
)

func (s *Server) getHome(w http.ResponseWriter, r *http.Request) {
	u := s.getCtxUser(r)
	flashMessage := s.getFlashMessage(r)

	userName := ""
	loggedIn := u != nil
	if u != nil {
		userName = u.FirstName
	}

	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.Home(userName, flashMessage), "home", loggedIn).
		Render(r.Context(), w)
}

func (s *Server) getAbout(w http.ResponseWriter, r *http.Request) {
	u := s.getCtxUser(r)
	loggedIn := u != nil

	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.AboutPage(), "About - Julian Roberts", loggedIn).
		Render(r.Context(), w)
}

func (s *Server) getRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.Register(), "register", false).Render(r.Context(), w)
}

func (s *Server) postRegister(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.logger.Error("failed to parse registration form", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Invalid form data</div>`))
		return
	}

	request := users.CreateRequest{
		FirstName: r.FormValue("first_name"),
		LastName:  r.FormValue("last_name"),
		Email:     r.FormValue("email"),
		Password:  r.FormValue("password"),
	}

	// Sanitize input
	request.Sanitize()

	// Validate input
	if err := request.Validate(); err != nil {
		s.logger.Warn("registration validation failed", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="error">%s</div>`, err.Error())
		return
	}

	// Check if user already exists
	if users.Exists(r.Context(), s.db, request.Email) {
		s.logger.Warn("registration attempt with existing email", "email", request.Email)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`<div class="error">Email already registered</div>`))
		return
	}

	// Create user
	uid, err := users.Create(r.Context(), s.db, request)
	if err != nil {
		s.logger.Error("failed to create user", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="error">Failed to create user</div>`))
		return
	}

	// Create session
	token, err := auth.CreateSession(r.Context(), s.db, uid)
	if err != nil {
		s.logger.Error("failed to create session", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="error">Failed to create session</div>`))
		return
	}

	auth.SetCookie(w, token)
	s.setFlashMessage(r.Context(), token, "Registration successful!")
	s.logger.Info("user registered successfully", "uid", uid)

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getLogin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.Login(), "login", false).Render(r.Context(), w)
}

func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.logger.Error("failed to parse login form", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Invalid form data</div>`))
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Validate email format
	if err := users.ValidateEmail(email); err != nil {
		s.logger.Warn("login failed - invalid email format", "email", email)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Invalid email format</div>`))
		return
	}

	// Basic password validation (not as strict as registration)
	if password == "" {
		s.logger.Warn("login failed - empty password", "email", email)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Password is required</div>`))
		return
	}

	// Get user by email
	user, err := users.ByEmail(r.Context(), s.db, email)
	if err != nil {
		s.logger.Warn("login failed - user not found", "email", email)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<div class="error">Invalid credentials</div>`))
		return
	}

	// Verify password
	if !users.VerifyPassword(user, password) {
		s.logger.Warn("login failed - invalid password", "email", email)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`<div class="error">Invalid credentials</div>`))
		return
	}

	// Create session
	token, err := auth.CreateSession(r.Context(), s.db, user.UserId)
	if err != nil {
		s.logger.Error("failed to create session", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="error">Failed to create session</div>`))
		return
	}

	auth.SetCookie(w, token)
	s.setFlashMessage(r.Context(), token, "Login successful!")
	s.logger.Info("user logged in successfully", "uid", user.UserId)

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie and destroy session
	if cookie, err := r.Cookie("session_token"); err == nil {
		if err := auth.DestroySession(r.Context(), s.db, cookie.Value); err != nil {
			s.logger.Error("failed to destroy session", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	auth.ResetCookie(w)
	s.logger.Info("user logged out successfully")

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getSimpleLogs(w http.ResponseWriter, r *http.Request) {
	query := logs.ParseQuery(r)
	result := logs.GetLogs(query)

	w.Header().Set("Content-Type", "text/html")

	if r.Header.Get("HX-Request") == "true" {
		templates.SimpleLogEntries(result).Render(r.Context(), w)
	} else {
		user := s.getCtxUser(r)
		loggedIn := user != nil
		templates.Layout(templates.SimpleLogs(result, query), "logs", loggedIn).Render(r.Context(), w)
	}
}

// Health check endpoints
func (s *Server) getForgotPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.ForgotPassword(), "forgot-password", false).Render(r.Context(), w)
}

func (s *Server) postForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.logger.Error("failed to parse forgot password form", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Invalid form data</div>`))
		return
	}

	honeypot := r.FormValue("website")
	if honeypot != "" {
		s.logger.Warn("honeypot triggered", "value", honeypot)
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(
			[]byte(
				`<div class="success">If an account with that email exists, you will receive a password reset link shortly.</div>`,
			),
		)
		return
	}

	renderTimeStr := r.FormValue("render_time")
	if renderTimeStr != "" {
		decoded, err := base64.StdEncoding.DecodeString(renderTimeStr)
		if err == nil {
			renderTimeUnix, err := strconv.ParseInt(string(decoded), 10, 64)
			if err == nil {
				renderTime := time.Unix(renderTimeUnix, 0)
				elapsed := time.Since(renderTime)
				if elapsed < 2*time.Second {
					s.logger.Warn("form submitted too quickly", "elapsed", elapsed)
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK)
					w.Write(
						[]byte(
							`<div class="success">If an account with that email exists, you will receive a password reset link shortly.</div>`,
						),
					)
					return
				}
				if elapsed > 30*time.Minute {
					s.logger.Warn("form submission too old", "elapsed", elapsed)
					w.Header().Set("Content-Type", "text/html")
					w.WriteHeader(http.StatusOK)
					w.Write(
						[]byte(
							`<div class="error">Form expired. Please refresh the page and try again.</div>`,
						),
					)
					return
				}
			}
		}
	}

	emailAddr := strings.TrimSpace(strings.ToLower(r.FormValue("email")))

	if err := users.ValidateEmail(emailAddr); err != nil {
		s.logger.Warn("forgot password failed - invalid email format", "email", emailAddr)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(
			[]byte(
				`<div class="success">If an account with that email exists, you will receive a password reset link shortly.</div>`,
			),
		)
		return
	}

	clientIP := getClientIP(r)

	if err := s.rateLimiter.CheckPasswordReset(r.Context(), emailAddr, clientIP); err != nil {
		s.logger.Warn("rate limit exceeded", "email", emailAddr, "ip", clientIP, "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write(
			[]byte(
				`<div class="error">Too many password reset attempts. Please try again later.</div>`,
			),
		)
		return
	}

	if err := s.rateLimiter.RecordPasswordResetAttempt(r.Context(), emailAddr, clientIP); err != nil {
		s.logger.Error("failed to record password reset attempt", "error", err)
	}

	token, err := users.CreatePasswordReset(r.Context(), s.db, emailAddr)
	if err != nil {
		s.logger.Error("failed to create password reset", "error", err, "email", emailAddr)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(
			[]byte(
				`<div class="success">If an account with that email exists, you will receive a password reset link shortly.</div>`,
			),
		)
		return
	}

	if token != "" {
		emailService := email.NewEmailService()
		if emailService != nil {
			baseURL := "http://localhost:8080"
			if os.Getenv("GO_ENV") == "production" || os.Getenv("GO_ENV") == "prod" {
				baseURL = "https://pippaothy.com"
			}

			if err := emailService.SendPasswordResetEmail(emailAddr, token, baseURL); err != nil {
				s.logger.Error(
					"failed to send password reset email",
					"error",
					err,
					"email",
					emailAddr,
				)
			} else {
				s.logger.Info("password reset email sent", "email", emailAddr)
			}
		} else {
			s.logger.Warn("email service not configured - password reset request ignored", "email", emailAddr)
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(
		[]byte(
			`<div class="success">If an account with that email exists, you will receive a password reset link shortly.</div>`,
		),
	)
}

func (s *Server) getResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		s.logger.Warn("reset password page accessed without token")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Invalid reset link</div>`))
		return
	}

	_, err := users.ValidateResetToken(r.Context(), s.db, token)
	if err != nil {
		s.logger.Warn("invalid reset token accessed", "token", token, "error", err)
		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.ResetPassword(), "reset-password", false).Render(r.Context(), w)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	component := templates.ResetPassword()
	// We need to inject the token into the hidden field via JavaScript
	response := fmt.Sprintf(`
		<script>
			document.addEventListener('DOMContentLoaded', function() {
				const tokenField = document.getElementById('token');
				if (tokenField) {
					tokenField.value = '%s';
				}
			});
		</script>
	`, token)

	templates.Layout(component, "reset-password", false).Render(r.Context(), w)
	w.Write([]byte(response))
}

func (s *Server) postResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.logger.Error("failed to parse reset password form", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Invalid form data</div>`))
		return
	}

	token := r.FormValue("token")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	if token == "" {
		s.logger.Warn("reset password attempt without token")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Invalid reset token</div>`))
		return
	}

	if password != confirmPassword {
		s.logger.Warn("reset password attempt with mismatched passwords")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`<div class="error">Passwords do not match</div>`))
		return
	}

	err := users.ResetPassword(r.Context(), s.db, token, password)
	if err != nil {
		s.logger.Warn("password reset failed", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `<div class="error">%s</div>`, err.Error())
		return
	}

	s.logger.Info("password reset successful")
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(
		[]byte(
			`<div class="success">Password reset successfully! <a href="/login" class="link">Click here to sign in</a></div>`,
		),
	)
}

func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	// Basic health check - just return 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   fmt.Sprintf("%d", time.Now().Unix()),
	})
}
