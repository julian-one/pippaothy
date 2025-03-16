package users

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"time"

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

func ById(db *sqlx.DB, id string) (*User, error) {
	var u User
	err := db.Get(&u, `SELECT * FROM users WHERE user_id = $1`, id)
	if err != nil {
		return nil, errors.Join(errors.New("failed to get user"), err)
	}
	return &u, nil
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

func List(db *sqlx.DB) ([]User, error) {
	ul := make([]User, 0)
	err := db.Select(&ul, `SELECT * FROM users`)
	if err != nil {
		return nil, errors.Join(errors.New("failed to list users"), err)
	}
	return ul, nil
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
