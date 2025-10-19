package route

import (
	"net/http"

	"pippaothy/internal/middleware"
	"pippaothy/internal/templates"
)

// GetHome returns a handler for the home page
func GetHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user := middleware.GetUserFromContext(r)

		// Get flash message from context
		flashMessage := middleware.GetFlashMessageFromContext(r)

		userName := ""
		loggedIn := user != nil
		if user != nil {
			userName = user.FirstName
		}

		w.Header().Set("Content-Type", "text/html")
		templates.Layout(templates.Home(userName, flashMessage), "home", loggedIn).
			Render(r.Context(), w)
	}
}