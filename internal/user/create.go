package user

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
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

// ErrConflict indicates a unique constraint violation.
var ErrConflict = errors.New("unique constraint violation")

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
		if IsConflict(err) {
			return 0, ErrConflict
		}
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	return uid, nil
}

// IsConflict checks if the error is a unique constraint violation (PostgreSQL).
func IsConflict(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL unique violation error code is 23505
	// The error message contains "duplicate key" or "unique constraint"
	errStr := err.Error()
	return strings.Contains(errStr, "23505") ||
		strings.Contains(errStr, "duplicate key") ||
		strings.Contains(errStr, "unique constraint") ||
		errors.Is(err, ErrConflict)
}

func Verify(password string, storedHash string, salt []byte) (bool, error) {
	computed, _, err := hash(password, salt)
	if err != nil {
		return false, fmt.Errorf("failed to compute hash: %w", err)
	}
	match := subtle.ConstantTimeCompare([]byte(computed), []byte(storedHash)) == 1
	return match, nil
}

// hash uses scrypt with OWASP 2024 recommended parameters.
// N=32768 (2^15), r=8, p=1, keyLen=32
// See: https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
func hash(password string, salt []byte) (string, []byte, error) {
	if salt == nil {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return "", nil, fmt.Errorf("failed to generate salt: %w", err)
		}
	}

	hash, err := scrypt.Key([]byte(password), salt, 32768, 8, 1, 32)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(hash), salt, nil
}
