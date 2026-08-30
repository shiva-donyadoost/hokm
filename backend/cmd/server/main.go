// Command server is the composition root: it loads config, builds
// dependencies, and starts the HTTP server with graceful shutdown.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
	"github.com/hokm/platform/internal/config"
	"github.com/hokm/platform/internal/httpapi"
	"github.com/hokm/platform/internal/infra/memory"
	"github.com/hokm/platform/internal/infra/postgres"
	"github.com/hokm/platform/internal/infra/redisx"
	"github.com/hokm/platform/internal/rating"
	"github.com/hokm/platform/internal/room"
	"github.com/hokm/platform/internal/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	setupLogging(cfg.LogLevel)
	ctx := context.Background()

	// Composition root. Durable stores replace in-memory ones when the
	// databases are reachable (ADR-0006); otherwise the server degrades
	// to memory-only with a warning.
	var userRepo app.UserRepo = memory.NewUserStore()
	var refreshStore auth.RefreshStore = auth.NewMemoryRefreshStore()
	var pool *pgxpool.Pool
	if p, err := postgres.Connect(ctx, cfg.Postgres.DSN()); err != nil {
		slog.Warn("postgres unavailable, using in-memory stores", "err", err)
	} else {
		pool = p
		defer pool.Close()
		slog.Info("postgres connected, migrations applied")
		userRepo = postgres.NewUserStore(pool)
		refreshStore = postgres.NewRefreshStore(pool)
	}

	var limiter httpapi.Limiter
	if rdb, err := redisx.NewClient(ctx, cfg.Redis.Addr); err != nil {
		slog.Warn("redis unavailable, rate limiting disabled", "err", err)
	} else {
		defer func() { _ = rdb.Close() }()
		limiter = redisx.NewRateLimiter(rdb, time.Minute, 60)
	}

	var scores rating.ScoreStore = rating.NewMemoryStore()
	if pool != nil {
		scores = postgres.NewScoreStore(pool)
	}

	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTTL)
	users := app.NewUserService(userRepo, tokens, refreshStore, cfg.RefreshTTL)
	rooms := room.NewManager()
	tables := app.NewTableManager(rooms, tokens, scores, cfg.Game)
	hub := ws.NewHub(tables)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewServer(users, tokens, rooms, hub, limiter, scores).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", cfg.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}

func setupLogging(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})
	slog.SetDefault(slog.New(h))
}
