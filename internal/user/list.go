package user

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func List(ctx context.Context, db *sqlx.DB) ([]User, error) {
	var users []User
	err := db.SelectContext(ctx, &users, `SELECT * FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}
