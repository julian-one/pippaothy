package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	*redis.Client
}

type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// New creates a new Redis client
func New(ctx context.Context, cfg Config) (*Client, error) {
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// TODO: figure out a way to not duplicate context with timeout
	// Test connection
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Client{Client: rdb}, nil
}

// BlacklistToken adds a token to the blacklist with TTL
func (c *Client) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", jti)
	return c.Set(ctx, key, "1", ttl).Err()
}

// IsTokenBlacklisted checks if a token is blacklisted
func (c *Client) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", jti)
	val, err := c.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

// StoreRefreshToken stores a refresh token with automatic expiration
// Maps: refresh:{tokenID} -> userID
// Also maintains a set of tokens per user for batch deletion
func (c *Client) StoreRefreshToken(
	ctx context.Context,
	tokenID string,
	userID int64,
	ttl time.Duration,
) error {
	key := fmt.Sprintf("refresh:%s", tokenID)
	userKey := fmt.Sprintf("user_tokens:%d", userID)

	pipe := c.Pipeline()

	// Store token -> userID mapping with TTL
	pipe.Set(ctx, key, userID, ttl)

	// Add token to user's set of tokens (for batch deletion on logout)
	pipe.SAdd(ctx, userKey, tokenID)
	pipe.Expire(ctx, userKey, ttl)

	_, err := pipe.Exec(ctx)
	return err
}

// GetRefreshToken retrieves the user ID associated with a refresh token
func (c *Client) GetRefreshToken(ctx context.Context, tokenID string) (int64, error) {
	key := fmt.Sprintf("refresh:%s", tokenID)
	val, err := c.Get(ctx, key).Int64()
	if err == redis.Nil {
		return 0, fmt.Errorf("refresh token not found or expired")
	}
	if err != nil {
		return 0, err
	}
	return val, nil
}

// DeleteRefreshToken removes a specific refresh token
func (c *Client) DeleteRefreshToken(ctx context.Context, tokenID string, userID int64) error {
	key := fmt.Sprintf("refresh:%s", tokenID)
	userKey := fmt.Sprintf("user_tokens:%d", userID)

	pipe := c.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, userKey, tokenID)

	_, err := pipe.Exec(ctx)
	return err
}

// DeleteUserRefreshTokens removes all refresh tokens for a user
func (c *Client) DeleteUserRefreshTokens(ctx context.Context, userID int64) error {
	userKey := fmt.Sprintf("user_tokens:%d", userID)

	// Get all token IDs for this user
	tokenIDs, err := c.SMembers(ctx, userKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if len(tokenIDs) == 0 {
		return nil
	}

	// Delete all tokens
	pipe := c.Pipeline()
	for _, tokenID := range tokenIDs {
		key := fmt.Sprintf("refresh:%s", tokenID)
		pipe.Del(ctx, key)
	}
	pipe.Del(ctx, userKey) // Delete the set itself

	_, err = pipe.Exec(ctx)
	return err
}
