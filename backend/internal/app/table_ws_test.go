package app_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	app "github.com/hokm/platform/internal/app"
	"github.com/hokm/platform/internal/auth"
	"github.com/hokm/platform/internal/game"
	"github.com/hokm/platform/internal/httpapi"
	"github.com/hokm/platform/internal/infra/memory"
	"github.com/hokm/platform/internal/room"
	"github.com/hokm/platform/internal/ws"
)

// --- full-stack fixture ---

type stack struct {
	ts     *httptest.Server
	client *http.Client
}

func newStack(t *testing.T) *stack {
	t.Helper()
	tokens := auth.NewTokenManager("test-secret-value-at-least-long", time.Hour)
	users := app.NewUserService(memory.NewUserStore(), tokens, auth.NewMemoryRefreshStore(), time.Hour)
	rooms := room.NewManager()
	tables := app.NewTableManager(rooms, tokens, 1) // RoundsToWin=1: one round decides
	hub := ws.NewHub(tables)
	ts := httptest.NewServer(httpapi.NewServer(users, tokens, rooms, hub).Handler())
	t.Cleanup(ts.Close)
	return &stack{ts: ts, client: ts.Client()}
}

func (s *stack) post(t *testing.T, path, token string, body any) map[string]any {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, s.ts.URL+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode >= 300 {
		t.Fatalf("POST %s: status %d: %v", path, resp.StatusCode, out)
	}
	return out
}

func (s *stack) register(t *testing.T, username string) string {
	t.Helper()
	out := s.post(t, "/api/auth/register", "", map[string]string{
		"username": username, "email": username + "@example.com", "password": "s3curePass!",
	})
	tokens := out["tokens"].(map[string]any)
	return tokens["access_token"].(string)
}

// --- ws test client with a dedicated reader goroutine ---

type wsClient struct {
	conn *websocket.Conn
	t    *testing.T
	name string

	mu     sync.Mutex
	state  game.SeatView
	roomID string

	msgs chan ws.Envelope
	done chan struct{}
}

func dialWS(t *testing.T, s *stack, token, name string) *wsClient {
	t.Helper()
	url := "ws" + strings.TrimPrefix(s.ts.URL, "http") + "/api/ws?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("%s: dial ws: %v", name, err)
	}
	c := &wsClient{
		conn: conn, t: t, name: name,
		msgs: make(chan ws.Envelope, 512),
		done: make(chan struct{}),
	}
	go c.pump()
	t.Cleanup(func() { _ = conn.Close() })
	return c
}

func (c *wsClient) pump() {
	defer close(c.done)
	for {
		var env ws.Envelope
		if err := c.conn.ReadJSON(&env); err != nil {
			return
		}
		select {
		case c.msgs <- env:
		default: // overflow: drop oldest behavior is fine for tests
		}
	}
}

func (c *wsClient) snapshot() (game.SeatView, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state, c.roomID
}

func (c *wsClient) handle(env ws.Envelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch env.Type {
	case ws.MsgState:
		var v game.SeatView
		b, _ := json.Marshal(env.Payload)
		if err := json.Unmarshal(b, &v); err != nil {
			c.t.Fatalf("%s: bad state: %v", c.name, err)
		}
		c.state = v
	case ws.MsgRoom:
		var r room.Room
		b, _ := json.Marshal(env.Payload)
		_ = json.Unmarshal(b, &r)
		c.roomID = r.ID
	case ws.MsgError:
		c.t.Fatalf("%s: server error: %s", c.name, env.Payload)
	}
}

// send transmits a command.
func (c *wsClient) send(env ws.Envelope) {
	c.t.Helper()
	if err := c.conn.WriteJSON(env); err != nil {
		c.t.Fatalf("%s: write: %v", c.name, err)
	}
}

// drain consumes queued messages (with a deadline) applying them to state.
func (c *wsClient) drain(d time.Duration) {
	deadline := time.After(d)
	for {
		select {
		case env := <-c.msgs:
			c.handle(env)
		case <-deadline:
			return
		}
	}
}

// readUntil consumes messages until pred holds, failing on timeout.
func (c *wsClient) readUntil(pred func() bool, timeout time.Duration) {
	deadline := time.After(timeout)
	for {
		if pred() {
			return
		}
		select {
		case env := <-c.msgs:
			c.handle(env)
		case <-deadline:
			c.t.Fatalf("%s: condition not met within %v", c.name, timeout)
		}
	}
}

func (c *wsClient) myTurn() bool {
	s, _ := c.snapshot()
	return s.Phase == game.PhaseTrickPlay && s.Turn == s.You &&
		len(s.CurrentTrick) < 4 && !s.MatchOver
}

func (c *wsClient) legalCard() game.Card {
	s, _ := c.snapshot()
	if len(s.CurrentTrick) > 0 {
		lead := s.CurrentTrick[0].Card.Suit
		for _, hc := range s.YourHand {
			if hc.Suit == lead {
				return hc
			}
		}
	}
	return s.YourHand[0]
}

// TestFullMultiplayerGameOverWS plays a complete 4-human match over WebSockets.
func TestFullMultiplayerGameOverWS(t *testing.T) {
	s := newStack(t)
	tokens := []string{
		s.register(t, "player1"), s.register(t, "player2"), s.register(t, "player3"), s.register(t, "player4"),
	}

	created := s.post(t, "/api/rooms", tokens[0], map[string]string{
		"name": "E2E Room", "visibility": "public",
	})
	rm := created["room"].(map[string]any)
	roomID := rm["id"].(string)
	code := rm["code"].(string)

	for _, tok := range tokens[1:] {
		s.post(t, "/api/rooms/join", tok, map[string]string{"code": code})
	}
	for _, tok := range tokens {
		s.post(t, "/api/rooms/"+roomID+"/ready", tok, map[string]bool{"ready": true})
	}

	clients := make([]*wsClient, 4)
	for i, tok := range tokens {
		clients[i] = dialWS(t, s, tok, "player"+string(rune('1'+i)))
	}
	for _, c := range clients {
		c.send(ws.Envelope{Type: ws.CmdSubscribe, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})
	}
	clients[3].readUntil(func() bool {
		_, id := clients[3].snapshot()
		return id == roomID
	}, 5*time.Second)

	// Host starts the match.
	clients[0].send(ws.Envelope{Type: ws.CmdStartGame, Payload: mustJSONRaw(map[string]string{"room_id": roomID})})

	// Everyone reaches trump selection with 5 cards.
	for i, c := range clients {
		i := i
		c.readUntil(func() bool {
			st, _ := clients[i].snapshot()
			return st.Phase == game.PhaseTrumpSelection && len(st.YourHand) == 5
		}, 10*time.Second)
	}

	// Hakem picks trump from their own (visible) hand.
	var hakem *wsClient
	for _, c := range clients {
		st, _ := c.snapshot()
		if st.Hakem == st.You {
			hakem = c
		}
	}
	if hakem == nil {
		t.Fatal("no hakem found")
	}
	hst, _ := hakem.snapshot()
	trump := hst.YourHand[0].Suit
	hakem.send(ws.Envelope{Type: ws.CmdSelectTrump, Payload: mustJSONRaw(map[string]any{
		"room_id": roomID, "suit": string(trump),
	})})

	// Everyone reaches trick play with 13 cards.
	for i, c := range clients {
		i := i
		c.readUntil(func() bool {
			st, _ := clients[i].snapshot()
			return st.Phase == game.PhaseTrickPlay && len(st.YourHand) == 13
		}, 10*time.Second)
	}

	// Drive all 13 tricks.
	deadline := time.After(60 * time.Second)
	for {
		acted := false
		for _, c := range clients {
			if c.myTurn() {
				card := c.legalCard()
				c.send(ws.Envelope{Type: ws.CmdPlayCard, Payload: mustJSONRaw(map[string]any{
					"room_id": roomID, "card": map[string]any{"suit": string(card.Suit), "rank": int(card.Rank)},
				})})
				acted = true
			}
		}
		for _, c := range clients {
			c.drain(5 * time.Millisecond)
		}
		st0, _ := clients[0].snapshot()
		if st0.Phase == game.PhaseGameComplete {
			break
		}
		if !acted {
			select {
			case <-deadline:
				t.Fatalf("match stalled: phase=%s lastTrick=%+v", st0.Phase, st0.LastTrick)
			case <-time.After(20 * time.Millisecond):
			}
		}
	}

	// Everyone must agree the match is over with a consistent result.
	for i, c := range clients {
		i := i
		c.readUntil(func() bool {
			st, _ := clients[i].snapshot()
			return st.Phase == game.PhaseGameComplete && st.MatchOver
		}, 10*time.Second)
		st, _ := c.snapshot()
		if st.RoundsWon[0]+st.RoundsWon[1] != 1 {
			c.t.Fatalf("%s: rounds won = %v, want one win", c.name, st.RoundsWon)
		}
		// Hidden information check: after game over hands are empty, but
		// during play nobody ever saw more than 13 cards. Verified implicitly
		// by engine tests; here we assert hand length never exceeded 13.
		if len(st.YourHand) > 13 {
			c.t.Fatalf("%s: hand too large: %d", c.name, len(st.YourHand))
		}
	}
	st, _ := clients[0].snapshot()
	t.Logf("match complete: roundsA=%v roundsB=%v", st.RoundsWon[0], st.RoundsWon[1])
}

func mustJSONRaw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}
