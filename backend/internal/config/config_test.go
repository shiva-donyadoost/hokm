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
	if c.Game.AllowedRoundCounts == nil || len(c.Game.AllowedRoundCounts) == 0 {
		t.Errorf("allowedRoundCounts = %v", c.Game.AllowedRoundCounts)
	}
	if c.Game.HakemSelectionTimeout != 10*time.Second {
		t.Errorf("hakemSelectionTimeout = %v, want 10s", c.Game.HakemSelectionTimeout)
	}
	if c.Game.CardTimeout("fast") != 5*time.Second {
		t.Errorf("fast timeout = %v, want 5s", c.Game.CardTimeout("fast"))
	}
	if c.Game.CardTimeout("slow") != 15*time.Second {
		t.Errorf("slow timeout = %v, want 15s", c.Game.CardTimeout("slow"))
	}
	if c.Game.ReconnectGracePeriod != 30*time.Second {
		t.Errorf("grace = %v, want 30s", c.Game.ReconnectGracePeriod)
	}
	if c.AccessTTL != 720*time.Hour {
		t.Errorf("accessTTL = %v, want 720h", c.AccessTTL)
	}
	if c.RefreshTTL != 720*time.Hour {
		t.Errorf("refreshTTL = %v, want 720h", c.RefreshTTL)
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
