package server

import (
	"net/http"
	"pippaothy/internal/auth"
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

func (s *Server) getRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.Register(), "register", false).Render(r.Context(), w)
}

func (s *Server) postRegister(w http.ResponseWriter, r *http.Request) {
	request := users.CreateRequest{
		FirstName: r.FormValue("first_name"),
		LastName:  r.FormValue("last_name"),
		Email:     r.FormValue("email"),
		Password:  r.FormValue("password"),
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
	w.Header().Set("Content-Type", "text/html")
	templates.Layout(templates.Login(), "login", false).Render(r.Context(), w)
}

func (s *Server) postLogin(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.logger.Error("failed to parse login form", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK) // Always return 200 for HTMX to process
		w.Write([]byte(`<div class="error">Invalid form data</div>`))
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Get user by email
	user, err := users.ByEmail(s.db, email)
	if err != nil {
		s.logger.Warn("login failed - user not found", "email", email)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK) // Always return 200 for HTMX to process
		w.Write([]byte(`<div class="error">Invalid credentials</div>`))
		return
	}

	// Verify password
	if !users.VerifyPassword(user, password) {
		s.logger.Warn("login failed - invalid password", "email", email)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK) // Always return 200 for HTMX to process
		w.Write([]byte(`<div class="error">Invalid credentials</div>`))
		return
	}

	// Create session
	token, err := auth.CreateSession(s.db, user.UserId)
	if err != nil {
		s.logger.Error("failed to create session", "error", err)
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK) // Always return 200 for HTMX to process
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
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	auth.ResetCookie(w)
	s.logger.Info("user logged out successfully")

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}
