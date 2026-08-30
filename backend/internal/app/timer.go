package app

import (
	"log/slog"
	"time"

	"github.com/hokm/platform/internal/ai"
	"github.com/hokm/platform/internal/game"
	"github.com/hokm/platform/internal/metrics"
)

// Deadline kinds carried in SeatView (§12): the client renders the
// countdown locally from the server deadline — no per-second events.
const (
	DeadlineTrump = "trump"
	DeadlineCard  = "card"
)

// Automatic step kinds (presentation pacing, §13/§39): each performs exactly
// ONE engine action, then broadcasts - the next step is scheduled after the
// configured delay so every play stays visually readable (AI vs AI tables
// advance one card per AI move delay).
const (
	StepDeal          = "deal"           // deal next round (hakem known)
	StepTrump         = "ai_trump"       // AI hakem selects trump + deals
	StepCard          = "ai_card"        // AI plays a card
	StepCompleteTrick = "complete_trick" // resolve a full trick (winner reveal window)
	StepTakeoverTrump = "takeover_trump" // fallback AI trump for absent human
	StepTakeoverCard  = "takeover_card"  // fallback AI card for absent human
)

// TimeoutPolicy decides the automatic action when a deadline expires.
// Decisions use ONLY the acting player's own hand + public state (§11:
// deterministic, never random).
type TimeoutPolicy interface {
	AutomaticTrump(is ai.InformationSet) game.Suit
	AutomaticCard(is ai.InformationSet, legal []game.Card) game.Card
}

// DefaultTimeoutPolicy: deterministic trump (most trumps, tie -> higher
// cards) and the lowest legal card (§11 default LowestLegalCard).
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
// computed from the seat's own view (§24).
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
// seat when the grace period expires (§29). Caller holds t.mu.
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
		t.broadcast(tm) // schedules the paced takeover steps
	})
}

// rescheduleTimerLocked recomputes the authoritative HUMAN deadline after
// any state change. AI seats and disconnected seats are handled by the
// step scheduler instead (§41). Caller holds t.mu.
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

// armLocked arms the human deadline timer. Caller holds t.mu.
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

// clearDeadlineLocked disarms the deadline timer. Caller holds t.mu.
func (t *Table) clearDeadlineLocked() {
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
	t.deadlineUnixMs = 0
	t.deadlineKind = ""
}

// onTimeout performs the automatic action for an expired human deadline
// (§8, §11) and continues the game. A stale generation makes the fire a
// no-op (§42).
func (t *Table) onTimeout(tm *TableManager, kind string, seat game.Seat, gen int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if gen != t.timerGen || t.g == nil {
		return // state changed since arming - ignore
	}
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
	t.broadcast(tm)
}

// scheduleNextLocked arms the next paced automatic step after any state
// change (§13/§39): AI moves wait aiMoveDelay; a full trick stays on the
// table for trickPause before the winner resolves. Caller holds t.mu.
func (t *Table) scheduleNextLocked() {
	if t.stepTimer != nil {
		t.stepTimer.Stop()
		t.stepTimer = nil
	}
	if t.g == nil {
		return
	}
	kind := ""
	seat := game.NoSeat
	delay := t.aiMoveDelay
	switch t.g.Phase() {
	case game.PhaseHakemSelection:
		kind = StepDeal
		delay = 0
	case game.PhaseTrumpSelection:
		h := t.g.Hakem()
		switch {
		case t.ai[h] != nil:
			kind, seat = StepTrump, h
		case t.takeoverDue(h):
			kind, seat = StepTakeoverTrump, h
		default:
			return // human hakem: deadline timer governs
		}
	case game.PhaseTrickPlay:
		turn := t.g.ViewFor(game.Seat0).Turn
		if turn == game.NoSeat || turn < 0 {
			// Trick full: hold the winner reveal window on the table (§14).
			kind = StepCompleteTrick
			delay = t.trickPause
			break
		}
		switch {
		case t.ai[turn] != nil:
			kind, seat = StepCard, turn
		case t.takeoverDue(turn):
			kind, seat = StepTakeoverCard, turn
		default:
			return // human must act
		}
	default:
		return
	}
	gen := t.stepGen
	t.stepTimer = time.AfterFunc(delay, func() { t.onStep(t.tm, kind, seat, gen) })
}

// onStep performs exactly ONE automatic step then broadcasts; the next
// step (if any) is scheduled by broadcast. A stale generation makes the
// fire a no-op (§42). Caller holds t.mu via lock below.
func (t *Table) onStep(tm *TableManager, kind string, seat game.Seat, gen int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if gen != t.stepGen || t.g == nil {
		return // state changed since scheduling - ignore
	}
	t.stepGen++
	is := ai.BuildInformationSet(t.g.ViewFor(seat), publicEvents(t.g.Events()))
	switch kind {
	case StepDeal:
		if _, err := t.g.SelectHakem(); err != nil {
			slog.Error("step: select hakem", "err", err)
			return
		}
		if _, err := t.g.DealInitialCards(); err != nil {
			slog.Error("step: deal initial", "err", err)
			return
		}
	case StepTrump:
		start := time.Now()
		suit := t.ai[seat].DecideTrump(is)
		metrics.ObserveAIDecision(time.Since(start).Nanoseconds())
		if _, err := t.g.SelectTrumpFor(seat, suit, false); err != nil {
			slog.Error("step: ai trump", "err", err)
			return
		}
		if _, err := t.g.DealRemainingCards(); err != nil {
			slog.Error("step: deal remaining", "err", err)
			return
		}
	case StepTakeoverTrump:
		if t.sessions[seat] != nil {
			break // reconnected - human acts
		}
		suit := DefaultTimeoutPolicy{}.AutomaticTrump(is)
		if _, err := t.g.SelectTrumpFor(seat, suit, true); err != nil {
			slog.Error("step: takeover trump", "err", err)
			return
		}
		if _, err := t.g.DealRemainingCards(); err != nil {
			slog.Error("step: takeover deal", "err", err)
			return
		}
	case StepCard:
		start := time.Now()
		card := t.ai[seat].DecideCard(is)
		metrics.ObserveAIDecision(time.Since(start).Nanoseconds())
		if _, err := t.g.PlayCardFor(seat, card, false); err != nil {
			slog.Error("step: ai card", "err", err, "card", card)
			return
		}
	case StepTakeoverCard:
		if t.sessions[seat] != nil {
			break // reconnected - human acts
		}
		card := t.fallback.DecideCard(is)
		if _, err := t.g.PlayCardFor(seat, card, true); err != nil {
			slog.Error("step: takeover card", "err", err, "card", card)
			return
		}
	case StepCompleteTrick:
		if _, err := t.g.CompleteTrick(); err != nil {
			slog.Error("step: complete trick", "err", err)
			return
		}
		if t.g.Phase() == game.PhaseRoundComplete {
			if _, err := t.g.CompleteRound(); err != nil {
				slog.Error("step: complete round", "err", err)
				return
			}
			if t.g.Phase() == game.PhaseGameComplete {
				evs, err := t.g.CompleteGame()
				if err != nil {
					slog.Error("step: complete game", "err", err)
				}
				t.tm.recordMatch(t, evs)
			}
		}
	}
	t.broadcast(tm)
}
