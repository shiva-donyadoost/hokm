package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

// CommandHandler processes authenticated client commands. Implemented by
// the application layer (table manager) so this package stays transport-only.
type CommandHandler interface {
	// HandleCommand routes a client command. Errors are translated to
	// ERROR envelopes by the hub.
	HandleCommand(s *Session, env Envelope) error
	// OnDisconnect notifies the application that a session ended.
	OnDisconnect(s *Session)
	// Authenticate validates the upgrade token and returns the user.
	Authenticate(token string) (userID, username string, err error)
}

// Hub owns the upgrade endpoint and session lifecycle.
type Hub struct {
	handler CommandHandler
}

func NewHub(h CommandHandler) *Hub { return &Hub{handler: h} }

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true // same-origin enforced by cookie-free token auth; tighten per deployment
	},
}

// ServeWS is the /api/ws HTTP handler: authenticates, upgrades, and starts
// the session pumps.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	userID, username, err := h.handler.Authenticate(token)
	if err != nil || userID == "" {
		writePlainError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing token")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("ws: upgrade failed", "err", err)
		return
	}
	s := NewSession(conn, userID, username)
	go func() {
		defer h.handler.OnDisconnect(s)
		s.ReadPump(h.onMessage)
	}()
	go s.WritePump()
}

func (h *Hub) onMessage(s *Session, env Envelope) {
	switch env.Type {
	case CmdPing:
		s.Send(Envelope{Type: MsgPong, ID: env.ID})
		return
	default:
		if err := h.handler.HandleCommand(s, env); err != nil {
			s.Send(Envelope{
				Type:    MsgError,
				ID:      env.ID,
				Payload: mustJSON(ErrorPayload{Code: "command_failed", Message: err.Error()}),
			})
		}
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

func writePlainError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorPayload{Code: code, Message: msg})
}
