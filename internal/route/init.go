package route

import (
	"context"
	"encoding/json"
	"net/http"
	"pippaothy/internal/auth"
	"pippaothy/internal/middleware"
	"pippaothy/internal/templates"
	"pippaothy/internal/users"

	"github.com/jmoiron/sqlx"
)

type Router struct {
	DB *sqlx.DB
}

func NewRouter(db *sqlx.DB) *Router {
	return &Router{DB: db}
}

func (r *Router) RegisterRoutes(mux *http.ServeMux) {
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Authenticated routes
	mux.Handle("/", middleware.IsAuthenticated(r.DB, Home(r.DB)))
	mux.Handle("POST /logout", middleware.IsAuthenticated(r.DB, Logout(r.DB)))

	// Unauthenticated routes
	mux.Handle("GET /register", Register())
	mux.Handle("POST /register", RegisterUser(r.DB))
	mux.Handle("GET /login", Login())
	mux.Handle("POST /login", LoginUser(r.DB))
}

func Home(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetCtxUser(r)
		userName := ""
		loggedIn := user != nil

		if user != nil {
			userName = user.FirstName
		}

		msg := r.URL.Query().Get("message")

		comp := templates.Layout(templates.Home(userName, msg), "home", loggedIn)
		w.Header().Set("Content-Type", "text/html")

		if err := comp.Render(context.Background(), w); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}

func Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		comp := templates.Layout(templates.Register(), "register", false)
		w.Header().Set("Content-Type", "text/html")

		if err := comp.Render(context.Background(), w); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}

func RegisterUser(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request users.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Failed to decode request", http.StatusBadRequest)
			return
		}
		uid, err := users.Create(db, request)
		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
		token, err := auth.CreateSession(db, uid)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}
		auth.SetCookie(w, token)

		w.Header().Set("HX-Redirect", "/?message=Registration successful!")
		w.WriteHeader(http.StatusOK)
	}
}

func Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		comp := templates.Layout(templates.Login(), "login", false)
		w.Header().Set("Content-Type", "text/html")

		if err := comp.Render(context.Background(), w); err != nil {
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
		}
	}
}

func LoginUser(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Failed to decode request", http.StatusBadRequest)
			return
		}
		user, err := users.ByEmail(db, request.Email)
		if err != nil {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		if !users.VerifyPassword(user, request.Password) {
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		token, err := auth.CreateSession(db, user.UserId)
		if err != nil {
			http.Error(w, "Failed to create session", http.StatusInternalServerError)
			return
		}
		auth.SetCookie(w, token)

		w.Header().Set("HX-Redirect", "/?message=Login successful!")
		w.WriteHeader(http.StatusOK)
	}
}

func Logout(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err == nil {
			err = auth.DestorySession(db, cookie.Value)
			if err != nil {
				http.Error(w, "Failed to destroy session", http.StatusInternalServerError)
				return
			}
		}
		auth.ResetCookie(w)

		w.Header().Set("HX-Redirect", "/?message=Logout successful!")
		w.WriteHeader(http.StatusOK)
	}
}
