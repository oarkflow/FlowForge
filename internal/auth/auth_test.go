package auth

import (
	"strings"
	"testing"
	"time"
)

// --- Password Hashing ---

func TestHashPassword_RoundTrip(t *testing.T) {
	password := "my-secure-password-123!"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
	if hash == password {
		t.Error("hash should not equal plaintext")
	}
	if !CheckPassword(hash, password) {
		t.Error("password should match its hash")
	}
}

func TestCheckPassword_WrongPassword(t *testing.T) {
	hash, _ := HashPassword("correct-password")
	if CheckPassword(hash, "wrong-password") {
		t.Error("wrong password should not match")
	}
}

func TestHashPassword_DifferentHashesForSamePassword(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")
	if h1 == h2 {
		t.Error("two hashes of the same password should differ (bcrypt uses random salt)")
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	if CheckPassword("not-a-valid-hash", "password") {
		t.Error("invalid hash should not match")
	}
}

// --- JWT Tokens ---

func TestGenerateAccessToken_Success(t *testing.T) {
	token, err := GenerateAccessToken("user-1", "user@example.com", "testuser", "admin", "test-secret-key", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Error("token should not be empty")
	}
}

func TestGenerateAccessToken_EmptySecret(t *testing.T) {
	_, err := GenerateAccessToken("user-1", "user@example.com", "testuser", "admin", "", 15*time.Minute)
	if err == nil {
		t.Error("should return error for empty secret")
	}
}

func TestParseToken_ValidToken(t *testing.T) {
	secret := "test-secret-key-12345"
	token, err := GenerateAccessToken("user-1", "user@test.com", "testuser", "admin", secret, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	claims, err := ParseToken(token, secret)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-1")
	}
	if claims.Email != "user@test.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "user@test.com")
	}
	if claims.Username != "testuser" {
		t.Errorf("Username = %q, want %q", claims.Username, "testuser")
	}
	if claims.Role != "admin" {
		t.Errorf("Role = %q, want %q", claims.Role, "admin")
	}
	if claims.Issuer != "flowforge" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "flowforge")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, _ := GenerateAccessToken("user-1", "email", "user", "admin", "secret1", 15*time.Minute)
	_, err := ParseToken(token, "secret2")
	if err == nil {
		t.Error("should return error for wrong secret")
	}
}

func TestParseToken_ExpiredToken(t *testing.T) {
	// Generate with a negative expiration so it's already expired
	token, _ := GenerateAccessToken("user-1", "email", "user", "admin", "secret", -1*time.Minute)
	_, err := ParseToken(token, "secret")
	if err == nil {
		t.Error("should return error for expired token")
	}
}

func TestParseToken_InvalidString(t *testing.T) {
	_, err := ParseToken("not.a.valid.token", "secret")
	if err == nil {
		t.Error("should return error for invalid token string")
	}
}

func TestGenerateRefreshToken_Success(t *testing.T) {
	token, err := GenerateRefreshToken("user-1", "refresh-secret", 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Error("refresh token should not be empty")
	}
}

func TestGenerateRefreshToken_EmptySecret(t *testing.T) {
	_, err := GenerateRefreshToken("user-1", "", 7*24*time.Hour)
	if err == nil {
		t.Error("should return error for empty secret")
	}
}

func TestParseRefreshToken_Valid(t *testing.T) {
	secret := "refresh-secret-12345"
	token, _ := GenerateRefreshToken("user-1", secret, 7*24*time.Hour)

	claims, err := ParseRefreshToken(token, secret)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want %q", claims.UserID, "user-1")
	}
	if claims.Issuer != "flowforge" {
		t.Errorf("Issuer = %q, want %q", claims.Issuer, "flowforge")
	}
}

func TestParseRefreshToken_WrongSecret(t *testing.T) {
	token, _ := GenerateRefreshToken("user-1", "secret-a", 7*24*time.Hour)
	_, err := ParseRefreshToken(token, "secret-b")
	if err == nil {
		t.Error("should return error for wrong secret")
	}
}

func TestParseRefreshToken_Invalid(t *testing.T) {
	_, err := ParseRefreshToken("garbage", "secret")
	if err == nil {
		t.Error("should return error for invalid string")
	}
}

// --- API Keys ---

func TestGenerateAPIKey_Format(t *testing.T) {
	key, hash, err := GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "ff_") {
		t.Errorf("key should have ff_ prefix: %s", key)
	}
	// ff_ prefix + 64 hex chars (32 bytes)
	if len(key) != 3+64 {
		t.Errorf("key length = %d, want %d", len(key), 3+64)
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
	if hash == key {
		t.Error("hash should differ from plaintext key")
	}
}

func TestGenerateAPIKey_Unique(t *testing.T) {
	key1, _, _ := GenerateAPIKey()
	key2, _, _ := GenerateAPIKey()
	if key1 == key2 {
		t.Error("two generated keys should differ")
	}
}

func TestValidateAPIKey_Correct(t *testing.T) {
	key, hash, _ := GenerateAPIKey()
	if !ValidateAPIKey(key, hash) {
		t.Error("valid key should validate against its hash")
	}
}

func TestValidateAPIKey_WrongKey(t *testing.T) {
	_, hash, _ := GenerateAPIKey()
	if ValidateAPIKey("ff_wrong_key_abcdef1234567890abcdef1234567890abcdef1234567890abcdef12", hash) {
		t.Error("wrong key should not validate")
	}
}

func TestHashAPIKey_Deterministic(t *testing.T) {
	key := "ff_test_key_1234"
	h1 := HashAPIKey(key)
	h2 := HashAPIKey(key)
	if h1 != h2 {
		t.Error("SHA-256 hash should be deterministic")
	}
}

func TestHashAPIKey_HexEncoded(t *testing.T) {
	hash := HashAPIKey("ff_test")
	// SHA-256 produces 32 bytes = 64 hex chars
	if len(hash) != 64 {
		t.Errorf("hash length = %d, want 64", len(hash))
	}
}
