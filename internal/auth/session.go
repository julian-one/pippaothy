package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"pippaothy/internal/users"
	"time"

	"github.com/jmoiron/sqlx"
)

type Session struct {
	SessionId    string    `db:"session_id"    json:"session_id"`
	UserId       int64     `db:"user_id"       json:"user_id"`
	ExpiresAt    time.Time `db:"expires_at"    json:"expires_at"`
	FlashMessage *string   `db:"flash_message" json:"flash_message,omitempty"`
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
		return "", errors.Join(errors.New("failed to generate token"), err)
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	_, err = db.Exec(
		`INSERT INTO sessions (session_id, user_id, expires_at) VALUES ($1, $2, $3)`,
		token,
		userId,
		expiresAt,
	)
	if err != nil {
		return "", errors.Join(errors.New("failed to create session"), err)
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
		return nil, errors.New("invalid session")
	}
	return &user, nil
}

func SetCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})
}

func ResetCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now(),
		HttpOnly: true,
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
