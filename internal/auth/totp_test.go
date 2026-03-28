package auth

import (
	"encoding/base32"
	"encoding/binary"
	"crypto/hmac"
	"crypto/sha1"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

func TestGenerateTOTPSecret_Format(t *testing.T) {
	secret, url, err := GenerateTOTPSecret("FlowForge", "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if secret == "" {
		t.Error("secret should not be empty")
	}
	// Base32 encoded 20 bytes = 32 chars (no padding)
	if len(secret) != 32 {
		t.Errorf("secret length = %d, want 32", len(secret))
	}
	if !strings.HasPrefix(url, "otpauth://totp/") {
		t.Errorf("url should start with otpauth://totp/: %s", url)
	}
	if !strings.Contains(url, "FlowForge") {
		t.Errorf("url should contain issuer: %s", url)
	}
	if !strings.Contains(url, "user@example.com") {
		t.Errorf("url should contain account: %s", url)
	}
	if !strings.Contains(url, "algorithm=SHA1") {
		t.Errorf("url should contain algorithm: %s", url)
	}
	if !strings.Contains(url, "digits=6") {
		t.Errorf("url should contain digits: %s", url)
	}
	if !strings.Contains(url, "period=30") {
		t.Errorf("url should contain period: %s", url)
	}
}

func TestGenerateTOTPSecret_Unique(t *testing.T) {
	s1, _, _ := GenerateTOTPSecret("Issuer", "user1")
	s2, _, _ := GenerateTOTPSecret("Issuer", "user2")
	if s1 == s2 {
		t.Error("two secrets should differ")
	}
}

func TestValidateTOTP_WrongLength(t *testing.T) {
	if ValidateTOTP("JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP", "12345") {
		t.Error("should reject code with wrong length")
	}
	if ValidateTOTP("JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP", "1234567") {
		t.Error("should reject code with wrong length")
	}
}

func TestValidateTOTP_InvalidSecret(t *testing.T) {
	if ValidateTOTP("not-valid-base32!!!", "123456") {
		t.Error("should reject invalid base32 secret")
	}
}

func TestValidateTOTP_CurrentCode(t *testing.T) {
	// Generate a secret and compute the current TOTP code
	secret, _, err := GenerateTOTPSecret("FlowForge", "test@test.com")
	if err != nil {
		t.Fatal(err)
	}

	// Decode the secret
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		t.Fatal(err)
	}

	// Compute current code
	counter := time.Now().Unix() / 30
	code := computeTestTOTP(key, counter)

	if !ValidateTOTP(secret, code) {
		t.Errorf("current TOTP code %s should validate", code)
	}
}

func TestValidateTOTP_SkewWindow(t *testing.T) {
	secret, _, _ := GenerateTOTPSecret("FlowForge", "test@test.com")
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))

	counter := time.Now().Unix() / 30

	// Previous window code should also be valid (skew -1)
	prevCode := computeTestTOTP(key, counter-1)
	if !ValidateTOTP(secret, prevCode) {
		t.Error("previous window code should validate with skew=1")
	}

	// Next window code should also be valid (skew +1)
	nextCode := computeTestTOTP(key, counter+1)
	if !ValidateTOTP(secret, nextCode) {
		t.Error("next window code should validate with skew=1")
	}
}

// computeTestTOTP computes a TOTP code for testing, duplicating the algorithm
// from the source to provide an independent verification.
func computeTestTOTP(key []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	offset := hash[len(hash)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff
	otp := truncated % uint32(math.Pow10(6))

	return fmt.Sprintf("%06d", otp)
}
