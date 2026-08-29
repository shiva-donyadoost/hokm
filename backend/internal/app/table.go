package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/hokm/platform/internal/ai"
	"github.com/hokm/platform/internal/game"
	"github.com/hokm/platform/internal/rating"
	"github.com/hokm/platform/internal/room"
	"github.com/hokm/platform/internal/ws"
)

var (
	ErrTableNotFound = errors.New("app: table not found for room")
	ErrNotAllReady   = errors.New("app: not all members are ready")
	ErrNeedFourSeats = errors.New("app: room needs exactly four seated members")
	ErrNotSubscribed = errors.New("app: not subscribed to this room")
	ErrInternal      = errors.New("app: internal error")
)

// Table binds one room to one running match. Sessions are the connected
// humans; AI seats have nil sessions and are played by the AI engine.
type Table struct {
	RoomID     string
	mu         sync.Mutex
	room       room.Room // frozen membership snapshot at start
	g          *game.Game
	sessions   [4]*ws.Session // by seat; nil for AI or disconnected
	ai         [4]ai.PlayerStrategy
	rng        *rand.Rand
	sentEvents int // events already broadcast
}

// TableManager implements ws.CommandHandler and orchestrates matches.
type TableManager struct {
	mu          sync.RWMutex
	tables      map[string]*Table                 // roomID â†’ table
	subs        map[string]map[string]*ws.Session // roomID â†’ userID â†’ session
	rooms       *room.Manager
	tokens      tokenVerifier
	roundsToWin int
	chat        *ChatService
	scores      rating.ScoreStore            // nil â†’ stats not recorded
	prevMembers map[string]map[string]string // roomID â†’ userID â†’ username
}

// tokenVerifier is the subset of auth.TokenManager we need (kept narrow so
// tests can stub it).
type tokenVerifier interface {
	VerifyAccess(token string) (string, error)
}

func NewTableManager(rooms *room.Manager, tokens tokenVerifier, roundsToWin int, scores rating.ScoreStore) *TableManager {
	tm := &TableManager{
		tables:      make(map[string]*Table),
		subs:        make(map[string]map[string]*ws.Session),
		rooms:       rooms,
		tokens:      tokens,
		roundsToWin: roundsToWin,
		chat:        NewChatService(),
		scores:      scores,
		prevMembers: make(map[string]map[string]string),
	}
	rooms.SetNotifier(tm)
	tm.chat.SetSink(tm)
	return tm
}

// Chat exposes the chat service (for history on subscribe).
func (tm *TableManager) Chat() *ChatService { return tm.chat }

// ChatMessage implements ChatSink: fan chat out to room subscribers.
func (tm *TableManager) ChatMessage(msg ChatMessage) {
	tm.mu.RLock()
	subs := tm.subs[msg.RoomID]
	sessions := make([]*ws.Session, 0, len(subs))
	for _, s := range subs {
		sessions = append(sessions, s)
	}
	tm.mu.RUnlock()
	for _, s := range sessions {
		s.Send(ws.Envelope{Type: ws.MsgChat, Payload: mustMarshal(msg)})
	}
}

// RoomUpdated fans out lobby snapshots to subscribers (room.Notifier) and
// emits system chat messages for membership changes. Chat emissions happen
// after the manager lock is released to avoid re-entrant locking.
func (tm *TableManager) RoomUpdated(r room.Room) {
	var systemMsgs []string
	tm.mu.Lock()
	subs := tm.subs[r.ID]
	sessions := make([]*ws.Session, 0, len(subs))
	for _, s := range subs {
		sessions = append(sessions, s)
	}
	// Diff membership for system messages.
	current := make(map[string]string, len(r.Members))
	for _, m := range r.Members {
		current[m.UserID] = m.Username
	}
	prev := tm.prevMembers[r.ID]
	if prev != nil {
		for uid, name := range current {
			if _, was := prev[uid]; !was {
				systemMsgs = append(systemMsgs, name+" joined the room")
			}
		}
		for uid, name := range prev {
			if _, still := current[uid]; !still {
				systemMsgs = append(systemMsgs, name+" left the room")
			}
		}
	}
	tm.prevMembers[r.ID] = current
	tm.mu.Unlock()

	for _, s := range sessions {
		s.Send(ws.Envelope{Type: ws.MsgRoom, Payload: mustMarshal(r)})
	}
	for _, body := range systemMsgs {
		tm.chat.System(r.ID, body)
	}
}

// Authenticate validates the WS upgrade token.
func (tm *TableManager) Authenticate(token string) (string, string, error) {
	uid, err := tm.tokens.VerifyAccess(token)
	if err != nil {
		return "", "", fmt.Errorf("unauthorized")
	}
	return uid, uid, nil // username resolved client-side; profile endpoint exists
}

// OnDisconnect removes the session from subscriptions; the match continues
// (reconnection handling deepens in Phase 14).
func (tm *TableManager) OnDisconnect(s *ws.Session) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	for roomID, subs := range tm.subs {
		if subs[s.UserID] == s {
			delete(subs, s.UserID)
			if len(subs) == 0 {
				delete(tm.subs, roomID)
			}
		}
	}
	for _, t := range tm.tables {
		t.mu.Lock()
		for i := range t.sessions {
			if t.sessions[i] == s {
				t.sessions[i] = nil
			}
		}
		t.mu.Unlock()
	}
}

// HandleCommand routes an authenticated client command.
func (tm *TableManager) HandleCommand(s *ws.Session, env ws.Envelope) error {
	switch env.Type {
	case ws.CmdSubscribe:
		var p ws.SubscribePayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return fmt.Errorf("bad payload")
		}
		return tm.subscribe(s, p.RoomID)
	case ws.CmdStartGame:
		var p ws.StartGamePayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return fmt.Errorf("bad payload")
		}
		return tm.startGame(s, p.RoomID)
	case ws.CmdSelectTrump:
		var p ws.SelectTrumpPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return fmt.Errorf("bad payload")
		}
		return tm.selectTrump(s, p.RoomID, game.Suit(p.Suit))
	case ws.CmdPlayCard:
		var p ws.PlayCardPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return fmt.Errorf("bad payload")
		}
		return tm.playCard(s, p.RoomID, game.Card{Suit: game.Suit(p.Card.Suit), Rank: game.Rank(p.Card.Rank)})
	case ws.CmdChat:
		var p ws.ChatPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return fmt.Errorf("bad payload")
		}
		return tm.sendChat(s, p.RoomID, p.Body)
	default:
		return fmt.Errorf("unknown command %q", env.Type)
	}
}

// sendChat validates membership, appends to history, and broadcasts.
func (tm *TableManager) sendChat(s *ws.Session, roomID, body string) error {
	rm, err := tm.rooms.Get(roomID)
	if err != nil {
		return err
	}
	if !rm.InSeat(s.UserID) {
		return ErrNotSubscribed
	}
	username := s.UserID
	for _, m := range rm.Members {
		if m.UserID == s.UserID {
			username = m.Username
			break
		}
	}
	_, err = tm.chat.Send(roomID, s.UserID, username, body)
	return err
}

// subscribe registers a session for a room's updates. Membership is
// required (server-authoritative; see ADR-0004).
func (tm *TableManager) subscribe(s *ws.Session, roomID string) error {
	rm, err := tm.rooms.Get(roomID)
	if err != nil {
		return err
	}
	if !rm.InSeat(s.UserID) {
		return ErrNotSubscribed
	}
	tm.mu.Lock()
	if tm.subs[roomID] == nil {
		tm.subs[roomID] = make(map[string]*ws.Session)
	}
	tm.subs[roomID][s.UserID] = s
	var table *Table
	if t, ok := tm.tables[roomID]; ok {
		table = t
	}
	tm.mu.Unlock()

	// Lobby snapshot.
	s.Send(ws.Envelope{Type: ws.MsgRoom, Payload: mustMarshal(rm)})
	// Chat history so the client renders the recent conversation.
	for _, msg := range tm.chat.History(roomID, 50) {
		s.Send(ws.Envelope{Type: ws.MsgChat, Payload: mustMarshal(msg)})
	}
	// Live game snapshot if a table exists.
	if table != nil {
		table.mu.Lock()
		defer table.mu.Unlock()
		tm.sendStateLocked(table, s)
	}
	return nil
}

// startGame validates lobby conditions, builds the engine, and deals.
func (tm *TableManager) startGame(s *ws.Session, roomID string) error {
	rm, err := tm.rooms.Get(roomID)
	if err != nil {
		return err
	}
	if rm.HostID != s.UserID {
		return room.ErrNotHost
	}
	if rm.Status != "lobby" {
		return room.ErrGameInProgress
	}
	if len(rm.Members) != 4 {
		return ErrNeedFourSeats
	}
	for _, m := range rm.Members {
		if !m.Ready {
			return ErrNotAllReady
		}
	}
	t := &Table{RoomID: roomID, room: rm}
	var players [4]game.Player
	for _, m := range rm.Members {
		players[m.Seat] = game.Player{ID: m.UserID, Name: m.Username}
	}
	g, err := game.NewGame(players, game.Options{RoundsToWin: tm.roundsToWin})
	if err != nil {
		return err
	}
	t.g = g
	t.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, m := range rm.Members {
		if m.IsAI {
			t.ai[m.Seat] = ai.New(m.AIDifficulty, t.rng)
		}
	}
	if _, err := g.StartGame(); err != nil {
		return err
	}
	if _, err := g.SelectHakem(); err != nil {
		return err
	}
	if _, err := g.DealInitialCards(); err != nil {
		return err
	}
	tm.mu.Lock()
	tm.tables[roomID] = t
	tm.mu.Unlock()
	if err := tm.rooms.MarkStarted(roomID); err != nil {
		return err
	}
	// Bind already-subscribed sessions.
	tm.mu.RLock()
	subs := tm.subs[roomID]
	tm.mu.RUnlock()
	t.mu.Lock()
	for _, m := range rm.Members {
		if !m.IsAI {
			if sess, ok := subs[m.UserID]; ok {
				t.sessions[m.Seat] = sess
			}
		}
	}
	t.runAIAndBroadcast(tm)
	t.mu.Unlock()
	return nil
}

// selectTrump validates hakem identity via seat, runs the engine, and
// completes the deal.
func (tm *TableManager) selectTrump(s *ws.Session, roomID string, suit game.Suit) error {
	t, g, seat, err := tm.authenticatedTable(s, roomID)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if g.Hakem() != seat {
		return game.ErrNotHakem
	}
	if _, err := g.SelectTrump(suit); err != nil {
		return err
	}
	if _, err := g.DealRemainingCards(); err != nil {
		return err
	}
	t.runAIAndBroadcast(tm)
	return nil
}

// playCard validates seat identity, plays, and auto-resolves trick/round.
func (tm *TableManager) playCard(s *ws.Session, roomID string, card game.Card) error {
	t, g, seat, err := tm.authenticatedTable(s, roomID)
	if err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, err := g.PlayCard(seat, card); err != nil {
		return err
	}
	t.runAIAndBroadcast(tm)
	return nil
}

// authenticatedTable resolves the table and the caller's seat, enforcing
// membership (auth, membership; phase/turn/legality live in the engine).
func (tm *TableManager) authenticatedTable(s *ws.Session, roomID string) (*Table, *game.Game, game.Seat, error) {
	tm.mu.RLock()
	t, ok := tm.tables[roomID]
	tm.mu.RUnlock()
	if !ok {
		return nil, nil, 0, ErrTableNotFound
	}
	seat, ok := t.roomSeat(s.UserID)
	if !ok {
		return nil, nil, 0, room.ErrNotInRoom
	}
	return t, t.g, seat, nil
}

func (t *Table) roomSeat(userID string) (game.Seat, bool) {
	for _, m := range t.room.Members {
		if m.UserID == userID {
			return game.Seat(m.Seat), true
		}
	}
	return 0, false
}

// sendStateLocked pushes the per-seat view to one session. Caller holds t.mu.
func (tm *TableManager) sendStateLocked(t *Table, s *ws.Session) {
	seat, ok := t.roomSeat(s.UserID)
	if !ok {
		return
	}
	s.Send(ws.Envelope{Type: ws.MsgState, Payload: mustMarshal(t.g.ViewFor(seat))})
}

// runAIAndBroadcast advances AI seats until a human must act or the match
// ends, then pushes views + new public events to everyone. Caller holds t.mu.
func (t *Table) runAIAndBroadcast(tm *TableManager) {
	t.aiLoop(tm)
	t.broadcast()
}

// aiLoop applies engine commands for AI seats until a human turn or end.
func (t *Table) aiLoop(tm *TableManager) {
	g := t.g
	steps := 0
	for steps < 100000 {
		steps++
		switch g.Phase() {
		case game.PhaseTrumpSelection:
			seat := g.Hakem()
			if t.ai[seat] == nil {
				return
			}
			is := ai.BuildInformationSet(g.ViewFor(seat), publicEvents(g.Events()))
			if _, err := g.SelectTrump(t.ai[seat].DecideTrump(is)); err != nil {
				slog.Error("ai: select trump", "err", err)
				return
			}
			if _, err := g.DealRemainingCards(); err != nil {
				slog.Error("ai: deal remaining", "err", err)
				return
			}
		case game.PhaseTrickPlay:
			turn := g.ViewFor(game.Seat0).Turn
			if turn == game.NoSeat {
				if _, err := g.CompleteTrick(); err != nil {
					slog.Error("ai: complete trick", "err", err)
					return
				}
				continue
			}
			if t.ai[turn] == nil {
				return // human must act
			}
			is := ai.BuildInformationSet(g.ViewFor(turn), publicEvents(g.Events()))
			card := t.ai[turn].DecideCard(is)
			if _, err := g.PlayCard(turn, card); err != nil {
				slog.Error("ai: play card", "err", err, "card", card)
				return
			}
		case game.PhaseRoundComplete:
			if _, err := g.CompleteRound(); err != nil {
				slog.Error("ai: complete round", "err", err)
				return
			}
		case game.PhaseGameComplete:
			evs, err := g.CompleteGame()
			if err != nil {
				slog.Error("ai: complete game", "err", err)
			}
			tm.recordMatch(t, evs)
			return
		default:
			return
		}
	}
}

// recordMatch persists stats/ratings for a finished match (human seats only).
func (tm *TableManager) recordMatch(t *Table, evs []game.Event) {
	if tm.scores == nil || len(evs) == 0 {
		return
	}
	gd, ok := evs[len(evs)-1].Data.(game.GameCompletedData)
	if !ok || evs[len(evs)-1].Kind != game.EventGameCompleted {
		return
	}
	rec := rating.MatchRecord{
		GameID:     newID(),
		RoomID:     t.RoomID,
		RoundsWonA: gd.RoundsWonA,
		RoundsWonB: gd.RoundsWonB,
		WinnerTeam: int(gd.WinnerTeam),
	}
	for _, m := range t.room.Members {
		rec.Players = append(rec.Players, rating.MatchPlayer{
			UserID:       m.UserID,
			Username:     m.Username,
			Seat:         m.Seat,
			Team:         m.Seat % 2,
			IsAI:         m.IsAI,
			AIDifficulty: m.AIDifficulty,
		})
	}
	if err := tm.scores.ApplyMatch(rec); err != nil {
		slog.Error("record match", "err", err)
	}
}

// broadcast sends per-seat views and *new* public events to all seated
// sessions. Caller holds t.mu. Only public event kinds cross the wire.
func (t *Table) broadcast() {
	evs := t.g.Events()
	fresh := evs
	if t.sentEvents < len(evs) {
		fresh = evs[t.sentEvents:]
	} else {
		fresh = nil
	}
	t.sentEvents = len(evs)
	public := publicEvents(fresh)
	for seat, sess := range t.sessions {
		if sess == nil {
			continue
		}
		sess.Send(ws.Envelope{Type: ws.MsgState, Payload: mustMarshal(t.g.ViewFor(game.Seat(seat)))})
		for _, ev := range public {
			sess.Send(ws.Envelope{Type: ws.MsgEvents, Name: string(ev.Kind), Payload: mustMarshal(ev.Data)})
		}
	}
}

// publicEvents returns events safe for everyone (no private hands).
func publicEvents(evs []game.Event) []game.Event {
	out := make([]game.Event, 0, len(evs))
	for _, ev := range evs {
		switch ev.Kind {
		case game.EventHakemSelected, game.EventTrumpSelected, game.EventCardPlayed,
			game.EventTrickCompleted, game.EventRoundCompleted, game.EventGameCompleted,
			game.EventNextRoundStarted:
			out = append(out, ev)
		}
	}
	return out
}

func mustMarshal(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Error("app: marshal", "err", err)
		return nil
	}
	return b
}
