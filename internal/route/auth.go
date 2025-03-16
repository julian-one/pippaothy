package route

import (
	"encoding/json"
	"net/http"
	"pippaothy/internal/auth"
	"pippaothy/internal/templates"
	"pippaothy/internal/users"

	"github.com/jmoiron/sqlx"
)

func Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.Register(), "register", false).Render(r.Context(), w)
	}
}

func RegisterUser(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var request users.CreateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/html")
			templates.ServerError().Render(ctx, w)
			return
		}
		exists := users.Exists(db, request.Email)
		if exists {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/html")
			templates.RegisterError().Render(ctx, w)
			return
		}

		uid, err := users.Create(db, request)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/html")
			templates.ServerError().Render(ctx, w)
			return
		}
		token, err := auth.CreateSession(db, uid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/html")
			templates.ServerError().Render(ctx, w)
			return
		}
		auth.SetCookie(w, token)

		w.Header().Set("HX-Redirect", "/?message=Registration successful!")
		w.WriteHeader(http.StatusOK)
	}
}

func Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.Login(), "login", false).Render(r.Context(), w)
	}
}

func LoginUser(db *sqlx.DB) http.HandlerFunc {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "text/html")
			templates.LoginError().Render(ctx, w)
			return
		}

		user, err := users.ByEmail(db, request.Email)
		if err != nil || !users.VerifyPassword(user, request.Password) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("Content-Type", "text/html")
			templates.LoginError().Render(ctx, w)
			return
		}

		token, err := auth.CreateSession(db, user.UserId)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/html")
			templates.ServerError().Render(ctx, w)
			return
		}
		auth.SetCookie(w, token)
		// TODO: update users.last_login field

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
