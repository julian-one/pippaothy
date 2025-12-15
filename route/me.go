package route

import (
	"encoding/json"
	"net/http"

	"pippaothy/internal/auth"
)

func GetMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := ctx.Value(auth.ClaimsKey).(*auth.Claims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"user_id":  claims.UserID,
			"email":    claims.Email,
			"username": claims.Username,
		})
	}
}
