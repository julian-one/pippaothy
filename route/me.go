package route

import (
	"encoding/json"
	"net/http"

	"citadel/internal/auth"
	"citadel/internal/middleware"
)

func GetMe() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("get me handler started")

		ctx := r.Context()
		claims, ok := ctx.Value(middleware.ClaimsKey).(*auth.Claims)
		if !ok {
			log.Warn("get me failed: no claims in context")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		log.Info("get me handler completed successfully", "user_id", claims.UserId)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"user_id":  claims.UserId,
			"email":    claims.Email,
			"username": claims.Username,
		})
	}
}
