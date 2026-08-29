package httpapi

import "net/http"

// Server owns the HTTP mux and its dependencies. Construction is the
// composition root for the transport layer (see cmd/server).
type Server struct {
	mux *http.ServeMux
}

// NewServer wires routes. Dependencies are added incrementally per phase;
// unknown routes return 404 JSON.
func NewServer() *Server {
	s := &Server{mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
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
