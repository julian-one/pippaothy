package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func Blacklist(ctx context.Context, c *redis.Client, jti string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", jti)
	return c.Set(ctx, key, "1", ttl).Err()
}

func IsBlacklisted(ctx context.Context, c *redis.Client, jti string) (bool, error) {
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

func StoreRefresh(
	ctx context.Context,
	c *redis.Client,
	tokenID string,
	userID int64,
	ttl time.Duration,
) error {
	key := fmt.Sprintf("refresh:%s", tokenID)
	userKey := fmt.Sprintf("user_tokens:%d", userID)

	pipe := c.Pipeline()
	pipe.Set(ctx, key, userID, ttl)
	pipe.SAdd(ctx, userKey, tokenID)
	pipe.Expire(ctx, userKey, ttl)

	_, err := pipe.Exec(ctx)
	return err
}

func GetRefresh(ctx context.Context, c *redis.Client, tokenID string) (int64, error) {
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

func DeleteRefresh(ctx context.Context, c *redis.Client, tokenID string, userID int64) error {
	key := fmt.Sprintf("refresh:%s", tokenID)
	userKey := fmt.Sprintf("user_tokens:%d", userID)

	pipe := c.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, userKey, tokenID)

	_, err := pipe.Exec(ctx)
	return err
}

func DeleteUserRefresh(ctx context.Context, c *redis.Client, userID int64) error {
	userKey := fmt.Sprintf("user_tokens:%d", userID)

	tokenIDs, err := c.SMembers(ctx, userKey).Result()
	if err != nil && err != redis.Nil {
		return err
	}

	if len(tokenIDs) == 0 {
		return nil
	}

	pipe := c.Pipeline()
	for _, tokenID := range tokenIDs {
		key := fmt.Sprintf("refresh:%s", tokenID)
		pipe.Del(ctx, key)
	}
	pipe.Del(ctx, userKey)

	_, err = pipe.Exec(ctx)
	return err
}
