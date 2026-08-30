package app

import (
	"log/slog"
	"time"

	"github.com/hokm/platform/internal/ai"
	"github.com/hokm/platform/internal/game"
	"github.com/hokm/platform/internal/metrics"
)

// Deadline kinds carried in SeatView (Â§12): the client renders the
// countdown locally from the server deadline â€” no per-second events.
const (
	DeadlineTrump = "trump"
	DeadlineCard  = "card"
)

// TimeoutPolicy decides the automatic action when a deadline expires.
// Decisions use ONLY the acting player's own hand + public state (Â§11:
// deterministic, never random).
type TimeoutPolicy interface {
	AutomaticTrump(is ai.InformationSet) game.Suit
	AutomaticCard(is ai.InformationSet, legal []game.Card) game.Card
}

// DefaultTimeoutPolicy: deterministic trump (most trumps, tie â†’ higher
// cards) and the lowest legal card (Â§11 default LowestLegalCard).
type DefaultTimeoutPolicy struct{}

func (DefaultTimeoutPolicy) AutomaticTrump(is ai.InformationSet) game.Suit {
	count := map[game.Suit]int{}
	high := map[game.Suit]game.Rank{}
	for _, c := range is.Hand {
		count[c.Suit]++
		if c.Rank > high[c.Suit] {
			high[c.Suit] = c.Rank
		}
	}
	best := game.Suit("")
	bestCount, bestHigh := -1, game.Rank(0)
	for _, s := range game.Suits {
		if count[s] > bestCount || (count[s] == bestCount && high[s] > bestHigh) {
			best, bestCount, bestHigh = s, count[s], high[s]
		}
	}
	return best
}

func (DefaultTimeoutPolicy) AutomaticCard(is ai.InformationSet, legal []game.Card) game.Card {
	if len(legal) == 0 {
		return game.Card{}
	}
	low := legal[0]
	for _, c := range legal[1:] {
		if c.Rank < low.Rank || (c.Rank == low.Rank && c.Suit < low.Suit) {
			low = c
		}
	}
	return low
}

// legalCardsFor returns the cards the seat may legally play right now,
// computed from the seat's own view (Â§24).
func legalCardsFor(v game.SeatView) []game.Card {
	hand := v.YourHand
	if len(v.CurrentTrick) == 0 || len(hand) == 0 {
		return hand
	}
	lead := v.CurrentTrick[0].Card.Suit
	var inSuit []game.Card
	for _, c := range hand {
		if c.Suit == lead {
			inSuit = append(inSuit, c)
		}
	}
	if len(inSuit) > 0 {
		return inSuit
	}
	return hand
}

// armTakeoverLocked schedules the AI-takeover check for a disconnected
// seat when the grace period expires (Â§29). Caller holds t.mu.
func (t *Table) armTakeoverLocked(tm *TableManager, seat game.Seat) {
	deadline := t.disconnected[seat]
	time.AfterFunc(time.Until(deadline), func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		if until, ok := t.disconnected[seat]; !ok || time.Now().Before(until) {
			return // stale or reconnected
		}
		if t.sessions[seat] != nil || t.g == nil || t.g.Phase() == game.PhaseGameComplete {
			return
		}
		slog.Info("ai takeover", "room", t.RoomID, "seat", seat)
		if t.room.ChatEnabled {
			t.tm.chat.System(t.RoomID, "AI took over for a disconnected player")
		}
		t.aiLoop(tm)
		t.broadcast()
	})
}

// rescheduleTimerLocked recomputes the authoritative deadline after any
// state change. Only connected HUMAN actors get deadlines; AI seats and
// disconnected seats are handled by aiLoop/takeover (Â§41). The generation
// counter makes stale timer fires harmless (Â§42). Caller holds t.mu.
func (t *Table) rescheduleTimerLocked() {
	t.clearDeadlineLocked()
	t.timerGen++
	if t.g == nil {
		return
	}
	switch t.g.Phase() {
	case game.PhaseTrumpSelection:
		seat := t.g.Hakem()
		if t.ai[seat] != nil || t.sessions[seat] == nil {
			return
		}
		t.armLocked(DeadlineTrump, seat, t.hakemTimeout)
	case game.PhaseTrickPlay:
		turn := t.g.ViewFor(game.Seat0).Turn
		if turn == game.NoSeat || turn < 0 || int(turn) >= len(t.sessions) {
			return
		}
		if t.ai[turn] != nil || t.sessions[turn] == nil {
			return
		}
		t.armLocked(DeadlineCard, turn, t.cardTimeout)
	}
}

// armLocked arms the single table timer. Caller holds t.mu.
func (t *Table) armLocked(kind string, seat game.Seat, d time.Duration) {
	if t.timer != nil {
		t.timer.Stop()
	}
	deadline := time.Now().Add(d)
	t.deadlineUnixMs = deadline.UnixMilli()
	t.deadlineKind = kind
	gen := t.timerGen
	t.timer = time.AfterFunc(d, func() { t.onTimeout(t.tm, kind, seat, gen) })
}

// clearDeadlineLocked disarms the timer. Caller holds t.mu.
func (t *Table) clearDeadlineLocked() {
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
	t.deadlineUnixMs = 0
	t.deadlineKind = ""
}

// onTimeout performs the automatic action for an expired deadline (Â§8, Â§11)
// and continues the game. A stale generation makes the fire a no-op (Â§42).
func (t *Table) onTimeout(tm *TableManager, kind string, seat game.Seat, gen int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if gen != t.timerGen || t.g == nil {
		return // state changed since arming â€” ignore
	}
	metrics.IncHTTP("internal", "timeout:"+kind, 200) // audit trail in metrics
	is := ai.BuildInformationSet(t.g.ViewFor(seat), publicEvents(t.g.Events()))
	policy := DefaultTimeoutPolicy{}
	switch kind {
	case DeadlineTrump:
		suit := policy.AutomaticTrump(is)
		if _, err := t.g.SelectTrumpFor(seat, suit, true); err != nil {
			slog.Error("auto trump", "err", err)
			return
		}
		if _, err := t.g.DealRemainingCards(); err != nil {
			slog.Error("auto trump deal", "err", err)
			return
		}
		if t.room.ChatEnabled {
			t.tm.chat.System(t.RoomID, "trump was selected automatically")
		}
	case DeadlineCard:
		legal := legalCardsFor(t.g.ViewFor(seat))
		card := policy.AutomaticCard(is, legal)
		if card.Rank == 0 {
			return
		}
		if _, err := t.g.PlayCardFor(seat, card, true); err != nil {
			slog.Error("auto card", "err", err, "card", card)
			return
		}
		if t.room.ChatEnabled {
			t.tm.chat.System(t.RoomID, "a card was played automatically (timeout)")
		}
	}
	t.timerGen++
	t.clearDeadlineLocked()
	t.aiLoop(tm)
	t.broadcast()
}
