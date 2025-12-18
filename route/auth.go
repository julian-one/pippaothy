package route

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"pippaothy/internal/auth"
	"pippaothy/internal/middleware"
	rdb "pippaothy/internal/redis"
	"pippaothy/internal/user"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type AuthResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	ExpiresIn    int64      `json:"expires_in"` // seconds
	User         *user.User `json:"user"`
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// maxBodySize limits request body to 1MB to prevent DoS.
const maxBodySize = 1 << 20

func Register(
	db *sqlx.DB,
	redisClient *redis.Client,
	issuer *auth.Issuer,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode register request", "error", err)
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Username == "" || req.Email == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "Username, email, and password are required")
			return
		}

		userID, err := user.Create(r.Context(), db, user.CreateRequest{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			// Check for unique constraint violation without revealing which field
			if user.IsConflict(err) {
				writeError(w, http.StatusConflict, "Unable to create account")
				return
			}
			logger.Error("failed to create user", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		accessToken, err := issuer.GenerateAccessToken(userID, req.Email, req.Username)
		if err != nil {
			logger.Error("failed to generate access token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		refreshToken := uuid.New().String()
		ttl := 24 * time.Hour
		if err := rdb.StoreRefresh(r.Context(), redisClient, refreshToken, userID, ttl); err != nil {
			logger.Error("failed to store refresh token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to store refresh token")
			return
		}

		createdUser, err := user.ByEmail(r.Context(), db, req.Email)
		if err != nil {
			logger.Error("failed to fetch created user", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to fetch user")
			return
		}

		writeJSON(w, http.StatusCreated, AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    300,
			User:         createdUser,
		})
	}
}

func Login(
	db *sqlx.DB,
	redisClient *redis.Client,
	issuer *auth.Issuer,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode login request", "error", err)
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		u, err := user.ByEmail(r.Context(), db, req.Email)
		if err != nil {
			logger.Warn("login attempt for non-existent user", "email", req.Email)
			writeError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		match, err := user.Verify(req.Password, u.Hash, u.Salt)
		if err != nil {
			logger.Error("failed to verify password", "error", err)
			writeError(w, http.StatusInternalServerError, "Authentication error")
			return
		}
		if !match {
			logger.Warn("failed login attempt", "email", req.Email)
			writeError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		accessToken, err := issuer.GenerateAccessToken(u.UserId, u.Email, u.Username)
		if err != nil {
			logger.Error("failed to generate access token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		refreshToken := uuid.New().String()
		ttl := 24 * time.Hour
		if err := rdb.StoreRefresh(r.Context(), redisClient, refreshToken, u.UserId, ttl); err != nil {
			logger.Error("failed to store refresh token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to store refresh token")
			return
		}

		writeJSON(w, http.StatusOK, AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    300,
			User:         u,
		})
	}
}

func RefreshToken(
	db *sqlx.DB,
	redisClient *redis.Client,
	issuer *auth.Issuer,
	logger *slog.Logger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode refresh request", "error", err)
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.RefreshToken == "" {
			writeError(w, http.StatusBadRequest, "Refresh token is required")
			return
		}

		userID, err := rdb.GetRefresh(r.Context(), redisClient, req.RefreshToken)
		if err != nil {
			logger.Warn("invalid refresh token attempt", "error", err)
			writeError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
			return
		}

		u, err := user.ByID(r.Context(), db, userID)
		if err != nil {
			logger.Error("failed to fetch user", "error", err)
			writeError(w, http.StatusUnauthorized, "User not found")
			return
		}

		accessToken, err := issuer.GenerateAccessToken(u.UserId, u.Email, u.Username)
		if err != nil {
			logger.Error("failed to generate access token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		newRefreshToken := uuid.New().String()
		ttl := 24 * time.Hour

		if err := rdb.DeleteRefresh(r.Context(), redisClient, req.RefreshToken, u.UserId); err != nil {
			logger.Error("failed to delete old refresh token", "error", err)
		}
		if err := rdb.StoreRefresh(r.Context(), redisClient, newRefreshToken, u.UserId, ttl); err != nil {
			logger.Error("failed to store new refresh token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to rotate refresh token")
			return
		}

		logger.Info("token refreshed", "user_id", u.UserId)

		writeJSON(w, http.StatusOK, AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    300,
			User:         u,
		})
	}
}

func Logout(redisClient *redis.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := ctx.Value(middleware.ClaimsKey).(*auth.Claims)
		if !ok {
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			if err := rdb.Blacklist(ctx, redisClient, claims.ID, ttl); err != nil {
				logger.Error("failed to blacklist token", "error", err, "jti", claims.ID)
			}
		}

		if err := rdb.DeleteUserRefresh(ctx, redisClient, claims.UserID); err != nil {
			logger.Error("failed to delete refresh tokens", "error", err)
		}

		logger.Info("user logged out", "user_id", claims.UserID, "email", claims.Email)

		w.WriteHeader(http.StatusNoContent)
	}
}
