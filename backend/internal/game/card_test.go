package game

import (
	"math/rand"
	"testing"
)

func TestNewDeckHas52UniqueCards(t *testing.T) {
	d := NewDeck()
	seen := make(map[Card]struct{}, len(d))
	if len(d) != 52 {
		t.Fatalf("deck size = %d, want 52", len(d))
	}
	for _, c := range d {
		if !c.Valid() {
			t.Fatalf("invalid card in deck: %+v", c)
		}
		if _, dup := seen[c]; dup {
			t.Fatalf("duplicate card: %+v", c)
		}
		seen[c] = struct{}{}
	}
	for _, s := range Suits {
		n := 0
		for _, c := range d {
			if c.Suit == s {
				n++
			}
		}
		if n != 13 {
			t.Fatalf("suit %s has %d cards, want 13", s, n)
		}
	}
}

func TestShufflePreservesMultiset(t *testing.T) {
	orig := NewDeck()
	shuffled := orig.Shuffle(rand.New(rand.NewSource(42)))
	if len(shuffled) != len(orig) {
		t.Fatalf("shuffled size = %d, want %d", len(shuffled), len(orig))
	}
	count := func(cards Deck) map[Card]int {
		m := make(map[Card]int, 52)
		for _, c := range cards {
			m[c]++
		}
		return m
	}
	a, b := count(orig), count(shuffled)
	for c, n := range a {
		if b[c] != n {
			t.Fatalf("card %+v count changed after shuffle", c)
		}
	}
}

func TestShuffleDeterministicWithSeed(t *testing.T) {
	a := NewDeck().Shuffle(rand.New(rand.NewSource(7)))
	b := NewDeck().Shuffle(rand.New(rand.NewSource(7)))
	if a != b {
		t.Fatal("same seed produced different shuffles")
	}
}

func TestCardBeats(t *testing.T) {
	trump := Hearts
	lead := Spades
	cases := []struct {
		name string
		a, b Card
		want bool
	}{
		{"higher trump beats lower trump", Card{Hearts, Rank10}, Card{Hearts, Rank2}, true},
		{"lower trump loses to higher trump", Card{Hearts, Rank2}, Card{Hearts, Rank10}, false},
		{"trump beats higher lead", Card{Hearts, Rank2}, Card{Spades, RankAce}, true},
		{"higher lead beats lower lead", Card{Spades, RankKing}, Card{Spades, Rank9}, true},
		{"off-suit never beats lead", Card{Diamonds, RankAce}, Card{Spades, Rank2}, false},
		{"higher trump beats lower trump vs lead", Card{Hearts, Rank3}, Card{Spades, RankAce}, true},
	}
	for _, tc := range cases {
		if got := tc.a.Beats(tc.b, lead, trump); got != tc.want {
			t.Errorf("%s: %v.Beats(%v) = %v, want %v", tc.name, tc.a, tc.b, got, tc.want)
		}
	}
}

func TestSeatHelpers(t *testing.T) {
	if NextSeat(Seat3) != Seat0 {
		t.Fatal("NextSeat(3) should wrap to 0")
	}
	if PartnerSeat(Seat0) != Seat2 || PartnerSeat(Seat1) != Seat3 {
		t.Fatal("partner seats wrong")
	}
	if TeamOf(Seat0) != TeamA || TeamOf(Seat2) != TeamA || TeamOf(Seat1) != TeamB || TeamOf(Seat3) != TeamB {
		t.Fatal("team mapping wrong")
	}
}
