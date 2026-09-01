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

// GameConfig centralizes every gameplay timing/limit (impliment.md Â§1).
// Gameplay logic reads these â€” never numeric literals. Defaults live here
// only; all are overridable via environment.
type GameConfig struct {
	// HakemSelectionTimeout: trump must be chosen within this window
	// (default 10s); expiry â†’ deterministic automatic trump.
	HakemSelectionTimeout time.Duration
	// CardSelectionTimeouts per game speed (fast 5s, medium 10s, slow 15s).
	CardSelectionTimeouts GameSpeedTimeouts
	// ReconnectGracePeriod before AI takeover (default 30s).
	ReconnectGracePeriod time.Duration
	// Presentation-only values (never coupled to gameplay timeouts):
	TrickWinnerDisplayDuration time.Duration // default 3s
	CardPlayAnimationDuration  time.Duration // default 0.5s
	// AIMoveDelay paces automatic plays so the table stays readable
	// (default 1s between AI/automatic card plays).
	AIMoveDelay time.Duration
	// TrickPause keeps a completed trick on the table before the next one
	// begins (default 1s) - winner reveal + collection window.
	TrickPause time.Duration
	// AllowedRoundCounts for room creation (default 1, 3, 5).
	AllowedRoundCounts []int
}

// GameSpeedTimeouts maps game speed names to card-selection timeouts.
type GameSpeedTimeouts struct {
	Fast   time.Duration
	Medium time.Duration
	Slow   time.Duration
}

// CardTimeout returns the card-selection timeout for a speed, defaulting to
// medium for unknown speeds.
func (g GameConfig) CardTimeout(speed string) time.Duration {
	switch speed {
	case "fast":
		return g.CardSelectionTimeouts.Fast
	case "slow":
		return g.CardSelectionTimeouts.Slow
	default:
		return g.CardSelectionTimeouts.Medium
	}
}

// DefaultGameConfig returns the built-in defaults; callers may override
// fields (tests) before use.
func DefaultGameConfig() GameConfig {
	return GameConfig{
		HakemSelectionTimeout:      10 * time.Second,
		CardSelectionTimeouts:      GameSpeedTimeouts{Fast: 5 * time.Second, Medium: 10 * time.Second, Slow: 15 * time.Second},
		ReconnectGracePeriod:       30 * time.Second,
		TrickWinnerDisplayDuration: 3 * time.Second,
		CardPlayAnimationDuration:  500 * time.Millisecond,
		AIMoveDelay:                1 * time.Second,
		TrickPause:                 1 * time.Second,
		AllowedRoundCounts:         []int{1, 3, 5},
	}
}

func defaultGameConfig() GameConfig { return DefaultGameConfig() }

func loadGameConfig() GameConfig {
	g := defaultGameConfig()
	g.HakemSelectionTimeout = getDur("GAME_HAKEM_SELECTION_TIMEOUT", g.HakemSelectionTimeout)
	g.CardSelectionTimeouts.Fast = getDur("GAME_CARD_SELECTION_TIMEOUT_FAST", g.CardSelectionTimeouts.Fast)
	g.CardSelectionTimeouts.Medium = getDur("GAME_CARD_SELECTION_TIMEOUT_MEDIUM", g.CardSelectionTimeouts.Medium)
	g.CardSelectionTimeouts.Slow = getDur("GAME_CARD_SELECTION_TIMEOUT_SLOW", g.CardSelectionTimeouts.Slow)
	g.ReconnectGracePeriod = getDur("GAME_RECONNECT_GRACE_PERIOD", g.ReconnectGracePeriod)
	g.TrickWinnerDisplayDuration = getDur("GAME_TRICK_WINNER_DISPLAY_DURATION", g.TrickWinnerDisplayDuration)
	g.CardPlayAnimationDuration = getDur("GAME_CARD_PLAY_ANIMATION_DURATION", g.CardPlayAnimationDuration)
	g.AIMoveDelay = getDur("GAME_AI_MOVE_DELAY", g.AIMoveDelay)
	g.TrickPause = getDur("GAME_TRICK_PAUSE", g.TrickPause)
	if counts := getIntList("GAME_ALLOWED_ROUND_COUNTS", nil); counts != nil {
		g.AllowedRoundCounts = counts
	}
	return g
}

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

	Game GameConfig
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
		AccessTTL:  getDur("JWT_ACCESS_TTL", 720*time.Hour),
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
		Game: loadGameConfig(),
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

func getIntList(key string, def []int) []int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(p), "%d", &n); err == nil && n > 0 {
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}
