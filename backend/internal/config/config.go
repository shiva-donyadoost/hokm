// Package config loads service configuration from the environment
// (12-factor style). Every knob has a documented default; secrets are never
// defaulted.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Config is the process-wide configuration.
type Config struct {
	Env       string // development | production
	Addr      string // HTTP listen address
	LogLevel  string // debug | info | warn | error
	JWTSecret string

	AccessTTL  time.Duration
	RefreshTTL time.Duration

	Postgres PostgresConfig
	Redis    RedisConfig

	RoundsToWin int
	TurnTimeout time.Duration
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DB       string
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.Port, p.DB)
}

type RedisConfig struct {
	Addr string
}

// Load reads configuration from the environment.
func Load() (*Config, error) {
	c := &Config{
		Env:        get("APP_ENV", "development"),
		Addr:       get("APP_ADDR", ":8080"),
		LogLevel:   get("LOG_LEVEL", "info"),
		JWTSecret:  get("JWT_SECRET", ""),
		AccessTTL:  getDur("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL: getDur("JWT_REFRESH_TTL", 720*time.Hour),
		Postgres: PostgresConfig{
			Host:     get("POSTGRES_HOST", "localhost"),
			Port:     get("POSTGRES_PORT", "5432"),
			User:     get("POSTGRES_USER", "hokm"),
			Password: get("POSTGRES_PASSWORD", ""),
			DB:       get("POSTGRES_DB", "hokm"),
		},
		Redis: RedisConfig{
			Addr: get("REDIS_ADDR", "localhost:6379"),
		},
		RoundsToWin: getInt("ROUNDS_TO_WIN", 7),
		TurnTimeout: time.Duration(getInt("TURN_TIMEOUT_SECONDS", 60)) * time.Second,
	}
	if c.Env != "development" && c.Env != "production" {
		return nil, fmt.Errorf("config: invalid APP_ENV %q", c.Env)
	}
	if c.Env == "production" && len(c.JWTSecret) < 32 {
		return nil, fmt.Errorf("config: JWT_SECRET must be >= 32 chars in production")
	}
	if c.JWTSecret == "" {
		// Development convenience secret; never valid in production.
		c.JWTSecret = "dev-only-insecure-secret-change-me-0123456789"
	}
	return c, nil
}

func get(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getDur(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func getInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}
