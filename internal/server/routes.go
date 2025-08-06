package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"pippaothy/internal/auth"
	"pippaothy/internal/logs"
	"pippaothy/internal/templates"
	"pippaothy/internal/users"
	"time"
)

func (s *Server) getHome(w http.ResponseWriter, r *http.Request) {
	u := s.getCtxUser(r)
	flashMessage := s.getFlashMessage(r)
	csrfToken := s.getCSRFToken(r)

	userName := ""
	loggedIn := u != nil
	if u != nil {
		userName = u.FirstName
	}

	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.Home(userName, flashMessage), "home", loggedIn, csrfToken).
		Render(r.Context(), w)
}

func (s *Server) getRegister(w http.ResponseWriter, r *http.Request) {
	csrfToken := s.getCSRFToken(r)
	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.Register(csrfToken), "register", false, csrfToken).Render(r.Context(), w)
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
		w.Write([]byte(fmt.Sprintf(`<div class="error">%s</div>`, err.Error())))
		return
	}

	// Check if user already exists
	if users.Exists(s.db, request.Email) {
		s.logger.Warn("registration attempt with existing email", "email", request.Email)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`<div class="error">Email already registered</div>`))
		return
	}

	// Create user
	uid, err := users.Create(s.db, request)
	if err != nil {
		s.logger.Error("failed to create user", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="error">Failed to create user</div>`))
		return
	}

	// Create session
	token, err := auth.CreateSession(s.db, uid)
	if err != nil {
		s.logger.Error("failed to create session", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="error">Failed to create session</div>`))
		return
	}

	auth.SetCookie(w, token)
	s.setFlashMessage(token, "Registration successful!")
	s.logger.Info("user registered successfully", "uid", uid)

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) getLogin(w http.ResponseWriter, r *http.Request) {
	csrfToken := s.getCSRFToken(r)
	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.Login(csrfToken), "login", false, csrfToken).Render(r.Context(), w)
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
	user, err := users.ByEmail(s.db, email)
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
	token, err := auth.CreateSession(s.db, user.UserId)
	if err != nil {
		s.logger.Error("failed to create session", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`<div class="error">Failed to create session</div>`))
		return
	}

	auth.SetCookie(w, token)
	s.setFlashMessage(token, "Login successful!")
	s.logger.Info("user logged in successfully", "uid", user.UserId)

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (s *Server) postLogout(w http.ResponseWriter, r *http.Request) {
	// Get session cookie and destroy session
	if cookie, err := r.Cookie("session_token"); err == nil {
		if err := auth.DestroySession(s.db, cookie.Value); err != nil {
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

// Simplified logs endpoint - single endpoint handles both page and HTMX requests
func (s *Server) getSimpleLogs(w http.ResponseWriter, r *http.Request) {
	query := logs.ParseQuery(r)
	result := logs.GetLogs(query)

	// Convert to template format
	var templateEntries []templates.SimpleLogEntry
	for _, entry := range result.Entries {
		templateEntries = append(templateEntries, templates.SimpleLogEntry{
			Timestamp: entry.Timestamp,
			Level:     entry.Level,
			Message:   entry.Message,
			ClientIP:  entry.ClientIP,
			Method:    entry.Method,
			Path:      entry.Path,
			RequestID: entry.RequestID,
		})
	}

	// Convert grouped entries if present
	var templateGroups map[string][]templates.SimpleLogEntry
	if result.Groups != nil {
		templateGroups = make(map[string][]templates.SimpleLogEntry)
		for key, entries := range result.Groups {
			var groupEntries []templates.SimpleLogEntry
			for _, entry := range entries {
				groupEntries = append(groupEntries, templates.SimpleLogEntry{
					Timestamp: entry.Timestamp,
					Level:     entry.Level,
					Message:   entry.Message,
					ClientIP:  entry.ClientIP,
					Method:    entry.Method,
					Path:      entry.Path,
					RequestID: entry.RequestID,
				})
			}
			templateGroups[key] = groupEntries
		}
	}

	data := templates.SimpleLogData{
		Entries: templateEntries,
		Groups:  templateGroups,
		Page:    result.Page,
		Limit:   result.Limit,
		HasMore: result.HasMore,
		Level:   query.Level,
		GroupBy: query.GroupBy,
		Error:   result.Error,
	}

	w.Header().Set("Content-Type", "text/html")

	// Check if this is an HTMX request (partial update)
	if r.Header.Get("HX-Request") == "true" {
		// Return just the log entries part
		templates.SimpleLogEntries(data).Render(r.Context(), w)
	} else {
		// Return full page
		user := s.getCtxUser(r)
		loggedIn := user != nil
		csrfToken := s.getCSRFToken(r)
		templates.Layout(templates.SimpleLogs(data), "logs", loggedIn, csrfToken).Render(r.Context(), w)
	}
}

// Health check endpoints
func (s *Server) getHealth(w http.ResponseWriter, r *http.Request) {
	// Basic health check - just return 200 OK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   fmt.Sprintf("%d", time.Now().Unix()),
	})
}

func (s *Server) getReady(w http.ResponseWriter, r *http.Request) {
	// Readiness check - verify database connectivity
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	
	if err := s.db.PingContext(ctx); err != nil {
		s.logger.Error("Readiness check failed - database ping failed", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "not ready",
			"reason": "database connection failed",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ready",
		"time":   fmt.Sprintf("%d", time.Now().Unix()),
	})
}
