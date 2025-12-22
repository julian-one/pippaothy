package route

import (
	"encoding/json"
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
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("register handler started")

		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode register request", "error", err)
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Username == "" || req.Email == "" || req.Password == "" {
			log.Warn("registration validation failed: missing fields")
			writeError(w, http.StatusBadRequest, "Username, email, and password are required")
			return
		}

		log.Info("creating user in database", "email", req.Email)
		userID, err := user.Create(r.Context(), db, user.CreateRequest{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			if user.IsConflict(err) {
				log.Warn("user creation conflict", "email", req.Email)
				writeError(w, http.StatusConflict, "Unable to create account")
				return
			}
			log.Error("failed to create user", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}
		log.Info("user created in database", "user_id", userID)

		log.Info("generating access token", "user_id", userID)
		accessToken, err := issuer.GenerateAccessToken(userID, req.Email, req.Username)
		if err != nil {
			log.Error("failed to generate access token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		refreshToken := uuid.New().String()
		ttl := 24 * time.Hour
		log.Info("storing refresh token in redis", "user_id", userID)
		if err := rdb.StoreRefresh(r.Context(), redisClient, refreshToken, userID, ttl); err != nil {
			log.Error("failed to store refresh token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to store refresh token")
			return
		}

		log.Info("fetching created user from database", "email", req.Email)
		createdUser, err := user.ByEmail(r.Context(), db, req.Email)
		if err != nil {
			log.Error("failed to fetch created user", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to fetch user")
			return
		}

		log.Info("register handler completed successfully", "user_id", userID)
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
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("login handler started")

		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode login request", "error", err)
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Email == "" || req.Password == "" {
			log.Warn("login validation failed: missing fields")
			writeError(w, http.StatusBadRequest, "Email and password are required")
			return
		}

		log.Info("looking up user by email", "email", req.Email)
		u, err := user.ByEmail(r.Context(), db, req.Email)
		if err != nil {
			log.Warn("login attempt for non-existent user", "email", req.Email)
			writeError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		log.Info("verifying password", "user_id", u.UserId)
		match, err := user.Verify(req.Password, u.Hash, u.Salt)
		if err != nil {
			log.Error("failed to verify password", "error", err)
			writeError(w, http.StatusInternalServerError, "Authentication error")
			return
		}
		if !match {
			log.Warn("failed login attempt: invalid password", "email", req.Email)
			writeError(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}

		log.Info("generating access token", "user_id", u.UserId)
		accessToken, err := issuer.GenerateAccessToken(u.UserId, u.Email, u.Username)
		if err != nil {
			log.Error("failed to generate access token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		refreshToken := uuid.New().String()
		ttl := 24 * time.Hour
		log.Info("storing refresh token in redis", "user_id", u.UserId)
		if err := rdb.StoreRefresh(r.Context(), redisClient, refreshToken, u.UserId, ttl); err != nil {
			log.Error("failed to store refresh token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to store refresh token")
			return
		}

		log.Info("login handler completed successfully", "user_id", u.UserId)
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
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("refresh token handler started")

		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		var req refreshRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode refresh request", "error", err)
			writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.RefreshToken == "" {
			log.Warn("refresh token validation failed: missing token")
			writeError(w, http.StatusBadRequest, "Refresh token is required")
			return
		}

		log.Info("looking up refresh token in redis")
		userID, err := rdb.GetRefresh(r.Context(), redisClient, req.RefreshToken)
		if err != nil {
			log.Warn("invalid refresh token attempt", "error", err)
			writeError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
			return
		}

		log.Info("fetching user from database", "user_id", userID)
		u, err := user.ByID(r.Context(), db, userID)
		if err != nil {
			log.Error("failed to fetch user", "error", err)
			writeError(w, http.StatusUnauthorized, "User not found")
			return
		}

		log.Info("generating new access token", "user_id", u.UserId)
		accessToken, err := issuer.GenerateAccessToken(u.UserId, u.Email, u.Username)
		if err != nil {
			log.Error("failed to generate access token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to generate token")
			return
		}

		newRefreshToken := uuid.New().String()
		ttl := 24 * time.Hour

		log.Info("rotating refresh token in redis", "user_id", u.UserId)
		if err := rdb.DeleteRefresh(r.Context(), redisClient, req.RefreshToken, u.UserId); err != nil {
			log.Error("failed to delete old refresh token", "error", err)
		}
		if err := rdb.StoreRefresh(r.Context(), redisClient, newRefreshToken, u.UserId, ttl); err != nil {
			log.Error("failed to store new refresh token", "error", err)
			writeError(w, http.StatusInternalServerError, "Failed to rotate refresh token")
			return
		}

		log.Info("refresh token handler completed successfully", "user_id", u.UserId)
		writeJSON(w, http.StatusOK, AuthResponse{
			AccessToken:  accessToken,
			RefreshToken: newRefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    300,
			User:         u,
		})
	}
}

func Logout(redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("logout handler started")

		ctx := r.Context()
		claims, ok := ctx.Value(middleware.ClaimsKey).(*auth.Claims)
		if !ok {
			log.Warn("logout failed: no claims in context")
			writeError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		log.Info("blacklisting access token", "user_id", claims.UserID, "jti", claims.ID)
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			if err := rdb.Blacklist(ctx, redisClient, claims.ID, ttl); err != nil {
				log.Error("failed to blacklist token", "error", err, "jti", claims.ID)
			}
		}

		log.Info("deleting user refresh tokens from redis", "user_id", claims.UserID)
		if err := rdb.DeleteUserRefresh(ctx, redisClient, claims.UserID); err != nil {
			log.Error("failed to delete refresh tokens", "error", err)
		}

		log.Info("logout handler completed successfully", "user_id", claims.UserID, "email", claims.Email)
		w.WriteHeader(http.StatusNoContent)
	}
}
