package httpapi

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
	"github.com/hokm/platform/internal/metrics"
	"github.com/hokm/platform/internal/rating"
	"github.com/hokm/platform/internal/room"
	"github.com/hokm/platform/internal/ws"
)

// Limiter is the rate-limit hook (Redis-backed in production; nil disables).
type Limiter interface {
	Allow(key string) bool
}

// Server owns the HTTP mux and its dependencies. Construction is the
// composition root for the transport layer (see cmd/server).
type Server struct {
	mux     *http.ServeMux
	users   *app.UserService
	tokens  *auth.TokenManager
	rooms   *room.Manager
	hub     *ws.Hub
	limiter Limiter
	scores  rating.ScoreStore
}

// NewServer wires routes. Dependencies are added incrementally per phase;
// unknown routes return 404 JSON.
func NewServer(users *app.UserService, tokens *auth.TokenManager, rooms *room.Manager, hub *ws.Hub, limiter Limiter, scores rating.ScoreStore) *Server {
	s := &Server{mux: http.NewServeMux(), users: users, tokens: tokens, rooms: rooms, hub: hub, limiter: limiter, scores: scores}
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/metrics", s.handleMetrics)

	// Auth (Phase 4) — rate limited (Phase 11/15).
	s.mux.Handle("POST /api/auth/register", s.limit(http.HandlerFunc(s.handleRegister)))
	s.mux.Handle("POST /api/auth/login", s.limit(http.HandlerFunc(s.handleLogin)))
	s.mux.Handle("POST /api/auth/refresh", s.limit(http.HandlerFunc(s.handleRefresh)))
	s.mux.Handle("GET /api/me", s.RequireAuth(http.HandlerFunc(s.handleMe)))
	s.mux.Handle("PATCH /api/me", s.RequireAuth(http.HandlerFunc(s.handleUpdateMe)))

	// Rooms (Phase 5).
	s.mux.Handle("POST /api/rooms", s.RequireAuth(http.HandlerFunc(s.handleCreateRoom)))
	s.mux.Handle("GET /api/rooms", s.RequireAuth(http.HandlerFunc(s.handleListRooms)))
	s.mux.Handle("GET /api/rooms/{id}", s.RequireAuth(http.HandlerFunc(s.handleGetRoom)))
	s.mux.Handle("POST /api/rooms/join", s.RequireAuth(http.HandlerFunc(s.handleJoinRoom)))
	s.mux.Handle("POST /api/rooms/{id}/leave", s.RequireAuth(http.HandlerFunc(s.handleLeaveRoom)))
	s.mux.Handle("POST /api/rooms/{id}/ready", s.RequireAuth(http.HandlerFunc(s.handleReady)))
	s.mux.Handle("POST /api/rooms/{id}/kick", s.RequireAuth(http.HandlerFunc(s.handleKick)))
	s.mux.Handle("POST /api/rooms/{id}/ai", s.RequireAuth(http.HandlerFunc(s.handleAddAI)))
	s.mux.Handle("POST /api/rooms/{id}/ai/fill", s.RequireAuth(http.HandlerFunc(s.handleFillAI)))
	s.mux.Handle("POST /api/rooms/{id}/ai/remove", s.RequireAuth(http.HandlerFunc(s.handleRemoveAI)))
	s.mux.Handle("POST /api/rooms/{id}/seats", s.RequireAuth(http.HandlerFunc(s.handleMoveSeats)))
	s.mux.Handle("DELETE /api/rooms/{id}", s.RequireAuth(http.HandlerFunc(s.handleDeleteRoom)))

	// Stats & ranking (Phase 13).
	s.mux.Handle("GET /api/leaderboard", s.RequireAuth(http.HandlerFunc(s.handleLeaderboard)))
	s.mux.Handle("GET /api/stats", s.RequireAuth(http.HandlerFunc(s.handleMyStats)))

	// WebSocket (Phases 6-7) — rate limited on the upgrade handshake.
	if hub != nil {
		s.mux.Handle("GET /api/ws", s.limit(http.HandlerFunc(hub.ServeWS)))
	}

	// Static frontend (Phase 18): when WEB_DIR is set, serve the built SPA
	// with an index.html fallback for client-side routes.
	if dir := os.Getenv("WEB_DIR"); dir != "" {
		s.mountStatic(dir)
	}

	return s
}

// mountStatic serves static assets with SPA fallback. API routes keep
// precedence because the mux matches the most specific pattern first.
func (s *Server) mountStatic(dir string) {
	fileServer := http.FileServer(http.Dir(dir))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeError(w, r, apiError(http.StatusMethodNotAllowed, "method", "method not allowed"))
			return
		}
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}

// handleLeaderboard returns the top players by wins (ADR-0012).
func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if s.scores == nil {
		writeJSON(w, http.StatusOK, map[string]any{"entries": []any{}})
		return
	}
	entries, err := s.scores.Leaderboard(50)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

// handleMyStats returns the caller's statistics entry.
func (s *Server) handleMyStats(w http.ResponseWriter, r *http.Request) {
	uid, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apiError(http.StatusUnauthorized, "unauthorized", "not authenticated"))
		return
	}
	if s.scores == nil {
		writeJSON(w, http.StatusOK, map[string]any{"stats": nil})
		return
	}
	entry, err := s.scores.StatsOf(uid)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": entry})
}

// limit applies the per-IP limiter when configured; fails open otherwise.
func (s *Server) limit(next http.Handler) http.Handler {
	if s.limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.RemoteAddr
		if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			host = h
		}
		if !s.limiter.Allow(host) {
			writeJSON(w, http.StatusTooManyRequests, &APIError{
				Code: "rate_limited", Message: "too many requests, slow down",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// usernameFor resolves the display name for a user id via the user service.
func (s *Server) usernameFor(userID string) string {
	name, _, _ := s.displayFor(userID)
	return name
}

// displayFor returns username and avatar fields for seating (ADR-0017/0018).
func (s *Server) displayFor(userID string) (username, avatarSeed, avatarStyle string) {
	if u, err := s.users.Profile(userID); err == nil {
		return u.Username, u.AvatarSeed, u.AvatarStyle
	}
	return "player", "", ""
}

// RoomCodeFromPath is a small helper for invite-link resolution.
func RoomCodeFromPath(code string) string { return strings.ToUpper(code) }

// Handler returns the fully wrapped handler chain.
func (s *Server) Handler() http.Handler {
	return secureHeaders(loggingMiddleware(recoverMiddleware(s.mux)))
}

// handleHealth is the liveness probe used by Docker healthchecks.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleMetrics exposes Prometheus text metrics (Phase 17).
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(metrics.Render()))
}

// DefaultTokenManager builds the server's token manager from config values.
func DefaultTokenManager(secret string, accessTTL time.Duration) *auth.TokenManager {
	return auth.NewTokenManager(secret, accessTTL)
}
