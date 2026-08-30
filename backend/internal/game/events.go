package game

// Event kinds emitted by engine commands. Events are the engine's output
// contract; the transport layer serializes them to clients after projecting
// per-recipient views (see view.go). Events never contain hidden info for
// the wrong recipient.
type EventKind string

const (
	EventHakemSelected     EventKind = "hakem_selected"
	EventInitialCardsDealt EventKind = "initial_cards_dealt"
	EventTrumpSelected     EventKind = "trump_selected"
	EventCardsDealt        EventKind = "cards_dealt"
	EventCardPlayed        EventKind = "card_played"
	EventTrickCompleted    EventKind = "trick_completed"
	EventRoundCompleted    EventKind = "round_completed"
	EventGameCompleted     EventKind = "game_completed"
	EventNextRoundStarted  EventKind = "next_round_started"
)

// Event is a single state transition record.
type Event struct {
	Kind EventKind `json:"kind"`
	Data any       `json:"data,omitempty"`
}

// HakemSelectedData reports who became hakem and by drawing which card.
type HakemSelectedData struct {
	Seat Seat `json:"seat"`
	Card Card `json:"card"` // the Ace that decided it
}

// CardsDealtData announces private delivery of cards to one seat.
type CardsDealtData struct {
	Seat  Seat   `json:"seat"`
	Count int    `json:"count"`
	Cards []Card `json:"cards,omitempty"` // only populated for that seat's private event
}

// TrumpSelectedData announces the chosen trump suit.
type TrumpSelectedData struct {
	Seat      Seat `json:"seat"`
	Suit      Suit `json:"suit"`
	Automatic bool `json:"automatic"` // true when chosen by the timeout policy
}

// CardPlayedData announces one card played into the current trick.
type CardPlayedData struct {
	Seat      Seat `json:"seat"`
	Card      Card `json:"card"`
	Automatic bool `json:"automatic"` // true when played by the timeout policy
}

// TrickCompletedData announces a finished trick (public info).
type TrickCompletedData struct {
	Trick CompletedTrick `json:"trick"`
}

// RoundCompletedData announces round results.
type RoundCompletedData struct {
	Number       int  `json:"number"`
	WinnerTeam   Team `json:"winner_team"`
	TricksA      int  `json:"tricks_team_a"`
	TricksB      int  `json:"tricks_team_b"`
	RoundsWonA   int  `json:"rounds_won_a"`
	RoundsWonB   int  `json:"rounds_won_b"`
	GameComplete bool `json:"game_complete"`
}

// GameCompletedData announces the match winner.
type GameCompletedData struct {
	WinnerTeam Team `json:"winner_team"`
	RoundsWonA int  `json:"rounds_won_a"`
	RoundsWonB int  `json:"rounds_won_b"`
}

// NextRoundStartedData announces the beginning of a subsequent round.
type NextRoundStartedData struct {
	Number int  `json:"number"`
	Hakem  Seat `json:"hakem"`
}
