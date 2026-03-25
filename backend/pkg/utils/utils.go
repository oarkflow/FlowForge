package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// StringPtr returns a pointer to the given string.
func StringPtr(s string) *string { return &s }

// IntPtr returns a pointer to the given int.
func IntPtr(i int) *int { return &i }

// Int64Ptr returns a pointer to the given int64.
func Int64Ptr(i int64) *int64 { return &i }

// BoolPtr returns a pointer to the given bool.
func BoolPtr(b bool) *bool { return &b }

// Deref dereferences a pointer, returning the zero value if nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

var slugRegexp = regexp.MustCompile(`[^a-z0-9-]`)
var slugMultiDash = regexp.MustCompile(`-{2,}`)

// Slugify converts a string to a URL-safe slug.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRegexp.ReplaceAllString(s, "-")
	s = slugMultiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "unnamed"
	}
	return s
}

// TruncateString truncates a string to max length, appending "..." if truncated.
func TruncateString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// RandomString generates a cryptographically secure random hex string of n bytes.
func RandomString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Contains checks if a slice contains an item.
func Contains[T comparable](slice []T, item T) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

// Unique returns a deduplicated copy of a slice.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{}, len(slice))
	result := make([]T, 0, len(slice))
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// Map applies a function to each element of a slice.
func Map[T, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

// Filter returns elements of a slice that satisfy a predicate.
func Filter[T any](slice []T, fn func(T) bool) []T {
	result := make([]T, 0)
	for _, v := range slice {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

// Paginate converts page/perPage to offset/limit for SQL queries.
func Paginate(page, perPage int) (offset, limit int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return (page - 1) * perPage, perPage
}

// MergeMaps merges multiple maps, with later maps taking precedence.
func MergeMaps[K comparable, V any](maps ...map[K]V) map[K]V {
	result := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// ParseDuration parses a human-readable duration string.
// Supports: "30s", "5m", "2h", "1d", "1w".
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}

	// Try standard Go duration first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Parse custom formats (d for days, w for weeks)
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	unit := s[len(s)-1]
	numStr := strings.TrimSpace(s[:len(s)-1])
	var num float64
	if _, err := fmt.Sscanf(numStr, "%f", &num); err != nil {
		return 0, fmt.Errorf("invalid duration number: %s", numStr)
	}

	switch unit {
	case 'd', 'D':
		return time.Duration(num * float64(24*time.Hour)), nil
	case 'w', 'W':
		return time.Duration(num * float64(7*24*time.Hour)), nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %c", unit)
	}
}

// FormatDuration formats a duration as a human-readable string.
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm %ds", m, s)
	}

	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

// CamelToSnake converts a CamelCase string to snake_case.
func CamelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result.WriteRune('_')
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Retry executes fn up to maxAttempts times with exponential backoff.
func Retry(maxAttempts int, initialDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := initialDelay

	for i := range maxAttempts {
		if err := fn(); err != nil {
			lastErr = err
			if i < maxAttempts-1 {
				time.Sleep(delay)
				delay *= 2
			}
		} else {
			return nil
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}
