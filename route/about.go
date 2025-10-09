package route

import (
	"net/http"

	ctxkeys "pippaothy/internal/context"
	"pippaothy/internal/templates"
	"pippaothy/internal/users"
)

// GetAbout returns a handler for the about page
func GetAbout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user, _ := r.Context().Value(ctxkeys.UserContextKey).(*users.User)
		loggedIn := user != nil

		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.AboutPage(), "About - Julian Roberts", loggedIn).
			Render(r.Context(), w)
	}
}