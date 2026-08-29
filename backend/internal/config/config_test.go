package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Env != "development" {
		t.Errorf("env = %q, want development", c.Env)
	}
	if c.Addr != ":8080" {
		t.Errorf("addr = %q", c.Addr)
	}
	if c.RoundsToWin != 7 {
		t.Errorf("roundsToWin = %d, want 7", c.RoundsToWin)
	}
	if c.AccessTTL != 15*time.Minute {
		t.Errorf("accessTTL = %v", c.AccessTTL)
	}
}

func TestLoadProductionRequiresSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_SECRET", "")
	if _, err := Load(); err == nil {
		t.Fatal("production without JWT_SECRET must fail")
	}
	t.Setenv("JWT_SECRET", "a-long-enough-production-secret-value-1234567890")
	if _, err := Load(); err != nil {
		t.Fatalf("Load with secret: %v", err)
	}
}

func TestPostgresDSN(t *testing.T) {
	p := PostgresConfig{Host: "db", Port: "5432", User: "u", Password: "p", DB: "hokm"}
	want := "postgres://u:p@db:5432/hokm?sslmode=disable"
	if p.DSN() != want {
		t.Errorf("DSN = %q, want %q", p.DSN(), want)
	}
}
