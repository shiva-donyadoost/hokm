package app_test

import (
	"encoding/json"
	"testing"
	"time"

	app "github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/ws"
)

// TestTwoClientsChat covers the chat round-trip over WebSockets, including
// the system message emitted when a member joins.
func TestTwoClientsChat(t *testing.T) {
	s := newStack(t)
	tokA := s.register(t, "chatter1")
	tokB := s.register(t, "chatter2")

	created := s.post(t, "/api/rooms", tokA, map[string]string{
		"name": "Chat Room", "visibility": "public",
	})
	rm := created["room"].(map[string]any)
	roomID := rm["id"].(string)
	code := rm["code"].(string)

	ca := dialWS(t, s, tokA, "chatter1")
	ca.send(ws.Envelope{Type: ws.CmdSubscribe, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	ca.readUntil(func() bool {
		_, id := ca.snapshot()
		return id == roomID
	}, 5*time.Second)

	// B joins via REST then subscribes → A receives a join system message.
	s.post(t, "/api/rooms/join", tokB, map[string]string{"code": code})
	cb := dialWS(t, s, tokB, "chatter2")
	cb.send(ws.Envelope{Type: ws.CmdSubscribe, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})

	// B sends a chat line; both clients must see it.
	cb.send(ws.Envelope{Type: ws.CmdChat, Payload: mustJSONRaw(map[string]string{
		"room_id": roomID, "body": "salaam everyone",
	})})

	waitForChat := func(c *wsClient, body string) {
		t.Helper()
		deadline := time.After(5 * time.Second)
		for {
			select {
			case env := <-c.msgs:
				if env.Type == ws.MsgChat {
					var m app.ChatMessage
					if err := json.Unmarshal(env.Payload, &m); err == nil && m.Body == body {
						return
					}
				}
			case <-deadline:
				c.t.Fatalf("%s: chat %q not received", c.name, body)
			}
		}
	}
	waitForChat(ca, "chatter2 joined the room")
	waitForChat(ca, "salaam everyone")
	waitForChat(cb, "salaam everyone")

	// Membership is enforced: a non-member cannot chat.
	tokC := s.register(t, "outsider1")
	cc := dialWS(t, s, tokC, "outsider1")
	cc.send(ws.Envelope{Type: ws.CmdSubscribe, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	cc.send(ws.Envelope{Type: ws.CmdChat, Payload: mustJSONRaw(map[string]string{
		"room_id": roomID, "body": "let me in",
	})})
	deadline := time.After(3 * time.Second)
	for {
		select {
		case env := <-cc.msgs:
			if env.Type == ws.MsgError {
				return // rejected as expected
			}
		case <-deadline:
			cc.t.Fatal("outsider chat was not rejected")
		}
	}
}
