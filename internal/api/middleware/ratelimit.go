package middleware

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

// RateLimiter creates a basic per-IP rate limiter.
func RateLimiter(max int, expiration time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: expiration,
	})
}

// RoleLimits defines rate limits per user role.
type RoleLimits struct {
	Owner     int
	Admin     int
	Developer int
	Viewer    int
}

// DefaultRoleLimits returns the default rate limits per role (requests per minute).
func DefaultRoleLimits() RoleLimits {
	return RoleLimits{
		Owner:     2000,
		Admin:     1000,
		Developer: 200,
		Viewer:    100,
	}
}

// userBucket tracks rate limit state for a single user.
type userBucket struct {
	remaining int
	limit     int
	resetAt   time.Time
}

// UserRateLimiter provides per-user rate limiting with role-based limits.
type UserRateLimiter struct {
	limits  RoleLimits
	window  time.Duration
	buckets map[string]*userBucket
	mu      sync.Mutex
}

// NewUserRateLimiter creates a per-user rate limiter with role-based limits.
func NewUserRateLimiter(limits RoleLimits, window time.Duration) *UserRateLimiter {
	if window <= 0 {
		window = time.Minute
	}
	return &UserRateLimiter{
		limits:  limits,
		window:  window,
		buckets: make(map[string]*userBucket),
	}
}

// Handler returns a Fiber middleware handler for per-user rate limiting.
func (rl *UserRateLimiter) Handler() fiber.Handler {
	return func(c fiber.Ctx) error {
		userID, _ := c.Locals("user_id").(string)
		if userID == "" {
			// Not authenticated — fall through to IP-based limiter
			return c.Next()
		}

		role, _ := c.Locals("role").(string)
		limit := rl.getLimitForRole(role)

		rl.mu.Lock()
		bucket, exists := rl.buckets[userID]
		now := time.Now()

		if !exists || now.After(bucket.resetAt) {
			bucket = &userBucket{
				remaining: limit,
				limit:     limit,
				resetAt:   now.Add(rl.window),
			}
			rl.buckets[userID] = bucket
		}

		bucket.remaining--
		remaining := bucket.remaining
		resetAt := bucket.resetAt
		rl.mu.Unlock()

		// Set rate limit headers
		c.Set("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Set("X-RateLimit-Remaining", strconv.Itoa(max(remaining, 0)))
		c.Set("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))

		if remaining < 0 {
			retryAfter := time.Until(resetAt).Seconds()
			c.Set("Retry-After", fmt.Sprintf("%.0f", retryAfter))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error":       "rate limit exceeded",
				"retry_after": retryAfter,
			})
		}

		return c.Next()
	}
}

func (rl *UserRateLimiter) getLimitForRole(role string) int {
	switch role {
	case "owner":
		return rl.limits.Owner
	case "admin":
		return rl.limits.Admin
	case "developer":
		return rl.limits.Developer
	case "viewer":
		return rl.limits.Viewer
	default:
		return rl.limits.Viewer
	}
}

// Cleanup removes expired buckets. Call periodically to prevent memory leaks.
func (rl *UserRateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, bucket := range rl.buckets {
		if now.After(bucket.resetAt) {
			delete(rl.buckets, key)
		}
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
