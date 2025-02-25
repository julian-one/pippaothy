package users

import (
	"errors"
	"github.com/jmoiron/sqlx"
	"time"
)

type User struct {
	UserId       int        `db:"user_id" json:"user_id"`
	FirstName    string     `db:"first_name" json:"first_name"`
	LastName     string     `db:"last_name" json:"last_name"`
	Email        string     `db:"email" json:"email"`
	PasswordHash string     `db:"password_hash" json:"password_hash"`
	LastLogin    *time.Time `db:"last_login" json:"last_login"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

type CreateRequst struct {
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

func ById(db *sqlx.DB, id string) (*User, error) {
	var u User
	err := db.Get(&u, `SELECT * FROM users WHERE user_id = ?`, id)
	if err != nil {
		return nil, errors.Join(errors.New("failed to get user"), err)
	}
	return &u, nil
}

func List(db *sqlx.DB) ([]User, error) {
	ul := make([]User, 0)
	err := db.Select(&ul, `SELECT * FROM users`)
	if err != nil {
		return nil, errors.Join(errors.New("failed to list users"), err)
	}
	return ul, nil
}

func Create(db *sqlx.DB, request CreateRequst) error {
	_, err := db.Exec(`INSERT INTO users (first_name, last_name, email, password_hash) VALUES (?, ?, ?, ?)`,
		request.FirstName, request.LastName, request.Email, request.PasswordHash)
	if err != nil {
		return errors.Join(errors.New("failed to create user"), err)
	}
	return nil
}
