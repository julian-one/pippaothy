package route

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"pippaothy/internal/user"

	"github.com/jmoiron/sqlx"
)

func ListUsers(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := user.List(r.Context(), db)
		if err != nil {
			logger.Error("failed to list users", "error", err)
			http.Error(w, "Failed to retrieve users", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

func UpdateUser(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract user ID from path parameter
		userIDStr := r.PathValue("id")
		if userIDStr == "" {
			http.Error(w, "User ID is required", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid user ID", http.StatusBadRequest)
			return
		}

		var req user.UpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode update request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if err := user.Update(r.Context(), db, userID, req); err != nil {
			logger.Error("failed to update user", "error", err, "user_id", userID)
			if err.Error() == "user not found" {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
			http.Error(w, "Failed to update user", http.StatusInternalServerError)
			return
		}

		// Fetch and return the updated user
		updatedUser, err := user.ByID(r.Context(), db, userID)
		if err != nil {
			logger.Error("failed to fetch updated user", "error", err)
			http.Error(w, "User updated but failed to fetch", http.StatusInternalServerError)
			return
		}

		logger.Info("user updated", "user_id", userID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedUser)
	}
}
