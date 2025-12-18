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
		ctx := r.Context()
		users, err := user.List(ctx, db)
		if err != nil {
			logger.Error("failed to list users", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to list users")
			return
		}

		writeJSON(w, http.StatusOK, users)
	}
}

func UpdateUser(db *sqlx.DB, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := r.PathValue("id")
		if id == "" {
			writeError(w, http.StatusBadRequest, "User ID is required")
			return
		}

		userID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}

		var req user.UpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode update request", "error", err)
			writeError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if err := user.Update(ctx, db, userID, req); err != nil {
			logger.Error("failed to update user", "error", err, "user_id", userID)
			if err.Error() == "user not found" {
				writeError(w, http.StatusNotFound, "User not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Failed to update user")
			return
		}

		updatedUser, err := user.ByID(ctx, db, userID)
		if err != nil {
			logger.Error("failed to fetch updated user", "error", err)
			writeError(w, http.StatusInternalServerError, "User updated but failed to fetch")
			return
		}

		logger.Info("user updated", "user_id", userID)
		writeJSON(w, http.StatusOK, updatedUser)
	}
}
