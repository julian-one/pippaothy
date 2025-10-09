package route

import (
	"net/http"

	ctxkeys "pippaothy/internal/context"
	"pippaothy/internal/templates"
	"pippaothy/internal/users"
)

// GetHome returns a handler for the home page
func GetHome() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context
		user, _ := r.Context().Value(ctxkeys.UserContextKey).(*users.User)

		// Get flash message from context
		flashMessage := ""
		if msg, ok := r.Context().Value(ctxkeys.FlashMessageContextKey).(string); ok {
			flashMessage = msg
		}

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