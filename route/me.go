package route

import (
	"net/http"

	"pippaothy/internal/auth"
	"pippaothy/internal/middleware"
)

func GetMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := ctx.Value(middleware.ClaimsKey).(*auth.Claims)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":  claims.UserID,
			"email":    claims.Email,
			"username": claims.Username,
		})
	}
}
