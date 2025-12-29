package user

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func ByEmail(ctx context.Context, db *sqlx.DB, email string) (*User, error) {
	var u User
	err := db.GetContext(ctx, &u, `SELECT * FROM users WHERE email = ?`, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &u, nil
}

func ByID(ctx context.Context, db *sqlx.DB, userID int64) (*User, error) {
	var u User
	err := db.GetContext(ctx, &u, `SELECT * FROM users WHERE user_id = ?`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &u, nil
}
