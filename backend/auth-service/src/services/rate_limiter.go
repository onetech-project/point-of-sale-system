package services

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type RateLimiter struct {
	redis         *redis.Client
	maxAttempts   int
	windowSeconds int
}

func NewRateLimiter(redisClient *redis.Client, maxAttempts, windowSeconds int) *RateLimiter {
	return &RateLimiter{
		redis:         redisClient,
		maxAttempts:   maxAttempts,
		windowSeconds: windowSeconds,
	}
}

// CheckLoginLimit checks if login attempts are within the allowed limit
func (rl *RateLimiter) CheckLoginLimit(ctx context.Context, email, tenantID string) (bool, int, error) {
	key := fmt.Sprintf("ratelimit:login:%s:%s", email, tenantID)

	// Get current count
	count, err := rl.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		// No previous attempts
		return true, rl.maxAttempts, nil
	}
	if err != nil {
		return false, 0, fmt.Errorf("failed to check rate limit: %w", err)
	}

	remaining := rl.maxAttempts - count
	if remaining <= 0 {
		return false, 0, nil
	}

	return true, remaining, nil
}

// IncrementLoginAttempts increments the login attempt counter
func (rl *RateLimiter) IncrementLoginAttempts(ctx context.Context, email, tenantID string) error {
	key := fmt.Sprintf("ratelimit:login:%s:%s", email, tenantID)

	pipe := rl.redis.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, time.Duration(rl.windowSeconds)*time.Second)
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to increment login attempts: %w", err)
	}

	// Check if this is the first attempt
	if incr.Val() == 1 {
		// Set TTL for the key
		err = rl.redis.Expire(ctx, key, time.Duration(rl.windowSeconds)*time.Second).Err()
		if err != nil {
			return fmt.Errorf("failed to set TTL for rate limit key: %w", err)
		}
	}

	return nil
}

// ResetLoginAttempts resets the login attempt counter (on successful login)
func (rl *RateLimiter) ResetLoginAttempts(ctx context.Context, email, tenantID string) error {
	key := fmt.Sprintf("ratelimit:login:%s:%s", email, tenantID)

	err := rl.redis.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to reset login attempts: %w", err)
	}

	return nil
}

// GetRemainingTime returns the remaining time before rate limit resets
func (rl *RateLimiter) GetRemainingTime(ctx context.Context, email, tenantID string) (time.Duration, error) {
	key := fmt.Sprintf("ratelimit:login:%s:%s", email, tenantID)

	ttl, err := rl.redis.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get rate limit TTL: %w", err)
	}

	if ttl < 0 {
		return 0, nil
	}

	return ttl, nil
}

// GetAttemptCount returns the current attempt count
func (rl *RateLimiter) GetAttemptCount(ctx context.Context, email, tenantID string) (int, error) {
	key := fmt.Sprintf("ratelimit:login:%s:%s", email, tenantID)

	count, err := rl.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get attempt count: %w", err)
	}

	return count, nil
}
