package user

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func Exists(ctx context.Context, db *sqlx.DB, email string) bool {
	var exists bool
	if err := db.GetContext(ctx, &exists, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email); err != nil {
		return false
	}
	return exists
}

func ByEmail(ctx context.Context, db *sqlx.DB, email string) (*User, error) {
	var u User
	err := db.GetContext(ctx, &u, `SELECT * FROM users WHERE email = $1`, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

func ByID(ctx context.Context, db *sqlx.DB, userID int64) (*User, error) {
	var u User
	err := db.GetContext(ctx, &u, `SELECT * FROM users WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &u, nil
}
