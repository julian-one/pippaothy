package route

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GetHealth returns a handler for health check endpoint
func GetHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"time":   fmt.Sprintf("%d", time.Now().Unix()),
		})
	}
}