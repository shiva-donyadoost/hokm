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

	"github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
	"github.com/hokm/platform/internal/config"
	"github.com/hokm/platform/internal/httpapi"
	"github.com/hokm/platform/internal/infra/memory"
	"github.com/hokm/platform/internal/room"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	setupLogging(cfg.LogLevel)

	// Composition root. Postgres-backed repositories replace the in-memory
	// stores in Phase 11 behind the same interfaces (ADR-0006).
	tokens := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTTL)
	users := app.NewUserService(memory.NewUserStore(), tokens,
		auth.NewMemoryRefreshStore(), cfg.RefreshTTL)
	rooms := room.NewManager()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewServer(users, tokens, rooms).Handler(),
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
