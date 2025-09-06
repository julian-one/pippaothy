package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"pippaothy/internal/users"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Session struct {
	SessionId    string    `db:"session_id"    json:"session_id"`
	UserId       int64     `db:"user_id"       json:"user_id"`
	ExpiresAt    time.Time `db:"expires_at"    json:"expires_at"`
	FlashMessage *string   `db:"flash_message" json:"flash_message,omitempty"`
}

// isProduction checks if the application is running in production mode
// based on environment variables
func isProduction() bool {
	env := strings.ToLower(os.Getenv("GO_ENV"))
	if env == "production" || env == "prod" {
		return true
	}

	// Check if TLS_ENABLED is explicitly set
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

func CreateSession(db *sqlx.DB, userId int64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = db.Exec(
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

func DestroySession(db *sqlx.DB, token string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE session_id = $1`, token)
	return err
}

func GetSession(db *sqlx.DB, token string) (*users.User, error) {
	var user users.User
	err := db.Get(&user, `
		SELECT u.* FROM users u
		INNER JOIN sessions s ON (s.user_id = u.user_id)
		WHERE s.session_id = $1 AND s.expires_at > $2`,
		token, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}
	return &user, nil
}

func SetCookie(w http.ResponseWriter, token string) {
	isSecure := isProduction()
	sameSite := http.SameSiteLaxMode
	if isSecure {
		sameSite = http.SameSiteStrictMode // Use Strict in production for better security
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Secure:   isSecure, // Dynamic based on environment
		SameSite: sameSite, // Dynamic based on environment
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
		Secure:   isSecure, // Dynamic based on environment
		SameSite: sameSite, // Dynamic based on environment
	})
}

func SetFlashMessage(db *sqlx.DB, token, message string) error {
	_, err := db.Exec(`
		UPDATE sessions 
		SET flash_message = $1 
		WHERE session_id = $2 AND expires_at > $3`,
		message, token, time.Now().UTC())
	return err
}

func GetAndClearFlashMessage(db *sqlx.DB, token string) (string, error) {
	var message sql.NullString

	err := db.QueryRow(`
		SELECT flash_message 
		FROM sessions 
		WHERE session_id = $1 AND expires_at > $2`,
		token, time.Now().UTC()).Scan(&message)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil // No session found, return empty string
		}
		return "", err
	}

	if message.Valid && message.String != "" {
		_, err = db.Exec(`
			UPDATE sessions 
			SET flash_message = NULL 
			WHERE session_id = $1 AND expires_at > $2`,
			token, time.Now().UTC())
		if err != nil {
			return "", err
		}
		return message.String, nil
	}
	return "", nil
}
