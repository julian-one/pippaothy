package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type UpdateRequest struct {
	Username *string `json:"username,omitempty"`
	Email    *string `json:"email,omitempty"`
	Password *string `json:"password,omitempty"`
}

func Update(ctx context.Context, db *sqlx.DB, userID int64, request UpdateRequest) error {
	updates := []string{}
	args := []interface{}{}
	argPosition := 1

	if request.Username != nil {
		updates = append(updates, fmt.Sprintf("username = $%d", argPosition))
		args = append(args, *request.Username)
		argPosition++
	}

	if request.Email != nil {
		updates = append(updates, fmt.Sprintf("email = $%d", argPosition))
		args = append(args, *request.Email)
		argPosition++
	}

	if request.Password != nil {
		h, s, err := hash(*request.Password, nil)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		updates = append(updates, fmt.Sprintf("password_hash = $%d", argPosition))
		args = append(args, h)
		argPosition++
		updates = append(updates, fmt.Sprintf("salt = $%d", argPosition))
		args = append(args, s)
		argPosition++
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Always update the updated_at timestamp
	updates = append(updates, "updated_at = CURRENT_TIMESTAMP")

	// Add userID as the final argument for WHERE clause
	args = append(args, userID)

	query := fmt.Sprintf(
		"UPDATE users SET %s WHERE user_id = $%d",
		strings.Join(updates, ", "),
		argPosition,
	)

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
