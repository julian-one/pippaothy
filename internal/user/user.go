package user

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/scrypt"
)

type User struct {
	// Primary identifier
	UserId int64 `db:"user_id" json:"user_id"`

	// User information
	FirstName string `db:"first_name" json:"first_name"`
	LastName  string `db:"last_name"  json:"last_name"`
	Email     string `db:"email"      json:"email"`

	// Authentication
	Hash string `db:"password_hash" json:"-"` // Never expose in JSON
	Salt []byte `db:"salt"          json:"-"` // Never expose in JSON

	// Timestamps
	LastLogin *time.Time `db:"last_login" json:"last_login,omitempty"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

type CreateRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

type PasswordReset struct {
	// Primary identifier
	ResetId int64 `db:"reset_id" json:"reset_id"`

	// Relationships
	UserId int64 `db:"user_id" json:"user_id"`

	// Reset information
	Token     string    `db:"token"      json:"-"` // Never expose token in JSON
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	Used      bool      `db:"used"       json:"used"`

	// Timestamps
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// Validation functions
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func ValidateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if len(email) > 254 {
		return errors.New("email is too long")
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email format")
	}
	return nil
}

func ValidateName(name, fieldName string) error {
	if name == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	name = strings.TrimSpace(name)
	if len(name) < 1 {
		return fmt.Errorf("%s is required", fieldName)
	}
	if len(name) > 50 {
		return fmt.Errorf("%s is too long (max 50 characters)", fieldName)
	}
	// Check for potentially dangerous characters
	for _, r := range name {
		if r < 32 || r == 127 { // Control characters
			return fmt.Errorf("%s contains invalid characters", fieldName)
		}
	}
	return nil
}

func ValidatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if len(password) > 128 {
		return errors.New("password is too long (max 128 characters)")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	var missing []string
	if !hasUpper {
		missing = append(missing, "uppercase letter")
	}
	if !hasLower {
		missing = append(missing, "lowercase letter")
	}
	if !hasDigit {
		missing = append(missing, "digit")
	}
	if !hasSpecial {
		missing = append(missing, "special character")
	}

	if len(missing) > 0 {
		return fmt.Errorf("password must contain at least one: %s", strings.Join(missing, ", "))
	}

	return nil
}

func (req *CreateRequest) Validate() error {
	if err := ValidateName(req.FirstName, "first name"); err != nil {
		return err
	}
	if err := ValidateName(req.LastName, "last name"); err != nil {
		return err
	}
	if err := ValidateEmail(req.Email); err != nil {
		return err
	}
	if err := ValidatePassword(req.Password); err != nil {
		return err
	}
	return nil
}

func (req *CreateRequest) Sanitize() {
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
}

func ByEmail(ctx context.Context, db *sqlx.DB, email string) (*User, error) {
	var u User
	err := db.GetContext(ctx, &u, `SELECT * FROM users WHERE email = $1`, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

func Exists(ctx context.Context, db *sqlx.DB, email string) bool {
	var exists bool
	if err := db.GetContext(ctx, &exists, "SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)", email); err != nil {
		return false
	}
	return exists
}

func Create(ctx context.Context, db *sqlx.DB, request CreateRequest) (int64, error) {
	h, s, err := hash(request.Password, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	var uid int64
	err = db.QueryRowContext(
		ctx,
		`INSERT INTO users (first_name, last_name, email, password_hash, salt) VALUES ($1, $2, $3, $4, $5) RETURNING user_id`,
		request.FirstName,
		request.LastName,
		request.Email,
		h,
		s,
	).Scan(&uid)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	return uid, nil
}

func VerifyPassword(user *User, password string) bool {
	hashed, _, err := hash(password, user.Salt)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hashed), []byte(user.Hash)) == 1
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

func generateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate reset token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func CreatePasswordReset(ctx context.Context, db *sqlx.DB, email string) (string, error) {
	user, err := ByEmail(ctx, db, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to find user: %w", err)
	}

	token, err := generateResetToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(time.Hour)

	_, err = db.ExecContext(ctx, `
		INSERT INTO password_resets (user_id, token, expires_at) 
		VALUES ($1, $2, $3)`,
		user.UserId, token, expiresAt,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create password reset: %w", err)
	}

	return token, nil
}

func ValidateResetToken(ctx context.Context, db *sqlx.DB, token string) (*PasswordReset, error) {
	var reset PasswordReset
	err := db.GetContext(ctx, &reset, `
		SELECT * FROM password_resets 
		WHERE token = $1 AND expires_at > NOW() AND used = false`,
		token,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("invalid or expired reset token")
		}
		return nil, fmt.Errorf("failed to validate reset token: %w", err)
	}
	return &reset, nil
}

func ResetPassword(ctx context.Context, db *sqlx.DB, token, newPassword string) error {
	reset, err := ValidateResetToken(ctx, db, token)
	if err != nil {
		return err
	}

	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	hashed, salt, err := hash(newPassword, nil)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE users 
		SET password_hash = $1, salt = $2, updated_at = CURRENT_TIMESTAMP 
		WHERE user_id = $3`,
		hashed, salt, reset.UserId,
	)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE password_resets 
		SET used = true 
		WHERE reset_id = $1`,
		reset.ResetId,
	)
	if err != nil {
		return fmt.Errorf("failed to mark reset token as used: %w", err)
	}

	return tx.Commit()
}
