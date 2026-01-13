package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pos/api-gateway/utils"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	redis *redis.Client
}

func NewRateLimiter() *RateLimiter {
	redisHost := utils.GetEnv("REDIS_HOST")
	redisPass := utils.GetEnv("REDIS_PASSWORD")
	client := redis.NewClient(&redis.Options{
		Addr:         redisHost,
		Password:     redisPass,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	return &RateLimiter{redis: client}
}

func (rl *RateLimiter) IsRedisConnected() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := rl.redis.Ping(ctx).Result()
	return err == nil
}

func (rl *RateLimiter) RateLimit(maxAttempts int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := fmt.Sprintf("ratelimit:%s:%s", c.Path(), c.RealIP())

			ctx := context.Background()
			count, err := rl.redis.Get(ctx, key).Int()
			if err != nil && err != redis.Nil {
				c.Logger().Errorf("Redis error: %v", err)
				return next(c)
			}

			if count >= maxAttempts {
				ttl, _ := rl.redis.TTL(ctx, key).Result()
				c.Response().Header().Set("Retry-After", strconv.Itoa(int(ttl.Seconds())))
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error": "Rate limit exceeded. Please try again later.",
				})
			}

			pipe := rl.redis.Pipeline()
			pipe.Incr(ctx, key)
			if count == 0 {
				pipe.Expire(ctx, key, window)
			}
			_, err = pipe.Exec(ctx)
			if err != nil {
				c.Logger().Errorf("Redis pipeline error: %v", err)
			}

			return next(c)
		}
	}
}

func (rl *RateLimiter) LoginRateLimit() echo.MiddlewareFunc {
	return rl.RateLimit(5, 15*time.Minute)
}
