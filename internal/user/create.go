package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/scrypt"
)

type User struct {
	UserId    int64      `db:"user_id"       json:"user_id"`
	Username  string     `db:"username"      json:"username"`
	Email     string     `db:"email"         json:"email"`
	Hash      string     `db:"password_hash" json:"-"`
	Salt      []byte     `db:"salt"          json:"-"`
	LastLogin *time.Time `db:"last_login"    json:"last_login,omitempty"`
	CreatedAt time.Time  `db:"created_at"    json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"    json:"updated_at"`
}

type CreateRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func Create(ctx context.Context, db *sqlx.DB, request CreateRequest) (int64, error) {
	h, s, err := hash(request.Password, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	var uid int64
	err = db.QueryRowContext(
		ctx,
		`INSERT INTO users (username, email, password_hash, salt) VALUES ($1, $2, $3, $4) RETURNING user_id`,
		request.Username,
		request.Email,
		h,
		s,
	).Scan(&uid)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	return uid, nil
}

func Verify(password string, storedHash string, salt []byte) bool {
	computed, _, err := hash(password, salt)
	if err != nil {
		return false
	}
	return computed == storedHash
}

// https://pkg.go.dev/golang.org/x/crypto/scrypt#pkg-overview
func hash(password string, salt []byte) (string, []byte, error) {
	if salt == nil {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return "", nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	hash, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(hash), salt, nil
}
