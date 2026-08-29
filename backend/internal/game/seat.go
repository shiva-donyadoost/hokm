package game

import "math/rand"

// Seat identifies a player's position at the table. Seats alternate between
// the two teams: seat 0 and 2 form Team A, seats 1 and 3 form Team B.
type Seat int

const (
	Seat0 Seat = iota
	Seat1
	Seat2
	Seat3
)

const (
	playerCount  = 4
	cardsPerSuit = 13
	// initialDealCount is the number of cards dealt to each player before the
	// Hakem selects trump.
	initialDealCount = 5
	// remainingDealCount is dealt after trump selection (5+8=13).
	remainingDealCount = 8
	// tricksPerRound = cardsPerSuit.
	tricksPerRound = 13
	// tricksNeededToWinRound: a team needs 7 of 13 tricks to win a round.
	tricksNeededToWinRound = 7
)

// NextSeat returns the seat to the "left" of s — the next player in turn
// order. Turn order proceeds seat0 → seat1 → seat2 → seat3 → seat0.
func NextSeat(s Seat) Seat { return Seat((int(s) + 1) % playerCount) }

// TricksPerRound exposes the tricks-per-round constant for external packages.
func TricksPerRound() int { return tricksPerRound }

// PartnerSeat returns the seat of s's partner.
func PartnerSeat(s Seat) Seat { return Seat((int(s) + 2) % playerCount) }

// TeamOf maps a seat to its team.
func TeamOf(s Seat) Team { return Team(int(s) % 2) }

// Team is one of the two fixed teams.
type Team int

const (
	TeamA Team = iota // seats 0 and 2
	TeamB             // seats 1 and 3
)

func (t Team) String() string {
	if t == TeamA {
		return "A"
	}
	return "B"
}

func (t Team) Valid() bool { return t == TeamA || t == TeamB }

// Other returns the opposing team.
func (t Team) Other() Team {
	if t == TeamA {
		return TeamB
	}
	return TeamA
}

// Deck is a full 52-card deck.
type Deck [playerCount * cardsPerSuit]Card

// NewDeck returns an ordered 52-card deck with no duplicates.
func NewDeck() Deck {
	var d Deck
	i := 0
	for _, s := range Suits {
		for r := Rank2; r <= RankAce; r++ {
			d[i] = Card{Suit: s, Rank: r}
			i++
		}
	}
	return d
}

// Shuffle randomizes deck order using the provided source.
func (d Deck) Shuffle(r *rand.Rand) Deck {
	shuffled := d
	// Fisher–Yates over a copy to keep the receiver immutable.
	for i := len(shuffled) - 1; i > 0; i-- {
		j := r.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	return shuffled
}
