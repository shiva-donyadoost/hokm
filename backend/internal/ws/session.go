package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 45 * time.Second
	pingPeriod = 20 * time.Second
	sendBuffer = 64
)

// Session is one authenticated WebSocket connection. Outgoing writes are
// serialized through a buffered channel to avoid concurrent write races.
type Session struct {
	UserID   string
	Username string

	conn *websocket.Conn
	send chan []byte

	mu     sync.Mutex
	closed bool
}

// NewSession wraps an upgraded connection.
func NewSession(conn *websocket.Conn, userID, username string) *Session {
	return &Session{
		UserID:   userID,
		Username: username,
		conn:     conn,
		send:     make(chan []byte, sendBuffer),
	}
}

// Send queues a JSON envelope for delivery. Returns false if the session
// is closing or its buffer is full (the client is too slow and will be
// dropped by the write pump failing).
func (s *Session) Send(env Envelope) bool {
	b, err := json.Marshal(env)
	if err != nil {
		slog.Error("ws: marshal", "err", err)
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	select {
	case s.send <- b:
		return true
	default:
		return false
	}
}

// Close stops the pumps and marks the session unusable.
func (s *Session) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.send)
	s.mu.Unlock()
	_ = s.conn.Close()
}

// ReadPump reads client envelopes and hands them to onMessage. It also
// enforces the pong deadline.
func (s *Session) ReadPump(onMessage func(*Session, Envelope)) {
	defer func() {
		_ = s.conn.Close()
	}()
	s.conn.SetReadLimit(64 * 1024) // 64 KiB message cap
	_ = s.conn.SetReadDeadline(time.Now().Add(pongWait))
	s.conn.SetPongHandler(func(string) error {
		return s.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		var env Envelope
		if err := s.conn.ReadJSON(&env); err != nil {
			return // normal close or error — hub cleans up
		}
		onMessage(s, env)
		_ = s.conn.SetReadDeadline(time.Now().Add(pongWait))
	}
}

// WritePump drains the send channel and pings on schedule.
func (s *Session) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = s.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-s.send:
			_ = s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = s.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := s.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = s.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
