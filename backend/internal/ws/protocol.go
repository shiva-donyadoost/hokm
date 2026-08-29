// Package ws implements the WebSocket transport per ADR-0003: one
// authenticated endpoint, a JSON envelope protocol, per-room fan-out, and
// heartbeat-based liveness.
package ws

import "encoding/json"

// Envelope is the single wire format in both directions.
type Envelope struct {
	Type    string          `json:"type"`              // command or server message type
	ID      string          `json:"id,omitempty"`      // client message id (echoed on ERROR)
	Name    string          `json:"name,omitempty"`    // event name for EVENTS messages
	Payload json.RawMessage `json:"payload,omitempty"` // decoded per Type/Name
}

// Client command types.
const (
	CmdPing        = "PING"
	CmdSubscribe   = "SUBSCRIBE"    // payload: {room_id}
	CmdStartGame   = "START_GAME"   // payload: {room_id}
	CmdSelectTrump = "SELECT_TRUMP" // payload: {room_id, suit}
	CmdPlayCard    = "PLAY_CARD"    // payload: {room_id, card:{suit,rank}}
	CmdChat        = "CHAT"         // payload: {room_id, body}
)

// Server message types.
const (
	MsgState  = "STATE"  // payload: full per-seat game view
	MsgEvents = "EVENTS" // payload: list of public events since last message
	MsgRoom   = "ROOM"   // payload: room snapshot (lobby updates)
	MsgChat   = "CHAT"   // payload: {room_id, user_id, username, body, is_system, at}
	MsgError  = "ERROR"  // payload: {code, message}
	MsgPong   = "PONG"
)

// Simple payloads.

type SubscribePayload struct {
	RoomID string `json:"room_id"`
}

type SelectTrumpPayload struct {
	RoomID string `json:"room_id"`
	Suit   string `json:"suit"`
}

type PlayCardPayload struct {
	RoomID string   `json:"room_id"`
	Card   WireCard `json:"card"`
}

type WireCard struct {
	Suit string `json:"suit"`
	Rank int    `json:"rank"`
}

type StartGamePayload struct {
	RoomID string `json:"room_id"`
}

type ChatPayload struct {
	RoomID string `json:"room_id"`
	Body   string `json:"body"`
}

// ErrorPayload is the structured error body.
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
