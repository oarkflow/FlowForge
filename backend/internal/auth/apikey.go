package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

const (
	// apiKeyLength is the number of random bytes in a generated API key.
	apiKeyLength = 32
	// apiKeyPrefix is prepended to all generated keys for easy identification.
	apiKeyPrefix = "ff_"
)

// GenerateAPIKey generates a cryptographically random API key and its SHA-256 hash.
// Returns the plaintext key (to show to the user once) and the hash (to store in DB).
func GenerateAPIKey() (plainKey string, hash string, err error) {
	b := make([]byte, apiKeyLength)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate api key: %w", err)
	}
	plainKey = apiKeyPrefix + hex.EncodeToString(b)
	hash = HashAPIKey(plainKey)
	return plainKey, hash, nil
}

// HashAPIKey computes the SHA-256 hash of an API key for storage.
func HashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// ValidateAPIKey compares a plaintext key against a stored SHA-256 hash
// using constant-time comparison to prevent timing attacks.
func ValidateAPIKey(key, hash string) bool {
	computed := HashAPIKey(key)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(hash)) == 1
}
