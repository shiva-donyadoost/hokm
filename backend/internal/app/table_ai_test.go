package app_test

import (
	"testing"
	"time"

	"github.com/hokm/platform/internal/ws"
)

// TestHumanWithThreeAIsCompletesGame registers one human, fills the other
// three seats with AI, and completes a full match over WebSockets.
func TestHumanWithThreeAIsCompletesGame(t *testing.T) {
	s := newStack(t)
	tok := s.register(t, "solo_human")

	created := s.post(t, "/api/rooms", tok, map[string]string{
		"name": "Solo vs AI", "visibility": "private",
	})
	rm := created["room"].(map[string]any)
	roomID := rm["id"].(string)

	// Fill three AI seats (host only can; AI members auto-ready).
	for i := 0; i < 3; i++ {
		s.post(t, "/api/rooms/"+roomID+"/ai", tok, map[string]string{
			"difficulty": []string{"medium", "hard", "pro"}[i],
		})
	}
	// The human host readies up too.
	s.post(t, "/api/rooms/"+roomID+"/ready", tok, map[string]bool{"ready": true})

	c := dialWS(t, s, tok, "human")
	c.send(ws.Envelope{Type: ws.CmdSubscribe, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	c.readUntil(func() bool {
		_, id := c.snapshot()
		return id == roomID
	}, 5*time.Second)

	// Host starts; the shared driver plays the human's legal moves while
	// AI seats answer automatically server-side.
	c.send(ws.Envelope{Type: ws.CmdStartGame, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	st := driveGame(t, []*wsClient{c}, roomID, 60*time.Second)

	if st.RoundsWon[0]+st.RoundsWon[1] != 1 {
		t.Fatalf("rounds won = %v, want exactly one round decided", st.RoundsWon)
	}
	if st.LastTrick == nil || st.LastTrick.Number != 13 {
		t.Fatalf("last trick = %+v, want #13", st.LastTrick)
	}
	t.Logf("human vs AI complete: roundsA=%v roundsB=%v", st.RoundsWon[0], st.RoundsWon[1])
}
