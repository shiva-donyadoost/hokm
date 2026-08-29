package httpapi

import (
	"net/http"
	"time"

	"github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
)

// Server owns the HTTP mux and its dependencies. Construction is the
// composition root for the transport layer (see cmd/server).
type Server struct {
	mux    *http.ServeMux
	users  *app.UserService
	tokens *auth.TokenManager
}

// NewServer wires routes. Dependencies are added incrementally per phase;
// unknown routes return 404 JSON.
func NewServer(users *app.UserService, tokens *auth.TokenManager) *Server {
	s := &Server{mux: http.NewServeMux(), users: users, tokens: tokens}
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	// Auth (Phase 4).
	s.mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/refresh", s.handleRefresh)
	s.mux.Handle("GET /api/me", s.RequireAuth(http.HandlerFunc(s.handleMe)))

	return s
}

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
