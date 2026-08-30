package game

// PlayedCard is a card tied to the seat that played it.
type PlayedCard struct {
	Seat Seat `json:"seat"`
	Card Card `json:"card"`
}

// Trick is one trick in progress: up to four played cards.
type Trick struct {
	LeadSuit Suit
	Cards    [playerCount]PlayedCard
	Plays    int
}

// Lead returns the seat that leads this trick (index 0 of the plays).
func (t *Trick) Lead() Seat { return t.Cards[0].Seat }

// Full reports whether all four players have played.
func (t *Trick) Full() bool { return t.Plays == playerCount }

// Winner returns the seat that wins the trick given the trump suit.
// Assumes the trick is full.
func (t *Trick) Winner(trump Suit) Seat {
	best := t.Cards[0]
	for i := 1; i < t.Plays; i++ {
		pc := t.Cards[i]
		if pc.Card.Beats(best.Card, t.LeadSuit, trump) {
			best = pc
		}
	}
	return best.Seat
}

// CompletedTrick is an immutable record of a finished trick.
type CompletedTrick struct {
	Number     int          `json:"number"` // 1-based within the round
	LeadSuit   Suit         `json:"lead_suit"`
	Cards      []PlayedCard `json:"cards"`
	Winner     Seat         `json:"winner"`
	WinnerTeam Team         `json:"winner_team"`
}

// RoundResult records who won a completed round (§18).
type RoundResult struct {
	Number     int  `json:"number"`
	WinnerTeam Team `json:"winner_team"`
}
