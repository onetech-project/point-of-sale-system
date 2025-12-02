package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Cleanup old entries every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, timestamps := range rl.requests {
		// Remove timestamps outside the window
		valid := []time.Time{}
		for _, ts := range timestamps {
			if now.Sub(ts) < rl.window {
				valid = append(valid, ts)
			}
		}

		if len(valid) == 0 {
			delete(rl.requests, key)
		} else {
			rl.requests[key] = valid
		}
	}
}

func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get request timestamps for this identifier
	timestamps := rl.requests[identifier]

	// Remove timestamps outside the window
	valid := []time.Time{}
	for _, ts := range timestamps {
		if now.Sub(ts) < rl.window {
			valid = append(valid, ts)
		}
	}

	// Check if limit exceeded
	if len(valid) >= rl.limit {
		return false
	}

	// Add current timestamp
	valid = append(valid, now)
	rl.requests[identifier] = valid

	return true
}

// RateLimitMiddleware limits the number of requests per IP address
// Default: 100 requests per minute
func RateLimitMiddleware(limiter *RateLimiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Use IP address as identifier
			ip := c.RealIP()

			if !limiter.Allow(ip) {
				return c.JSON(http.StatusTooManyRequests, map[string]interface{}{
					"error":   "Rate limit exceeded",
					"message": "Too many requests, please try again later",
				})
			}

			return next(c)
		}
	}
}
