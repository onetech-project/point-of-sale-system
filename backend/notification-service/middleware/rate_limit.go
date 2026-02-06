package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pos/notification-service/src/utils"
	"golang.org/x/time/rate"
)

// RateLimiter stores rate limiters per IP
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

var limiter *RateLimiter

// InitRateLimiter initializes the rate limiter
func InitRateLimiter() {
	ratePerMinute := utils.GetEnvInt("RATE_LIMIT_REQUESTS_PER_MINUTE")
	burst := utils.GetEnvInt("RATE_LIMIT_BURST")

	limiter = &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(float64(ratePerMinute) / 60.0), // Convert to per-second
		burst:    burst,
	}

	// Cleanup old limiters periodically
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()
}

// RateLimit middleware limits requests per IP
func RateLimit() echo.MiddlewareFunc {
	if utils.GetEnv("RATE_LIMIT_ENABLED") == "false" {
		// Rate limiting disabled, pass through
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	if limiter == nil {
		InitRateLimiter()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			if !limiter.allow(ip) {
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded")
			}

			return next(c)
		}
	}
}

// RateLimitForTestNotifications creates a more restrictive rate limiter for test notifications
// Default: 5 requests per minute per IP to prevent abuse
func RateLimitForTestNotifications() echo.MiddlewareFunc {
	if utils.GetEnv("RATE_LIMIT_ENABLED") == "false" {
		// Rate limiting disabled, pass through
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	ratePerMinute := utils.GetEnvInt("TEST_NOTIFICATION_RATE_LIMIT")
	burst := utils.GetEnvInt("TEST_NOTIFICATION_BURST")

	testLimiter := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(float64(ratePerMinute) / 60.0), // Convert to per-second
		burst:    burst,
	}

	// Cleanup old limiters periodically
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			testLimiter.cleanup()
		}
	}()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()

			if !testLimiter.allow(ip) {
				return echo.NewHTTPError(http.StatusTooManyRequests, "rate limit exceeded for test notifications")
			}

			return next(c)
		}
	}
}

// allow checks if the request from this IP is allowed
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.RLock()
	limiter, exists := rl.limiters[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[ip] = limiter
		rl.mu.Unlock()
	}

	return limiter.Allow()
}

// cleanup removes old limiters (simple cleanup strategy)
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Clear all limiters periodically (simple approach)
	// In production, track last access time and remove stale ones
	if len(rl.limiters) > 10000 {
		rl.limiters = make(map[string]*rate.Limiter)
	}
}
