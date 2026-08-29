package game

// Suit is one of the four card suits.
type Suit string

const (
	Spades   Suit = "spades"
	Hearts   Suit = "hearts"
	Diamonds Suit = "diamonds"
	Clubs    Suit = "clubs"
)

// Suits lists all suits in a stable order.
var Suits = []Suit{Spades, Hearts, Diamonds, Clubs}

func (s Suit) Valid() bool {
	switch s {
	case Spades, Hearts, Diamonds, Clubs:
		return true
	}
	return false
}

// Rank is a card rank. 2 is lowest, Ace (14) is highest.
type Rank int

const (
	Rank2 Rank = iota + 2
	Rank3
	Rank4
	Rank5
	Rank6
	Rank7
	Rank8
	Rank9
	Rank10
	RankJack
	RankQueen
	RankKing
	RankAce
)

var rankSymbols = map[Rank]string{
	Rank2: "2", Rank3: "3", Rank4: "4", Rank5: "5", Rank6: "6", Rank7: "7",
	Rank8: "8", Rank9: "9", Rank10: "10", RankJack: "J", RankQueen: "Q",
	RankKing: "K", RankAce: "A",
}

func (r Rank) Valid() bool { return r >= Rank2 && r <= RankAce }

func (r Rank) String() string {
	if s, ok := rankSymbols[r]; ok {
		return s
	}
	return "?"
}

// Card is a single playing card. The zero value is not a valid card.
type Card struct {
	Suit Suit `json:"suit"`
	Rank Rank `json:"rank"`
}

func (c Card) Valid() bool { return c.Suit.Valid() && c.Rank.Valid() }

// Beats reports whether card a beats card b within a trick where lead is the
// lead suit and trump the trump suit. Exactly one of a/b beats the other when
// both share a suit-relevant comparison; off-lead, non-trump cards never beat.
func (c Card) Beats(other Card, lead, trump Suit) bool {
	if c.Suit == trump && other.Suit == trump {
		return c.Rank > other.Rank
	}
	if c.Suit == trump {
		return true
	}
	if other.Suit == trump {
		return false
	}
	if c.Suit == lead && other.Suit == lead {
		return c.Rank > other.Rank
	}
	if c.Suit == lead {
		return true
	}
	return false
}

func (c Card) String() string { return c.Rank.String() + suitSymbol(c.Suit) }

func suitSymbol(s Suit) string {
	switch s {
	case Spades:
		return "♠"
	case Hearts:
		return "♥"
	case Diamonds:
		return "♦"
	case Clubs:
		return "♣"
	}
	return "?"
}
