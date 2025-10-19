package route

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"pippaothy/internal/email"
	"pippaothy/internal/templates"
	"pippaothy/internal/user"

	"github.com/jmoiron/sqlx"
)

// GetForgotPassword returns a handler for the forgot password page
func GetForgotPassword() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.ForgotPassword(), "forgot-password", false).Render(r.Context(), w)
	}
}

// PostForgotPassword returns a handler for processing forgot password requests
func PostForgotPassword(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logger.Error("failed to parse forgot password form", "error", err)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="error">Invalid form data</div>`))
			return
		}

		honeypot := r.FormValue("website")
		if honeypot != "" {
			logger.Warn("honeypot triggered", "value", honeypot)
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
						logger.Warn("form submitted too quickly", "elapsed", elapsed)
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
						logger.Warn("form submission too old", "elapsed", elapsed)
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

		if err := user.ValidateEmail(emailAddr); err != nil {
			logger.Warn("forgot password failed - invalid email format", "email", emailAddr)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write(
				[]byte(
					`<div class="success">If an account with that email exists, you will receive a password reset link shortly.</div>`,
				),
			)
			return
		}

		token, err := user.CreatePasswordReset(r.Context(), db, emailAddr)
		if err != nil {
			logger.Error("failed to create password reset", "error", err, "email", emailAddr)
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
					logger.Error(
						"failed to send password reset email",
						"error",
						err,
						"email",
						emailAddr,
					)
				} else {
					logger.Info("password reset email sent", "email", emailAddr)
				}
			} else {
				logger.Warn("email service not configured - password reset request ignored", "email", emailAddr)
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
}

// GetResetPassword returns a handler for the reset password page
func GetResetPassword(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			logger.Warn("reset password page accessed without token")
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="error">Invalid reset link</div>`))
			return
		}

		_, err := user.ValidateResetToken(r.Context(), db, token)
		if err != nil {
			logger.Warn("invalid reset token accessed", "token", token, "error", err)
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
}

// PostResetPassword returns a handler for processing password reset
func PostResetPassword(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			logger.Error("failed to parse reset password form", "error", err)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="error">Invalid form data</div>`))
			return
		}

		token := r.FormValue("token")
		password := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")

		if token == "" {
			logger.Warn("reset password attempt without token")
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="error">Invalid reset token</div>`))
			return
		}

		if password != confirmPassword {
			logger.Warn("reset password attempt with mismatched passwords")
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`<div class="error">Passwords do not match</div>`))
			return
		}

		err := user.ResetPassword(r.Context(), db, token, password)
		if err != nil {
			logger.Warn("password reset failed", "error", err)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<div class="error">%s</div>`, err.Error())
			return
		}

		logger.Info("password reset successful")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(
			[]byte(
				`<div class="success">Password reset successfully! <a href="/login" class="link">Click here to sign in</a></div>`,
			),
		)
	}
}