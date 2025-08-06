package users

import (
	"crypto/rand"
	"crypto/subtle"
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
	UserId    int64      `db:"user_id" json:"user_id"`
	FirstName string     `db:"first_name" json:"first_name"`
	LastName  string     `db:"last_name" json:"last_name"`
	Email     string     `db:"email" json:"email"`
	Hash      string     `db:"password_hash" json:"password_hash"`
	Salt      []byte     `db:"salt" json:"salt"`
	LastLogin *time.Time `db:"last_login" json:"last_login"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

type CreateRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
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

func ByEmail(db *sqlx.DB, email string) (*User, error) {
	var u User
	err := db.Get(&u, `SELECT * FROM users WHERE email = $1`, email)
	if err != nil {
		return nil, errors.New("failed to get user by email")
	}
	return &u, nil
}

func Exists(db *sqlx.DB, email string) bool {
	var exists bool
	if err := db.Get(&exists, "SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)", email); err != nil {
		return false
	}
	return exists
}

func Create(db *sqlx.DB, request CreateRequest) (int64, error) {
	h, s, err := hash(request.Password, nil)
	if err != nil {
		return 0, errors.Join(errors.New("failed to hash password"), err)
	}
	var uid int64
	err = db.QueryRow(
		`INSERT INTO users (first_name, last_name, email, password_hash, salt) VALUES ($1, $2, $3, $4, $5) RETURNING user_id`,
		request.FirstName, request.LastName, request.Email, h, s,
	).Scan(&uid)
	if err != nil {
		return 0, errors.Join(errors.New("failed to create user"), err)
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
			return "", nil, errors.Join(errors.New("failed to generate salt"), err)
		}
	}

	hash, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		return "", nil, errors.Join(errors.New("failed to create key"), err)
	}
	return base64.StdEncoding.EncodeToString(hash), salt, nil
}
