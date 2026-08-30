package game

// SeatView is the per-recipient projection of game state. It is the ONLY
// state representation allowed to leave the engine toward a specific
// player: it contains that seat's own hand in full and everyone else's
// hands reduced to counts. Hidden information never crosses this boundary
// (see ADR-0004 and the serialization tests).
type SeatView struct {
	Phase       Phase `json:"phase"`
	RoundNumber int   `json:"round_number"`
	Hakem       Seat  `json:"hakem"`
	Trump       Suit  `json:"trump,omitempty"`
	Turn        Seat  `json:"turn"` // NoSeat when nobody must act
	You         Seat  `json:"you"`

	YourHand   []Card `json:"your_hand,omitempty"`
	HandCounts [4]int `json:"hand_counts"`

	CurrentTrick []PlayedCard    `json:"current_trick,omitempty"`
	LastTrick    *CompletedTrick `json:"last_trick,omitempty"`

	TricksThisRound [2]int `json:"tricks_this_round"` // index by Team
	RoundsWon       [2]int `json:"rounds_won"`        // index by Team
	// RoundHistory lists each completed round's winner (impliment.md §18).
	RoundHistory []RoundResult `json:"round_history,omitempty"`
	MatchOver    bool          `json:"match_over"`

	// Deadline drives the client countdown; zero means no timer is running.
	// The server remains authoritative and auto-plays on expiry (§12).
	DeadlineUnixMs int64  `json:"deadline_unix_ms,omitempty"`
	DeadlineKind   string `json:"deadline_kind,omitempty"` // "trump" | "card"
}

// ViewFor returns the safe view for the given seat. It is a point-in-time
// snapshot; callers must treat it as read-only.
func (g *Game) ViewFor(s Seat) SeatView {
	g.mu.Lock()
	defer g.mu.Unlock()

	v := SeatView{
		Phase:           g.phase,
		RoundNumber:     g.roundNumber,
		Hakem:           g.hakem,
		Turn:            g.turn,
		You:             s,
		HandCounts:      [4]int{len(g.hands[0]), len(g.hands[1]), len(g.hands[2]), len(g.hands[3])},
		TricksThisRound: [2]int{g.tricksA, g.tricksB},
		RoundsWon:       [2]int{g.roundsA, g.roundsB},
		MatchOver:       g.phase == PhaseGameComplete,
	}
	v.RoundHistory = append(v.RoundHistory, g.roundHistory...)
	if g.trumpSet {
		v.Trump = g.trump
	}
	if s >= Seat0 && s <= Seat3 {
		hand := make([]Card, len(g.hands[s]))
		copy(hand, g.hands[s])
		v.YourHand = hand
	}
	if g.trick.Plays > 0 {
		v.CurrentTrick = make([]PlayedCard, g.trick.Plays)
		copy(v.CurrentTrick, g.trick.Cards[:g.trick.Plays])
	}
	if n := len(g.trickHistory); n > 0 {
		lt := g.trickHistory[n-1]
		v.LastTrick = &lt
	}
	return v
}
