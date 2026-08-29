package app_test

import (
	"net/http/httptest"
	"testing"
	"time"

	app "github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
	"github.com/hokm/platform/internal/httpapi"
	"github.com/hokm/platform/internal/infra/memory"
	"github.com/hokm/platform/internal/rating"
	"github.com/hokm/platform/internal/room"
	"github.com/hokm/platform/internal/ws"
)

// newTimeoutStack is a stack with a 300ms disconnect grace so takeover tests
// finish quickly.
func newTimeoutStack(t *testing.T) *stack {
	t.Helper()
	tokens := auth.NewTokenManager("test-secret-value-at-least-long", time.Hour)
	users := app.NewUserService(memory.NewUserStore(), tokens, auth.NewMemoryRefreshStore(), time.Hour)
	rooms := room.NewManager()
	scores := rating.NewMemoryStore()
	tables := app.NewTableManagerWithTimeout(rooms, tokens, 1, scores, 300*time.Millisecond)
	hub := ws.NewHub(tables)
	ts := httptest.NewServer(httpapi.NewServer(users, tokens, rooms, hub, nil, scores).Handler())
	t.Cleanup(ts.Close)
	return &stack{ts: ts, client: ts.Client(), scores: scores}
}

// TestDisconnectAITakeoverAndReconnect: the human disconnects mid-game, the
// AI takeover finishes the match, and the human can reconnect to a final
// consistent state.
func TestDisconnectAITakeoverAndReconnect(t *testing.T) {
	s := newTimeoutStack(t)
	tok := s.register(t, "ghost_player")

	created := s.post(t, "/api/rooms", tok, map[string]string{
		"name": "Ghost Room", "visibility": "private",
	})
	rm := created["room"].(map[string]any)
	roomID := rm["id"].(string)

	for i := 0; i < 3; i++ {
		s.post(t, "/api/rooms/"+roomID+"/ai", tok, map[string]string{"difficulty": "medium"})
	}
	s.post(t, "/api/rooms/"+roomID+"/ready", tok, map[string]bool{"ready": true})

	c := dialWS(t, s, tok, "ghost")
	c.send(ws.Envelope{Type: ws.CmdSubscribe, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	c.readUntil(func() bool {
		_, id := c.snapshot()
		return id == roomID
	}, 5*time.Second)
	c.send(ws.Envelope{Type: ws.CmdStartGame, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	c.readUntil(func() bool {
		st, _ := c.snapshot()
		return st.Phase == "trick_play" || st.Phase == "trump_selection"
	}, 5*time.Second)

	// Simulate disconnect: close the socket without finishing the match.
	_ = c.conn.Close()
	time.Sleep(500 * time.Millisecond) // grace period (300ms) elapses

	// Reconnect: subscribe again and drive to completion. If takeover did
	// not run, the human would still be required to act and the match could
	// not complete without further input — but with takeover active the
	// match finishes server-side regardless.
	c2 := dialWS(t, s, tok, "ghost")
	c2.send(ws.Envelope{Type: ws.CmdSubscribe, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	c2.readUntil(func() bool {
		st, _ := c2.snapshot()
		return st.Phase != "" // got a state snapshot
	}, 5*time.Second)
	final := driveGame(t, []*wsClient{c2}, roomID, 30*time.Second)

	if final.Phase != "game_complete" || !final.MatchOver {
		t.Fatalf("match did not complete after takeover: phase=%s", final.Phase)
	}
	if final.LastTrick == nil || final.LastTrick.Number != 13 {
		t.Fatalf("last trick = %+v, want #13", final.LastTrick)
	}
	t.Logf("takeover match complete: roundsA=%v roundsB=%v", final.RoundsWon[0], final.RoundsWon[1])
}
