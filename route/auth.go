package route

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"pippaothy/internal/auth"
	"pippaothy/internal/redis"
	"pippaothy/internal/user"

	"github.com/jmoiron/sqlx"
)

type AuthResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	ExpiresIn    int64      `json:"expires_in"` // seconds
	User         *user.User `json:"user"`
}

func Register(db *sqlx.DB, redisClient *redis.Client, logger *slog.Logger) http.HandlerFunc {
	type Request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode register request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Username == "" || req.Email == "" || req.Password == "" {
			http.Error(w, "Username, email, and password are required", http.StatusBadRequest)
			return
		}

		if user.Exists(r.Context(), db, req.Email) {
			http.Error(w, "User with this email already exists", http.StatusConflict)
			return
		}

		userID, err := user.Create(r.Context(), db, user.CreateRequest{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			logger.Error("failed to create user", "error", err)
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}

		// Generate access token (5 minutes)
		accessToken, _, err := auth.GenerateAccessToken(userID, req.Email, req.Username)
		if err != nil {
			logger.Error("failed to generate access token", "error", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Generate and store refresh token in Redis (24 hours)
		refreshToken := auth.GenerateRefreshToken()
		ttl := 24 * time.Hour
		if err := redisClient.StoreRefreshToken(r.Context(), refreshToken, userID, ttl); err != nil {
			logger.Error("failed to store refresh token", "error", err)
			http.Error(w, "Failed to store refresh token", http.StatusInternalServerError)
			return
		}

		createdUser, err := user.ByEmail(r.Context(), db, req.Email)
		if err != nil {
			logger.Error("failed to fetch created user", "error", err)
			http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    300, // 5 minutes in seconds
			User:         createdUser,
		})
	}
}

func Login(db *sqlx.DB, redisClient *redis.Client, logger *slog.Logger) http.HandlerFunc {
	type Request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode login request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Email == "" || req.Password == "" {
			http.Error(w, "Email and password are required", http.StatusBadRequest)
			return
		}

		u, err := user.ByEmail(r.Context(), db, req.Email)
		if err != nil {
			logger.Warn("login attempt for non-existent user", "email", req.Email)
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		if !user.Verify(req.Password, u.Hash, u.Salt) {
			logger.Warn("failed login attempt", "email", req.Email)
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
			return
		}

		// Generate access token (5 minutes)
		accessToken, _, err := auth.GenerateAccessToken(u.UserId, u.Email, u.Username)
		if err != nil {
			logger.Error("failed to generate access token", "error", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Generate and store refresh token in Redis (24 hours)
		refreshToken := auth.GenerateRefreshToken()
		ttl := 24 * time.Hour
		if err := redisClient.StoreRefreshToken(r.Context(), refreshToken, u.UserId, ttl); err != nil {
			logger.Error("failed to store refresh token", "error", err)
			http.Error(w, "Failed to store refresh token", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    300, // 5 minutes in seconds
			User:         u,
		})
	}
}

func RefreshTokenHandler(
	db *sqlx.DB,
	redisClient *redis.Client,
	logger *slog.Logger,
) http.HandlerFunc {
	type Request struct {
		RefreshToken string `json:"refresh_token"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode refresh request", "error", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.RefreshToken == "" {
			http.Error(w, "Refresh token is required", http.StatusBadRequest)
			return
		}

		// Validate refresh token from Redis
		userID, err := redisClient.GetRefreshToken(r.Context(), req.RefreshToken)
		if err != nil {
			logger.Warn("invalid refresh token attempt", "error", err)
			http.Error(w, "Invalid or expired refresh token", http.StatusUnauthorized)
			return
		}

		// Get user info
		u, err := user.ByID(r.Context(), db, userID)
		if err != nil {
			logger.Error("failed to fetch user", "error", err)
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}

		// Generate new access token
		accessToken, _, err := auth.GenerateAccessToken(u.UserId, u.Email, u.Username)
		if err != nil {
			logger.Error("failed to generate access token", "error", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// Rotate refresh token (more secure - old token becomes invalid)
		newRefreshToken := auth.GenerateRefreshToken()
		ttl := 24 * time.Hour

		// Delete old refresh token and store new one
		if err := redisClient.DeleteRefreshToken(r.Context(), req.RefreshToken, u.UserId); err != nil {
			logger.Error("failed to delete old refresh token", "error", err)
		}
		if err := redisClient.StoreRefreshToken(r.Context(), newRefreshToken, u.UserId, ttl); err != nil {
			logger.Error("failed to store new refresh token", "error", err)
			http.Error(w, "Failed to rotate refresh token", http.StatusInternalServerError)
			return
		}

		logger.Info("token refreshed", "user_id", u.UserId)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    300, // 5 minutes
			User:         u,
		})
	}
}

func Logout(redisClient *redis.Client, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		claims, ok := ctx.Value(auth.ClaimsKey).(*auth.Claims)
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Blacklist the access token in Redis
		// TTL = time until token expires
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			if err := redisClient.BlacklistToken(ctx, claims.ID, ttl); err != nil {
				logger.Error("failed to blacklist token", "error", err, "jti", claims.ID)
				// Don't fail the logout if Redis fails
			}
		}

		// Delete all refresh tokens for this user from Redis
		if err := redisClient.DeleteUserRefreshTokens(ctx, claims.UserID); err != nil {
			logger.Error("failed to delete refresh tokens", "error", err)
			// Don't fail the logout if Redis delete fails
		}

		logger.Info("user logged out", "user_id", claims.UserID, "email", claims.Email)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Successfully logged out",
		})
	}
}
