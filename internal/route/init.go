package route

import (
	"net/http"
	"pippaothy/internal/middleware"
	"pippaothy/internal/templates"

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

		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.Home(userName, msg), "home", loggedIn).Render(r.Context(), w)
	}
}
