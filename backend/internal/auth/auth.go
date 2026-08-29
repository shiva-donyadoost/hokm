// Package auth implements password hashing and token management per
// ADR-0008: bcrypt passwords, short-lived JWT access tokens, opaque
// rotating refresh tokens (storage delegated to a RefreshStore).
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken = errors.New("auth: invalid token")
	ErrExpiredToken = errors.New("auth: token expired")
	ErrWeakPassword = errors.New("auth: password too weak")
)

// HashPassword hashes a password with bcrypt.
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", ErrWeakPassword
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("auth: bcrypt: %w", err)
	}
	return string(b), nil
}

// VerifyPassword reports whether the password matches the hash.
func VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// TokenManager issues and verifies JWT access tokens.
type TokenManager struct {
	secret    []byte
	accessTTL time.Duration
}

// NewTokenManager builds a manager from a secret and access TTL.
func NewTokenManager(secret string, accessTTL time.Duration) *TokenManager {
	return &TokenManager{secret: []byte(secret), accessTTL: accessTTL}
}

type accessClaims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// IssueAccess creates a signed access token for the user.
func (tm *TokenManager) IssueAccess(userID string) (string, error) {
	now := time.Now()
	claims := accessClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tm.accessTTL)),
			Issuer:    "hokm",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := tok.SignedString(tm.secret)
	if err != nil {
		return "", fmt.Errorf("auth: sign: %w", err)
	}
	return s, nil
}

// VerifyAccess validates a token and returns its user id.
func (tm *TokenManager) VerifyAccess(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(token, &accessClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return tm.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return "", ErrExpiredToken
		}
		return "", ErrInvalidToken
	}
	claims, ok := parsed.Claims.(*accessClaims)
	if !ok || !parsed.Valid || claims.UserID == "" {
		return "", ErrInvalidToken
	}
	return claims.UserID, nil
}

// NewRefreshToken returns a fresh opaque refresh token and its SHA-256 hex
// hash (only the hash is persisted).
func NewRefreshToken() (token string, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("auth: random: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}

// HashRefreshToken hashes an incoming refresh token for lookup.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// RefreshStore persists hashed refresh tokens.
type RefreshStore interface {
	// Save stores a hashed token for the user until expiry.
	Save(hash, userID string, expiresAt time.Time) error
	// Consume atomically removes and returns the owner of a hashed token.
	// Unknown or already-used tokens return ErrInvalidToken.
	Consume(hash string) (userID string, err error)
}
