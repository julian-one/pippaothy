package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"pippaothy/internal/user"

	"github.com/jmoiron/sqlx"
)

type Session struct {
	SessionId    string    `db:"session_id"    json:"session_id"`
	UserId       int64     `db:"user_id"       json:"user_id"`
	FlashMessage *string   `db:"flash_message" json:"flash_message,omitempty"`
	ExpiresAt    time.Time `db:"expires_at"    json:"expires_at"`
}

func isProduction() bool {
	env := strings.ToLower(os.Getenv("GO_ENV"))
	if env == "production" || env == "prod" {
		return true
	}
	if strings.ToLower(os.Getenv("TLS_ENABLED")) == "true" {
		return true
	}
	return false
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func CreateSession(ctx context.Context, db *sqlx.DB, userId int64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = db.ExecContext(ctx,
		`INSERT INTO sessions (session_id, user_id, expires_at) VALUES ($1, $2, $3)`,
		token,
		userId,
		expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	return token, nil
}

func DestroySession(ctx context.Context, db *sqlx.DB, token string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM sessions WHERE session_id = $1`, token)
	if err != nil {
		return fmt.Errorf("failed to destroy session: %w", err)
	}
	return nil
}

func GetSession(ctx context.Context, db *sqlx.DB, token string) (*user.User, error) {
	var u user.User
	err := db.GetContext(ctx, &u, `
		SELECT u.* FROM users u
		INNER JOIN sessions s ON (s.user_id = u.user_id)
		WHERE s.session_id = $1 AND s.expires_at > $2`,
		token, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}
	return &u, nil
}

func SetCookie(w http.ResponseWriter, token string) {
	isSecure := isProduction()
	sameSite := http.SameSiteLaxMode
	if isSecure {
		sameSite = http.SameSiteStrictMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
	})
}

func ResetCookie(w http.ResponseWriter) {
	isSecure := isProduction()
	sameSite := http.SameSiteLaxMode
	if isSecure {
		sameSite = http.SameSiteStrictMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now(),
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: sameSite,
	})
}

func SetFlashMessage(ctx context.Context, db *sqlx.DB, token, message string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE sessions 
		SET flash_message = $1 
		WHERE session_id = $2 AND expires_at > $3`,
		message, token, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("failed to set flash message: %w", err)
	}
	return nil
}

func GetAndClearFlashMessage(ctx context.Context, db *sqlx.DB, token string) (string, error) {
	var message sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT flash_message 
		FROM sessions 
		WHERE session_id = $1 AND expires_at > $2`,
		token, time.Now().UTC()).Scan(&message)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No session found, return empty string
		}
		return "", fmt.Errorf("failed to get flash message: %w", err)
	}

	if message.Valid && message.String != "" {
		_, err = db.ExecContext(ctx, `
			UPDATE sessions 
			SET flash_message = NULL 
			WHERE session_id = $1 AND expires_at > $2`,
			token, time.Now().UTC())
		if err != nil {
			return "", fmt.Errorf("failed to clear flash message: %w", err)
		}
		return message.String, nil
	}
	return "", nil
}
