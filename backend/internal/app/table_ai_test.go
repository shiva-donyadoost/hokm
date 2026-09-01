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
	if st.LastTrick == nil || st.LastTrick.Number < 7 || st.LastTrick.Number > 13 {
		t.Fatalf("last trick = %+v, want number 7-13", st.LastTrick)
	}

	// The human's stats must have been recorded (AI seats are never scored).
	lb, err := s.scores.Leaderboard(10)
	if err != nil {
		t.Fatalf("leaderboard: %v", err)
	}
	if len(lb) != 1 {
		t.Fatalf("leaderboard entries = %d, want 1 (human only)", len(lb))
	}
	e := lb[0]
	if e.GamesPlayed != 1 || (e.Wins+e.Losses) != 1 || e.Rating == 1000 {
		t.Fatalf("stats not updated: %+v", e)
	}
	t.Logf("stats recorded: games=%d wins=%d losses=%d rating=%d",
		e.GamesPlayed, e.Wins, e.Losses, e.Rating)
}

func TestReplayRestartsSameSeats(t *testing.T) {
	s := newStack(t)
	tok := s.register(t, "replay_host")

	created := s.post(t, "/api/rooms", tok, map[string]string{
		"name": "Replay Room", "visibility": "private",
	})
	rm := created["room"].(map[string]any)
	roomID := rm["id"].(string)
	s.post(t, "/api/rooms/"+roomID+"/ai/fill", tok, map[string]any{})

	c := dialWS(t, s, tok, "host")
	c.send(ws.Envelope{Type: ws.CmdSubscribe, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	c.readUntil(func() bool {
		_, id := c.snapshot()
		return id == roomID
	}, 5*time.Second)

	c.send(ws.Envelope{Type: ws.CmdStartGame, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	st := driveGame(t, []*wsClient{c}, roomID, 60*time.Second)
	if !st.MatchOver {
		t.Fatal("first match did not complete")
	}

	c.send(ws.Envelope{Type: ws.CmdReplayGame, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	c.readUntil(func() bool {
		ns, _ := c.snapshot()
		return !ns.MatchOver && ns.RoundNumber == 1 && len(ns.YourHand) >= 5
	}, 10*time.Second)

	st2 := driveGame(t, []*wsClient{c}, roomID, 60*time.Second)
	if !st2.MatchOver {
		t.Fatal("replay match did not complete")
	}
	lb, err := s.scores.Leaderboard(10)
	if err != nil {
		t.Fatalf("leaderboard: %v", err)
	}
	if len(lb) != 1 || lb[0].GamesPlayed != 2 {
		t.Fatalf("want 2 recorded games, got %+v", lb)
	}
}
