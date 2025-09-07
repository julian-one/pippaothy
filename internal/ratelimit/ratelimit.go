package ratelimit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type RateLimiter struct {
	db *sqlx.DB
}

func New(db *sqlx.DB) *RateLimiter {
	return &RateLimiter{db: db}
}

func (r *RateLimiter) CheckPasswordReset(ctx context.Context, email string, ip string) error {
	now := time.Now()
	hourAgo := now.Add(-time.Hour)

	var emailCount int
	err := r.db.GetContext(ctx, &emailCount, `
		SELECT COUNT(*) FROM password_reset_attempts 
		WHERE email = $1 AND created_at > $2`,
		email, hourAgo,
	)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check email rate limit: %w", err)
	}

	if emailCount >= 3 {
		return fmt.Errorf("too many password reset attempts for this email. Please try again later")
	}

	var ipCount int
	err = r.db.GetContext(ctx, &ipCount, `
		SELECT COUNT(*) FROM password_reset_attempts 
		WHERE ip_address = $1 AND created_at > $2`,
		ip, hourAgo,
	)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check IP rate limit: %w", err)
	}

	if ipCount >= 10 {
		return fmt.Errorf("too many password reset attempts from this IP address. Please try again later")
	}

	return nil
}

func (r *RateLimiter) RecordPasswordResetAttempt(ctx context.Context, email string, ip string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO password_reset_attempts (email, ip_address, created_at) 
		VALUES ($1, $2, $3)`,
		email, ip, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to record password reset attempt: %w", err)
	}

	return nil
}

func (r *RateLimiter) CleanupOldAttempts(ctx context.Context) error {
	dayAgo := time.Now().Add(-24 * time.Hour)
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM password_reset_attempts 
		WHERE created_at < $1`,
		dayAgo,
	)
	if err != nil {
		return fmt.Errorf("failed to cleanup old attempts: %w", err)
	}

	return nil
}