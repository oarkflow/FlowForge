package middleware

import (
	"testing"
	"time"
)

// --- RBAC roleHierarchy ---

func TestRoleHierarchy_Values(t *testing.T) {
	tests := []struct {
		role string
		want int
	}{
		{"owner", 4},
		{"admin", 3},
		{"developer", 2},
		{"viewer", 1},
	}
	for _, tt := range tests {
		if got := roleHierarchy[tt.role]; got != tt.want {
			t.Errorf("roleHierarchy[%q] = %d, want %d", tt.role, got, tt.want)
		}
	}
}

func TestRoleHierarchy_UnknownRole(t *testing.T) {
	if v := roleHierarchy["unknown"]; v != 0 {
		t.Errorf("unknown role should have 0 level, got %d", v)
	}
}

// --- UserRateLimiter (unit tests on non-HTTP logic) ---

func TestDefaultRoleLimits(t *testing.T) {
	rl := DefaultRoleLimits()
	if rl.Owner != 2000 {
		t.Errorf("Owner = %d, want 2000", rl.Owner)
	}
	if rl.Admin != 1000 {
		t.Errorf("Admin = %d, want 1000", rl.Admin)
	}
	if rl.Developer != 200 {
		t.Errorf("Developer = %d, want 200", rl.Developer)
	}
	if rl.Viewer != 100 {
		t.Errorf("Viewer = %d, want 100", rl.Viewer)
	}
}

func TestNewUserRateLimiter_DefaultWindow(t *testing.T) {
	rl := NewUserRateLimiter(DefaultRoleLimits(), 0)
	if rl.window != time.Minute {
		t.Errorf("window = %v, want 1m (default)", rl.window)
	}
}

func TestNewUserRateLimiter_CustomWindow(t *testing.T) {
	rl := NewUserRateLimiter(DefaultRoleLimits(), 5*time.Minute)
	if rl.window != 5*time.Minute {
		t.Errorf("window = %v, want 5m", rl.window)
	}
}

func TestUserRateLimiter_GetLimitForRole(t *testing.T) {
	rl := NewUserRateLimiter(DefaultRoleLimits(), time.Minute)

	tests := []struct {
		role string
		want int
	}{
		{"owner", 2000},
		{"admin", 1000},
		{"developer", 200},
		{"viewer", 100},
		{"unknown", 100}, // defaults to viewer
		{"", 100},
	}
	for _, tt := range tests {
		got := rl.getLimitForRole(tt.role)
		if got != tt.want {
			t.Errorf("getLimitForRole(%q) = %d, want %d", tt.role, got, tt.want)
		}
	}
}

func TestUserRateLimiter_Cleanup(t *testing.T) {
	rl := NewUserRateLimiter(DefaultRoleLimits(), 50*time.Millisecond)

	// Manually add an expired bucket
	rl.mu.Lock()
	rl.buckets["user-1"] = &userBucket{
		remaining: 10,
		limit:     100,
		resetAt:   time.Now().Add(-time.Second), // already expired
	}
	rl.buckets["user-2"] = &userBucket{
		remaining: 50,
		limit:     100,
		resetAt:   time.Now().Add(time.Hour), // not expired
	}
	rl.mu.Unlock()

	rl.Cleanup()

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if _, exists := rl.buckets["user-1"]; exists {
		t.Error("expired bucket for user-1 should be cleaned up")
	}
	if _, exists := rl.buckets["user-2"]; !exists {
		t.Error("non-expired bucket for user-2 should be kept")
	}
}

func TestMax(t *testing.T) {
	if max(3, 5) != 5 {
		t.Error("max(3,5) should be 5")
	}
	if max(10, 2) != 10 {
		t.Error("max(10,2) should be 10")
	}
	if max(4, 4) != 4 {
		t.Error("max(4,4) should be 4")
	}
	if max(-1, 0) != 0 {
		t.Error("max(-1,0) should be 0")
	}
}

// --- Validate ---

func TestValidate_ValidStruct(t *testing.T) {
	type Input struct {
		Email string `validate:"required,email"`
		Name  string `validate:"required,min=3"`
	}
	err := Validate(&Input{Email: "test@example.com", Name: "Alice"})
	if err != nil {
		t.Errorf("valid struct should pass: %v", err)
	}
}

func TestValidate_InvalidStruct(t *testing.T) {
	type Input struct {
		Email string `validate:"required,email"`
	}
	err := Validate(&Input{Email: "not-an-email"})
	if err == nil {
		t.Error("invalid email should fail validation")
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	type Input struct {
		Name string `validate:"required"`
	}
	err := Validate(&Input{Name: ""})
	if err == nil {
		t.Error("missing required field should fail")
	}
}

// --- normalizeCIDR & extractIP (from ip_allowlist.go) ---

func TestNormalizeCIDR(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"10.0.0.1", "10.0.0.1/32"},
		{"10.0.0.0/8", "10.0.0.0/8"},
		{"::1", "::1/128"},
		{"2001:db8::/32", "2001:db8::/32"},
		{"  192.168.1.1  ", "192.168.1.1/32"},
	}
	for _, tt := range tests {
		got := normalizeCIDR(tt.input)
		if got != tt.want {
			t.Errorf("normalizeCIDR(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"192.168.1.1:8080", "192.168.1.1"},
		{"192.168.1.1", "192.168.1.1"},
		{"[::1]:443", "::1"},
		{"::1", "::1"},
	}
	for _, tt := range tests {
		got := extractIP(tt.input)
		if got != tt.want {
			t.Errorf("extractIP(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- statusText (from errors.go via Claims struct check) ---

func TestClaims_Fields(t *testing.T) {
	c := Claims{
		UserID:   "u-1",
		Email:    "test@test.com",
		Username: "tester",
		Role:     "admin",
	}
	if c.UserID != "u-1" {
		t.Error("UserID not set")
	}
	if c.Role != "admin" {
		t.Error("Role not set")
	}
}
