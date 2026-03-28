package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims represents the JWT claims used throughout the application.
// This mirrors the Claims struct in api/middleware/auth.go but lives here
// as the canonical definition for token generation and parsing.
type Claims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// RefreshClaims holds minimal claims for refresh tokens.
type RefreshClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// HashPassword hashes a plaintext password using bcrypt with the default cost.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a bcrypt hash with a plaintext password.
// Returns true if they match.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateAccessToken creates a signed JWT access token containing user identity and role.
func GenerateAccessToken(userID, email, username, role, secret string, expiration time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret must not be empty")
	}

	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Email:    email,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			Issuer:    "flowforge",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken creates a signed JWT refresh token with minimal claims.
func GenerateRefreshToken(userID, secret string, expiration time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret must not be empty")
	}

	now := time.Now()
	claims := RefreshClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			Issuer:    "flowforge",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken validates and parses a JWT token string, returning the embedded Claims.
func ParseToken(tokenStr, secret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ParseRefreshToken validates and parses a refresh JWT token string.
func ParseRefreshToken(tokenStr, secret string) (*RefreshClaims, error) {
	claims := &RefreshClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
