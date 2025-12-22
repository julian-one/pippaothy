package route

import (
	"net/http"

	"pippaothy/internal/auth"
	"pippaothy/internal/middleware"
)

func GetMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("get me handler started")

		ctx := r.Context()
		claims, ok := ctx.Value(middleware.ClaimsKey).(*auth.Claims)
		if !ok {
			log.Warn("get me failed: no claims in context")
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		log.Info("get me handler completed successfully", "user_id", claims.UserID)
		writeJSON(w, http.StatusOK, map[string]any{
			"user_id":  claims.UserID,
			"email":    claims.Email,
			"username": claims.Username,
		})
	}
}
