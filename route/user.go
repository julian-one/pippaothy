package route

import (
	"encoding/json"
	"net/http"
	"strconv"

	"pippaothy/internal/middleware"
	"pippaothy/internal/user"

	"github.com/jmoiron/sqlx"
)

func ListUsers(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("list users handler started")

		ctx := r.Context()
		log.Info("querying all users from database")
		users, err := user.List(ctx, db)
		if err != nil {
			log.Error("failed to list users", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to list users")
			return
		}

		log.Info("list users handler completed successfully", "count", len(users))
		writeJSON(w, http.StatusOK, users)
	}
}

func UpdateUser(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("update user handler started")

		ctx := r.Context()
		id := r.PathValue("id")
		if id == "" {
			log.Warn("update user validation failed: missing user ID")
			writeError(w, http.StatusBadRequest, "User ID is required")
			return
		}

		userID, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			log.Warn("update user validation failed: invalid user ID", "id", id)
			writeError(w, http.StatusBadRequest, "Invalid user ID")
			return
		}
		log.Info("parsed user ID", "user_id", userID)

		var req user.UpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode update request", "error", err)
			writeError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		log.Info("updating user in database", "user_id", userID)
		if err := user.Update(ctx, db, userID, req); err != nil {
			log.Error("failed to update user", "error", err, "user_id", userID)
			if err.Error() == "user not found" {
				writeError(w, http.StatusNotFound, "User not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "Failed to update user")
			return
		}

		log.Info("fetching updated user from database", "user_id", userID)
		updatedUser, err := user.ByID(ctx, db, userID)
		if err != nil {
			log.Error("failed to fetch updated user", "error", err)
			writeError(w, http.StatusInternalServerError, "User updated but failed to fetch")
			return
		}

		log.Info("update user handler completed successfully", "user_id", userID)
		writeJSON(w, http.StatusOK, updatedUser)
	}
}
