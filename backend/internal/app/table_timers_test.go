package app_test

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	app "github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
	"github.com/hokm/platform/internal/config"
	"github.com/hokm/platform/internal/game"
	"github.com/hokm/platform/internal/httpapi"
	"github.com/hokm/platform/internal/infra/memory"
	"github.com/hokm/platform/internal/rating"
	"github.com/hokm/platform/internal/room"
	"github.com/hokm/platform/internal/ws"
)

// newTimedStack builds a stack with aggressive timeouts for timer tests:
// hakem 300ms, card 300ms, reconnect grace 5s (unused here).
func newTimedStack(t *testing.T, chatEnabled bool) *stack {
	t.Helper()
	tokens := auth.NewTokenManager("test-secret-value-at-least-long", time.Hour)
	users := app.NewUserService(memory.NewUserStore(), tokens, auth.NewMemoryRefreshStore(), time.Hour)
	rooms := room.NewManager()
	scores := rating.NewMemoryStore()
	gameCfg := config.DefaultGameConfig()
	gameCfg.HakemSelectionTimeout = 300 * time.Millisecond
	gameCfg.CardSelectionTimeouts.Medium = 300 * time.Millisecond
	gameCfg.AIMoveDelay = 30 * time.Millisecond
	gameCfg.TrickPause = 60 * time.Millisecond
	tables := app.NewTableManager(rooms, tokens, scores, gameCfg)
	hub := ws.NewHub(tables)
	ts := httptest.NewServer(httpapi.NewServer(users, tokens, rooms, hub, nil, scores).Handler())
	t.Cleanup(ts.Close)
	return &stack{ts: ts, client: ts.Client(), scores: scores, tables: tables}
}

func createTimedRoom(t *testing.T, s *stack, tok string, chatEnabled bool) string {
	t.Helper()
	created := s.post(t, "/api/rooms", tok, map[string]any{
		"name": "Timer Room", "visibility": "private",
		"round_count": 1, "game_speed": "medium", "chat_enabled": chatEnabled,
	})
	rm := created["room"].(map[string]any)
	return rm["id"].(string)
}

// joinWithAIs fills three AI seats and readies everyone; returns the human
// client subscribed to the room.
func joinWithAIs(t *testing.T, s *stack, tok, roomID string) *wsClient {
	t.Helper()
	for i := 0; i < 3; i++ {
		s.post(t, "/api/rooms/"+roomID+"/ai", tok, map[string]string{"difficulty": "medium"})
	}
	s.post(t, "/api/rooms/"+roomID+"/ready", tok, map[string]bool{"ready": true})
	c := dialWS(t, s, tok, "human")
	c.send(ws.Envelope{Type: ws.CmdSubscribe, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	c.readUntil(func() bool {
		_, id := c.snapshot()
		return id == roomID
	}, 5*time.Second)
	return c
}

// TestHakemTimeoutAutoTrump: an idle human hakem gets trump selected
// automatically after the configured timeout, with automatic=true. The
// hakem is decided by an ace draw (1/4 chance per seat), so retry rooms
// until the host is hakem.
func TestHakemTimeoutAutoTrump(t *testing.T) {
	s := newTimedStack(t, true)
	tok := s.register(t, "idle_hakem")

	var c *wsClient
	var roomID string
	for attempt := 0; attempt < 25; attempt++ {
		roomID = createTimedRoom(t, s, tok, true)
		c = joinWithAIs(t, s, tok, roomID)
		c.send(ws.Envelope{Type: ws.CmdStartGame, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
		c.readUntil(func() bool {
			st, _ := c.snapshot()
			return st.Phase == game.PhaseTrumpSelection || st.Phase == game.PhaseTrickPlay
		}, 5*time.Second)
		st, _ := c.snapshot()
		if st.Hakem == st.You && st.Phase == game.PhaseTrumpSelection {
			break
		}
		c.conn.Close() // not the hakem — try another room
		c = nil
	}
	if c == nil {
		t.Skip("host never became hakem in 25 attempts")
	}

	// DO NOT act — the server must auto-select trump and complete the deal.
	c.readUntil(func() bool {
		st, _ := c.snapshot()
		return st.Phase == game.PhaseTrickPlay
	}, 5*time.Second)

	// Verify the automatic flag on the trump event.
	c.drain(200 * time.Millisecond)
	found := false
	for _, env := range c.allEvents() {
		if env.Name == "trump_selected" {
			var d game.TrumpSelectedData
			if err := json.Unmarshal(env.Payload, &d); err == nil && d.Automatic {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no automatic trump_selected event observed")
	}
}

// TestCardTimeoutLowestLegal: a player who does not act gets the LOWEST
// legal card played automatically, marked automatic.
func TestCardTimeoutLowestLegal(t *testing.T) {
	s := newTimedStack(t, true)
	tok := s.register(t, "idle_player")
	roomID := createTimedRoom(t, s, tok, true)
	c := joinWithAIs(t, s, tok, roomID)
	c.send(ws.Envelope{Type: ws.CmdStartGame, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})

	// Get to trick play (auto-trump fires if we idle through trump phase).
	c.readUntil(func() bool {
		st, _ := c.snapshot()
		return st.Phase == game.PhaseTrickPlay
	}, 10*time.Second)

	// Wait until it's the human's turn, then compute the expected card from
	// that exact snapshot (follow-suit aware).
	c.readUntil(func() bool {
		st, _ := c.snapshot()
		return myTurnSt(st)
	}, 10*time.Second)
	before, _ := c.snapshot()
	expected := lowestLegalCard(before)

	// Idle: the timeout policy must play exactly that card.
	c.readUntil(func() bool {
		st, _ := c.snapshot()
		if st.LastTrick == nil {
			return false
		}
		for _, pc := range st.LastTrick.Cards {
			if pc.Seat == st.You {
				if pc.Card != expected {
					t.Fatalf("timeout played %v, want lowest legal %v", pc.Card, expected)
				}
				return true
			}
		}
		return false
	}, 5*time.Second)
}

// lowestLegalCard mirrors the server-side legality hint from a view.
func lowestLegalCard(v game.SeatView) game.Card {
	hand := v.YourHand
	if len(v.CurrentTrick) > 0 {
		lead := v.CurrentTrick[0].Card.Suit
		var inSuit []game.Card
		for _, c := range hand {
			if c.Suit == lead {
				inSuit = append(inSuit, c)
			}
		}
		if len(inSuit) > 0 {
			hand = inSuit
		}
	}
	low := hand[0]
	for _, c := range hand[1:] {
		if c.Rank < low.Rank || (c.Rank == low.Rank && c.Suit < low.Suit) {
			low = c
		}
	}
	return low
}

// TestChatDisabledRoom: CHAT in a chat-disabled room is rejected; history is
// not replayed on subscribe.
func TestChatDisabledRoom(t *testing.T) {
	s := newTimedStack(t, false)
	tok := s.register(t, "quiet_host")
	roomID := createTimedRoom(t, s, tok, false)
	c := joinWithAIs(t, s, tok, roomID)

	// Chat attempt must produce an ERROR envelope.
	c.send(ws.Envelope{Type: ws.CmdChat, Payload: mustJSONRaw(map[string]string{
		"room_id": roomID, "body": "anyone there?",
	})})
	deadline := time.After(3 * time.Second)
	for {
		select {
		case env := <-c.msgs:
			if env.Type == ws.MsgError {
				return
			}
		case <-deadline:
			t.Fatal("chat in disabled room was not rejected")
		}
	}
}

// TestRoundCountFromRoom: the room's round_count flows into the engine —
// the match completes only when a team wins that many rounds.
func TestRoundCountFromRoom(t *testing.T) {
	s := newTimedStack(t, true)
	tok := s.register(t, "rounds_host")
	created := s.post(t, "/api/rooms", tok, map[string]any{
		"name": "Best of 3", "visibility": "private",
		"round_count": 3, "game_speed": "fast", "chat_enabled": true,
	})
	rm := created["room"].(map[string]any)
	roomID := rm["id"].(string)
	if rc, ok := rm["round_count"].(float64); !ok || rc != 3 {
		t.Fatalf("room round_count = %v, want 3", rm["round_count"])
	}
	// joinWithAIs returns the subscribed human client; start and drive it.
	c := joinWithAIs(t, s, tok, roomID)
	c.send(ws.Envelope{Type: ws.CmdStartGame, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})

	final := driveGame(t, []*wsClient{c}, roomID, 120*time.Second)
	winnerRounds := final.RoundsWon[0]
	if final.RoundsWon[1] > winnerRounds {
		winnerRounds = final.RoundsWon[1]
	}
	if winnerRounds != 3 {
		t.Fatalf("winner rounds = %d, want 3 (round_count respected)", winnerRounds)
	}
	if len(final.RoundHistory) < 3 || len(final.RoundHistory) > 5 {
		t.Fatalf("round history = %d entries, want 3..5 for a 3-win match", len(final.RoundHistory))
	}
}
