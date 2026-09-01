package app

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ChatMessage is one chat line in a room.
type ChatMessage struct {
	ID       int64     `json:"id"`
	RoomID   string    `json:"room_id"`
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Body     string    `json:"body"`
	IsSystem bool      `json:"is_system"`
	At       time.Time `json:"at"`
}

// ChatSink receives broadcast-ready chat messages (the WS layer implements
// it to fan out to room subscribers).
type ChatSink interface {
	ChatMessage(msg ChatMessage)
}

// ChatService stores room chat (bounded history per room) and enforces
// basic moderation (length, rate). Only player Send text is stored.
type ChatService struct {
	mu         sync.Mutex
	history    map[string][]ChatMessage
	nextID     int64
	sink       ChatSink
	sendRate   map[string][]time.Time // userID → recent send times
	maxPer10s  int
	maxHistory int
}

func NewChatService() *ChatService {
	return &ChatService{
		history:    make(map[string][]ChatMessage),
		sendRate:   make(map[string][]time.Time),
		maxPer10s:  5,
		maxHistory: 50,
	}
}

// SetSink registers the broadcast fan-out.
func (c *ChatService) SetSink(s ChatSink) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sink = s
}

// Send validates, stores, and broadcasts a player message. Rate limited.
func (c *ChatService) Send(roomID, userID, username, body string) (ChatMessage, error) {
	body = strings.TrimSpace(body)
	if len(body) == 0 || len(body) > 500 {
		return ChatMessage{}, fmt.Errorf("%w: message must be 1-500 characters", ErrValidation)
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// Rate limit per user.
	now := time.Now()
	recent := c.sendRate[userID][:0]
	for _, t := range c.sendRate[userID] {
		if now.Sub(t) < 10*time.Second {
			recent = append(recent, t)
		}
	}
	c.sendRate[userID] = recent
	if len(recent) >= c.maxPer10s {
		return ChatMessage{}, fmt.Errorf("%w: sending too fast", ErrValidation)
	}
	c.sendRate[userID] = append(c.sendRate[userID], now)

	c.nextID++
	msg := ChatMessage{
		ID:       c.nextID,
		RoomID:   roomID,
		UserID:   userID,
		Username: username,
		Body:     body,
		At:       now.UTC(),
	}
	c.appendLocked(msg)
	return msg, nil
}

// System is a no-op: only player messages appear in chat (ADR-0012).
func (c *ChatService) System(roomID, body string) ChatMessage {
	return ChatMessage{RoomID: roomID, Body: body, IsSystem: true}
}

// History returns the last n messages for a room (oldest first).
func (c *ChatService) History(roomID string, n int) []ChatMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := c.history[roomID]
	if n > len(h) {
		n = len(h)
	}
	out := make([]ChatMessage, n)
	copy(out, h[len(h)-n:])
	return out
}

func (c *ChatService) appendLocked(msg ChatMessage) {
	c.history[msg.RoomID] = append(c.history[msg.RoomID], msg)
	if len(c.history[msg.RoomID]) > c.maxHistory {
		c.history[msg.RoomID] = c.history[msg.RoomID][1:]
	}
	if c.sink != nil {
		c.sink.ChatMessage(msg)
	}
}
