package route

import (
	"encoding/json"
	"net/http"
	"time"
)

func GetHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "OK",
			"time":   time.Now().Format("2006-01-02 15:04:05"),
		})
	}
}
