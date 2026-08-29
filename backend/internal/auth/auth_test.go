package auth

import (
	"testing"
	"time"
)

func TestPasswordHashRoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !VerifyPassword(h, "correct horse battery") {
		t.Error("correct password rejected")
	}
	if VerifyPassword(h, "wrong") {
		t.Error("wrong password accepted")
	}
	if _, err := HashPassword("short"); err == nil {
		t.Error("weak password accepted")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	tm := NewTokenManager("test-secret-value-at-least-long", time.Minute)
	tok, err := tm.IssueAccess("user-1")
	if err != nil {
		t.Fatalf("IssueAccess: %v", err)
	}
	uid, err := tm.VerifyAccess(tok)
	if err != nil {
		t.Fatalf("VerifyAccess: %v", err)
	}
	if uid != "user-1" {
		t.Fatalf("uid = %q", uid)
	}
}

func TestAccessTokenExpiry(t *testing.T) {
	tm := NewTokenManager("test-secret-value-at-least-long", -time.Minute)
	tok, _ := tm.IssueAccess("user-1")
	if _, err := tm.VerifyAccess(tok); err != ErrExpiredToken {
		t.Fatalf("want ErrExpiredToken, got %v", err)
	}
}

func TestAccessTokenWrongSecret(t *testing.T) {
	a := NewTokenManager("secret-one-aaaaaaaaaaaaa", time.Minute)
	b := NewTokenManager("secret-two-bbbbbbbbbbbb", time.Minute)
	tok, _ := a.IssueAccess("u")
	if _, err := b.VerifyAccess(tok); err != ErrInvalidToken {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}

func TestRefreshTokenRotationStore(t *testing.T) {
	s := NewMemoryRefreshStore()
	tok, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("NewRefreshToken: %v", err)
	}
	if hash != HashRefreshToken(tok) {
		t.Fatal("hash mismatch")
	}
	if err := s.Save(hash, "user-9", time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Save: %v", err)
	}
	uid, err := s.Consume(hash)
	if err != nil || uid != "user-9" {
		t.Fatalf("Consume = %q, %v", uid, err)
	}
	// Second use must fail (rotation/single-use).
	if _, err := s.Consume(hash); err != ErrInvalidToken {
		t.Fatalf("reused token accepted: %v", err)
	}
}

func TestRefreshTokenExpiry(t *testing.T) {
	s := NewMemoryRefreshStore()
	_, hash, _ := NewRefreshToken()
	_ = s.Save(hash, "u", time.Now().Add(-time.Second))
	if _, err := s.Consume(hash); err != ErrInvalidToken {
		t.Fatalf("expired token accepted: %v", err)
	}
}
