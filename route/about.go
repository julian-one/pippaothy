package route

import (
	"net/http"

	"pippaothy/internal/middleware"
	"pippaothy/internal/templates"
)

// GetAbout returns a handler for the about page
func GetAbout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user := middleware.GetUserFromContext(r)
		loggedIn := user != nil

		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.AboutPage(), "About - Julian Roberts", loggedIn).
			Render(r.Context(), w)
	}
}