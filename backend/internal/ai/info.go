// Package ai implements player strategies per ADR-0005. Strategies reason
// ONLY over an InformationSet projected from public state + their own hand;
// nothing in this package can reach hidden engine internals.
package ai

import (
	"github.com/hokm/platform/internal/game"
)

// InformationSet is everything a player may legitimately know at decision
// time: their own hand, public events, and counts. Building one never
// touches other players' hands.
type InformationSet struct {
	Me      game.Seat
	Partner game.Seat
	Hand    []game.Card
	Trump   game.Suit
	Lead    game.Suit // empty when leading
	Trick   []game.PlayedCard
	// HandsLeft is the number of cards each seat still holds.
	HandsLeft [4]int
	// TricksWon indexed by team.
	TricksWon [2]int
	// Played holds every card publicly played this round.
	Played []game.Card
	// History is the public completed-trick record.
	History []game.CompletedTrick
	RoundNo int
}

// CurrentBest returns the seat+card currently winning the trick.
func (is *InformationSet) CurrentBest() (game.Seat, game.Card, bool) {
	if len(is.Trick) == 0 {
		return 0, game.Card{}, false
	}
	best := is.Trick[0]
	for _, pc := range is.Trick[1:] {
		if pc.Card.Beats(best.Card, is.Lead, is.Trump) {
			best = pc
		}
	}
	return best.Seat, best.Card, true
}

// PlayedSet returns the played cards as a set.
func (is *InformationSet) PlayedSet() map[game.Card]struct{} {
	m := make(map[game.Card]struct{}, len(is.Played))
	for _, c := range is.Played {
		m[c] = struct{}{}
	}
	return m
}

// Unseen returns all cards not in own hand and not publicly played.
func (is *InformationSet) Unseen() []game.Card {
	played := is.PlayedSet()
	mine := make(map[game.Card]struct{}, len(is.Hand))
	for _, c := range is.Hand {
		mine[c] = struct{}{}
	}
	var out []game.Card
	for _, s := range game.Suits {
		for r := game.Rank2; r <= game.RankAce; r++ {
			c := game.Card{Suit: s, Rank: r}
			if _, p := played[c]; p {
				continue
			}
			if _, m := mine[c]; m {
				continue
			}
			out = append(out, c)
		}
	}
	return out
}

// HasSuit reports whether the hand holds any card of the suit.
func (is *InformationSet) HasSuit(s game.Suit) bool {
	for _, c := range is.Hand {
		if c.Suit == s {
			return true
		}
	}
	return false
}

// BuildInformationSet projects a per-seat view plus public events into an
// InformationSet. This is the single sanctioned entry point (ADR-0005).
func BuildInformationSet(v game.SeatView, public []game.Event) InformationSet {
	is := InformationSet{
		Me:        v.You,
		Partner:   game.PartnerSeat(v.You),
		Hand:      append([]game.Card(nil), v.YourHand...),
		Trump:     v.Trump,
		Trick:     append([]game.PlayedCard(nil), v.CurrentTrick...),
		TricksWon: v.TricksThisRound,
		HandsLeft: v.HandCounts,
		RoundNo:   v.RoundNumber,
	}
	if len(is.Trick) > 0 {
		is.Lead = is.Trick[0].Card.Suit
	}
	for _, ev := range public {
		switch ev.Kind {
		case game.EventCardPlayed:
			if d, ok := ev.Data.(game.CardPlayedData); ok {
				is.Played = append(is.Played, d.Card)
			}
		case game.EventTrickCompleted:
			if d, ok := ev.Data.(game.TrickCompletedData); ok {
				is.History = append(is.History, d.Trick)
			}
		}
	}
	return is
}
