package route

import (
	"encoding/json"
	"net/http"
	"time"

	"citadel/internal/auth"
	"citadel/internal/cache"
	"citadel/internal/middleware"
	"citadel/internal/user"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func Register(
	db *sqlx.DB,
	rdb *redis.Client,
	issuer *auth.Issuer,
) http.HandlerFunc {
	type Request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := middleware.GetLogger(r)
		log.Info("register handler started")

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode register request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		if req.Username == "" || req.Email == "" || req.Password == "" {
			log.Warn("registration validation failed: missing fields")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Username, email, and password are required"})
			return
		}

		log.Info("creating user in database", "email", req.Email)
		userId, err := user.Create(ctx, db, user.CreateRequest{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			if user.IsConflict(err) {
				log.Warn("user creation conflict", "email", req.Email)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				json.NewEncoder(w).Encode(map[string]string{"error": "Unable to create account"})
				return
			}
			log.Error("failed to create user", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user"})
			return
		}
		log.Info("user created in database", "user_id", userId)

		log.Info("generating access token", "user_id", userId)
		accessToken, err := issuer.GenerateAccessToken(userId, req.Email, req.Username)
		if err != nil {
			log.Error("failed to generate access token", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate token"})
			return
		}

		refreshToken := uuid.New().String()
		ttl := 24 * time.Hour
		log.Info("storing refresh token in redis", "user_id", userId)
		if err := cache.StoreRefresh(ctx, rdb, refreshToken, userId, ttl); err != nil {
			log.Error("failed to store refresh token", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to store refresh token"})
			return
		}

		log.Info("register handler completed successfully", "user_id", userId)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

func Login(
	db *sqlx.DB,
	rdb *redis.Client,
	issuer *auth.Issuer,
) http.HandlerFunc {
	type Request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("login handler started")

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode login request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		if req.Email == "" || req.Password == "" {
			log.Warn("login validation failed: missing fields")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Email and password are required"})
			return
		}

		log.Info("looking up user by email", "email", req.Email)
		u, err := user.ByEmail(r.Context(), db, req.Email)
		if err != nil {
			log.Warn("login attempt for non-existent user", "email", req.Email)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid email or password"})
			return
		}

		log.Info("verifying password", "user_id", u.UserId)
		match, err := user.Verify(req.Password, u.Hash, u.Salt)
		if err != nil {
			log.Error("failed to verify password", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Authentication error"})
			return
		}
		if !match {
			log.Warn("failed login attempt: invalid password", "email", req.Email)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid email or password"})
			return
		}

		log.Info("generating access token", "user_id", u.UserId)
		accessToken, err := issuer.GenerateAccessToken(u.UserId, u.Email, u.Username)
		if err != nil {
			log.Error("failed to generate access token", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate token"})
			return
		}

		refreshToken := uuid.New().String()
		ttl := 24 * time.Hour
		log.Info("storing refresh token in redis", "user_id", u.UserId)
		if err := cache.StoreRefresh(r.Context(), rdb, refreshToken, u.UserId, ttl); err != nil {
			log.Error("failed to store refresh token", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to store refresh token"})
			return
		}

		log.Info("login handler completed successfully", "user_id", u.UserId)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		})
	}
}

func RefreshToken(
	db *sqlx.DB,
	rdb *redis.Client,
	issuer *auth.Issuer,
) http.HandlerFunc {
	type Request struct {
		RefreshToken string `json:"refresh_token"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("refresh token handler started")

		var req Request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			log.Error("failed to decode refresh request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		if req.RefreshToken == "" {
			log.Warn("refresh token validation failed: missing token")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Refresh token is required"})
			return
		}

		log.Info("looking up refresh token in redis")
		userID, err := cache.GetRefresh(r.Context(), rdb, req.RefreshToken)
		if err != nil {
			log.Warn("invalid refresh token attempt", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Invalid or expired refresh token"})
			return
		}

		log.Info("fetching user from database", "user_id", userID)
		u, err := user.ByID(r.Context(), db, userID)
		if err != nil {
			log.Error("failed to fetch user", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
			return
		}

		log.Info("generating new access token", "user_id", u.UserId)
		accessToken, err := issuer.GenerateAccessToken(u.UserId, u.Email, u.Username)
		if err != nil {
			log.Error("failed to generate access token", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate token"})
			return
		}

		newRefreshToken := uuid.New().String()
		ttl := 24 * time.Hour

		log.Info("rotating refresh token in redis", "user_id", u.UserId)
		if err := cache.DeleteRefresh(r.Context(), rdb, req.RefreshToken, u.UserId); err != nil {
			log.Error("failed to delete old refresh token", "error", err)
		}
		if err := cache.StoreRefresh(r.Context(), rdb, newRefreshToken, u.UserId, ttl); err != nil {
			log.Error("failed to store new refresh token", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to rotate refresh token"})
			return
		}

		log.Info("refresh token handler completed successfully", "user_id", u.UserId)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"access_token":  accessToken,
			"refresh_token": newRefreshToken,
		})
	}
}

func Logout(rdb *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log := middleware.GetLogger(r)
		log.Info("logout handler started")

		ctx := r.Context()
		claims, ok := ctx.Value(middleware.ClaimsKey).(*auth.Claims)
		if !ok {
			log.Warn("logout failed: no claims in context")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Unauthorized"})
			return
		}

		log.Info("blacklisting access token", "user_id", claims.UserId, "jti", claims.ID)
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			if err := cache.Blacklist(ctx, rdb, claims.ID, ttl); err != nil {
				log.Error("failed to blacklist token", "error", err, "jti", claims.ID)
			}
		}

		log.Info("deleting user refresh tokens from redis", "user_id", claims.UserId)
		if err := cache.DeleteUserRefresh(ctx, rdb, claims.UserId); err != nil {
			log.Error("failed to delete refresh tokens", "error", err)
		}

		log.Info(
			"logout handler completed successfully",
			"user_id",
			claims.UserId,
			"email",
			claims.Email,
		)
		w.WriteHeader(http.StatusNoContent)
	}
}
