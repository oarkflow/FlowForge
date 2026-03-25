package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	// totpSecretLength is the number of random bytes for the TOTP secret.
	totpSecretLength = 20
	// totpDigits is the number of digits in the TOTP code.
	totpDigits = 6
	// totpPeriod is the time step in seconds (standard: 30).
	totpPeriod = 30
	// totpSkew allows validating codes from adjacent time windows.
	totpSkew = 1
)

// GenerateTOTPSecret generates a new TOTP secret and returns the base32-encoded
// secret along with the otpauth:// URI for QR code generation.
func GenerateTOTPSecret(issuer, account string) (secret string, otpauthURL string, err error) {
	b := make([]byte, totpSecretLength)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate totp secret: %w", err)
	}

	secret = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
	otpauthURL = fmt.Sprintf(
		"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		issuer, account, secret, issuer, totpDigits, totpPeriod,
	)
	return secret, otpauthURL, nil
}

// ValidateTOTP validates a TOTP code against the given base32-encoded secret.
// It checks the current time window plus/minus the skew value (±1 window by default)
// to account for clock drift.
func ValidateTOTP(secret, code string) bool {
	if len(code) != totpDigits {
		return false
	}

	// Decode the base32 secret
	secret = strings.TrimRight(strings.ToUpper(secret), "=")
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return false
	}

	now := time.Now().Unix()
	counter := now / totpPeriod

	// Check current window and adjacent windows
	for i := -int64(totpSkew); i <= int64(totpSkew); i++ {
		computed := generateTOTPCode(key, counter+i)
		if computed == code {
			return true
		}
	}

	return false
}

// generateTOTPCode computes a TOTP code for the given secret key and counter value
// using HMAC-SHA1 as defined in RFC 4226 and RFC 6238.
func generateTOTPCode(key []byte, counter int64) string {
	// Convert counter to big-endian 8-byte buffer
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))

	// HMAC-SHA1
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)

	// Dynamic truncation (RFC 4226 Section 5.4)
	offset := hash[len(hash)-1] & 0x0f
	truncated := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7fffffff

	// Compute the OTP value
	otp := truncated % uint32(math.Pow10(totpDigits))

	return fmt.Sprintf("%0*d", totpDigits, otp)
}
