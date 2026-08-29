package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
	"github.com/hokm/platform/internal/room"
)

// Server owns the HTTP mux and its dependencies. Construction is the
// composition root for the transport layer (see cmd/server).
type Server struct {
	mux    *http.ServeMux
	users  *app.UserService
	tokens *auth.TokenManager
	rooms  *room.Manager
}

// NewServer wires routes. Dependencies are added incrementally per phase;
// unknown routes return 404 JSON.
func NewServer(users *app.UserService, tokens *auth.TokenManager, rooms *room.Manager) *Server {
	s := &Server{mux: http.NewServeMux(), users: users, tokens: tokens, rooms: rooms}
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	// Auth (Phase 4).
	s.mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/refresh", s.handleRefresh)
	s.mux.Handle("GET /api/me", s.RequireAuth(http.HandlerFunc(s.handleMe)))

	// Rooms (Phase 5).
	s.mux.Handle("POST /api/rooms", s.RequireAuth(http.HandlerFunc(s.handleCreateRoom)))
	s.mux.Handle("GET /api/rooms", s.RequireAuth(http.HandlerFunc(s.handleListRooms)))
	s.mux.Handle("GET /api/rooms/{id}", s.RequireAuth(http.HandlerFunc(s.handleGetRoom)))
	s.mux.Handle("POST /api/rooms/join", s.RequireAuth(http.HandlerFunc(s.handleJoinRoom)))
	s.mux.Handle("POST /api/rooms/{id}/leave", s.RequireAuth(http.HandlerFunc(s.handleLeaveRoom)))
	s.mux.Handle("POST /api/rooms/{id}/ready", s.RequireAuth(http.HandlerFunc(s.handleReady)))
	s.mux.Handle("POST /api/rooms/{id}/kick", s.RequireAuth(http.HandlerFunc(s.handleKick)))
	s.mux.Handle("POST /api/rooms/{id}/ai", s.RequireAuth(http.HandlerFunc(s.handleAddAI)))
	s.mux.Handle("POST /api/rooms/{id}/ai/remove", s.RequireAuth(http.HandlerFunc(s.handleRemoveAI)))

	return s
}

// usernameFor resolves the display name for a user id via the user service.
func (s *Server) usernameFor(userID string) string {
	if u, err := s.users.Profile(userID); err == nil {
		return u.Username
	}
	return "player"
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

// DefaultTokenManager builds the server's token manager from config values.
func DefaultTokenManager(secret string, accessTTL time.Duration) *auth.TokenManager {
	return auth.NewTokenManager(secret, accessTTL)
}
