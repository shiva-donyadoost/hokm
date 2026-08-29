package game

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
)

// --- test helpers ---

// parseCard parses codes like "AS" (ace of spades), "10H", "2D", "KC".
func parseCard(t *testing.T, code string) Card {
	t.Helper()
	if len(code) < 2 {
		t.Fatalf("bad card code %q", code)
	}
	var suit Suit
	switch code[len(code)-1] {
	case 'S':
		suit = Spades
	case 'H':
		suit = Hearts
	case 'D':
		suit = Diamonds
	case 'C':
		suit = Clubs
	default:
		t.Fatalf("bad suit in %q", code)
	}
	rankPart := code[:len(code)-1]
	var rank Rank
	switch rankPart {
	case "2":
		rank = Rank2
	case "3":
		rank = Rank3
	case "4":
		rank = Rank4
	case "5":
		rank = Rank5
	case "6":
		rank = Rank6
	case "7":
		rank = Rank7
	case "8":
		rank = Rank8
	case "9":
		rank = Rank9
	case "10":
		rank = Rank10
	case "J":
		rank = RankJack
	case "Q":
		rank = RankQueen
	case "K":
		rank = RankKing
	case "A":
		rank = RankAce
	default:
		t.Fatalf("bad rank in %q", code)
	}
	return Card{Suit: suit, Rank: rank}
}

// makeHands builds four 13-card hands from a list of card codes in seat
// order, filling unused cards of a standard deck into seat 0..3 round-robin.
// Codes must be unique.
func makeHands(t *testing.T, seatCodes [4][]string) [4][]Card {
	t.Helper()
	var used [4][]Card
	seen := map[Card]bool{}
	for s, codes := range seatCodes {
		for _, code := range codes {
			c := parseCard(t, code)
			if seen[c] {
				t.Fatalf("duplicate card %v", c)
			}
			seen[c] = true
			used[s] = append(used[s], c)
		}
	}
	// Fill remaining seats with leftover deck cards so every hand has 13.
	var deck Deck = NewDeck()
	next := 0
	for s := 0; s < 4; s++ {
		for len(used[s]) < 13 {
			if next >= len(deck) {
				t.Fatal("ran out of deck filling hands")
			}
			c := deck[next]
			next++
			if seen[c] {
				continue
			}
			used[s] = append(used[s], c)
			seen[c] = true
		}
	}
	for s := range used {
		if len(used[s]) != 13 {
			t.Fatalf("seat %d hand has %d cards, want 13", s, len(used[s]))
		}
	}
	return used
}

// buildDeck returns the exact deck ordering such that dealing card-by-card
// starting left of hakem produces the given hands: first 5 rounds are the
// initial deal, next 8 rounds the remaining deal. The hakem-draw reads the
// same head of the deck.
func buildDeck(t *testing.T, hakem Seat, hands [4][]Card) func(*rand.Rand) []Card {
	t.Helper()
	var out []Card
	order := []Seat{NextSeat(hakem), NextSeat(NextSeat(hakem)), NextSeat(NextSeat(NextSeat(hakem))), hakem}
	for round := 0; round < 13; round++ {
		for _, s := range order {
			if round >= len(hands[s]) {
				t.Fatalf("hand for seat %d too short", s)
			}
			out = append(out, hands[s][round])
		}
	}
	return func(*rand.Rand) []Card { return append([]Card(nil), out...) }
}

func newTestGame(t *testing.T, shuffler func(*rand.Rand) []Card, opts Options) *Game {
	t.Helper()
	players := [playerCount]Player{{ID: "p0", Name: "P0"}, {ID: "p1", Name: "P1"}, {ID: "p2", Name: "P2"}, {ID: "p3", Name: "P3"}}
	opts.DeckShuffler = shuffler
	g, err := NewGame(players, opts)
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	return g
}

// setupToPlay drives a scripted game to PhaseTrickPlay with the given hakem.
func setupToPlay(t *testing.T, g *Game, hakem Seat) {
	t.Helper()
	if _, err := g.StartGame(); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	if _, err := g.SelectHakem(); err != nil {
		t.Fatalf("SelectHakem: %v", err)
	}
	if g.Hakem() != hakem {
		t.Fatalf("hakem = %v, want %v", g.Hakem(), hakem)
	}
	if _, err := g.DealInitialCards(); err != nil {
		t.Fatalf("DealInitialCards: %v", err)
	}
	if _, err := g.SelectTrump(Hearts); err != nil {
		t.Fatalf("SelectTrump: %v", err)
	}
	if _, err := g.DealRemainingCards(); err != nil {
		t.Fatalf("DealRemainingCards: %v", err)
	}
	if g.Phase() != PhaseTrickPlay {
		t.Fatalf("phase = %s, want trick_play", g.Phase())
	}
}

// --- hakem selection ---

func TestHakemSelectionFirstAceWins(t *testing.T) {
	for want := 0; want < 4; want++ {
		// Draw order from dealer seat0: seat1, seat2, seat3, seat0, ...
		// Put the Ace in the draw position of the wanted seat; every earlier
		// position gets a non-ace filler from the seat's own hand codes.
		var codes [4][]string
		drawOrder := []Seat{Seat1, Seat2, Seat3, Seat0}
		for pos, s := range drawOrder {
			if s == Seat(want) {
				codes[s] = append(codes[s], "AS")
			} else {
				codes[s] = append(codes[s], fmt.Sprintf("%dH", 2+pos)) // unique non-ace
			}
		}
		hands := makeHands(t, codes)
		g := newTestGame(t, buildDeck(t, Seat0, hands), Options{})
		if _, err := g.StartGame(); err != nil {
			t.Fatalf("StartGame: %v", err)
		}
		evs, err := g.SelectHakem()
		if err != nil {
			t.Fatalf("SelectHakem: %v", err)
		}
		if len(evs) != 1 || evs[0].Kind != EventHakemSelected {
			t.Fatalf("expected one hakem_selected event, got %+v", evs)
		}
		data := evs[0].Data.(HakemSelectedData)
		if data.Seat != Seat(want) {
			t.Errorf("hakem = %v, want %v", data.Seat, want)
		}
		if data.Card.Rank != RankAce {
			t.Errorf("deciding card rank = %v, want ace", data.Card.Rank)
		}
		if g.Phase() != PhaseHakemSelection {
			t.Errorf("phase after select = %s, want hakem_selection", g.Phase())
		}
	}
}

// --- dealing ---

func TestDealCountsAndUniqueness(t *testing.T) {
	hands := makeHands(t, [4][]string{
		{"AS", "2H", "3D", "4C", "5S"},
		{"2S", "3H", "4D", "5C", "6S"},
		{"7S", "8H", "9D", "10C", "JS"},
		{"QS", "KH", "AD", "2C", "3S"},
	})
	// Ace of spades is seat0's first card → seat0 becomes hakem (4th draw).
	g := newTestGame(t, buildDeck(t, Seat0, hands), Options{})
	setupToPlay(t, g, Seat0)

	total := 0
	seen := map[Card]bool{}
	for s := Seat0; s <= Seat3; s++ {
		h := g.hands[s]
		if len(h) != 13 {
			t.Fatalf("seat %d has %d cards, want 13", s, len(h))
		}
		for _, c := range h {
			if seen[c] {
				t.Fatalf("card %v dealt twice", c)
			}
			seen[c] = true
			total++
		}
	}
	if total != 52 {
		t.Fatalf("total dealt = %d, want 52", total)
	}
	if len(g.deck) != 0 {
		t.Fatalf("deck has %d leftover cards", len(g.deck))
	}
}

// --- play validation ---

func TestPlayValidation(t *testing.T) {
	hands := makeHands(t, [4][]string{
		{"AS", "2H", "3D", "4C", "5S"},  // seat0 hakem, leads AS
		{"2S", "7H", "8D", "9C", "10S"}, // seat1 holds a spade
		{"KS", "JH", "QD", "AC", "2D"},  // seat2
		{"3S", "4H", "5D", "6C", "7S"},  // seat3
	})
	g := newTestGame(t, buildDeck(t, Seat0, hands), Options{})
	setupToPlay(t, g, Seat0)

	// Wrong seat tries to play first.
	if _, err := g.PlayCard(Seat1, Card{Spades, Rank2}); !errors.Is(err, ErrNotYourTurn) {
		t.Fatalf("expected ErrNotYourTurn, got %v", err)
	}
	// Unowned card.
	if _, err := g.PlayCard(Seat0, Card{Clubs, Rank2}); !errors.Is(err, ErrCardNotOwned) {
		t.Fatalf("expected ErrCardNotOwned, got %v", err)
	}
	// Hakem leads ace of spades.
	if _, err := g.PlayCard(Seat0, parseCard(t, "AS")); err != nil {
		t.Fatalf("lead play failed: %v", err)
	}
	// seat2 plays out of turn.
	if _, err := g.PlayCard(Seat2, parseCard(t, "KS")); !errors.Is(err, ErrNotYourTurn) {
		t.Fatalf("expected ErrNotYourTurn, got %v", err)
	}
	// seat1 holds spades but tries off-suit → must follow suit.
	if _, err := g.PlayCard(Seat1, parseCard(t, "7H")); !errors.Is(err, ErrMustFollowSuit) {
		t.Fatalf("expected ErrMustFollowSuit, got %v", err)
	}
	// Following with a spade is fine.
	if _, err := g.PlayCard(Seat1, parseCard(t, "2S")); err != nil {
		t.Fatalf("follow play failed: %v", err)
	}
	// seat2 has spades (KS) — off-suit attempt must fail.
	if _, err := g.PlayCard(Seat2, parseCard(t, "JH")); !errors.Is(err, ErrMustFollowSuit) {
		t.Fatalf("expected ErrMustFollowSuit, got %v", err)
	}
	if _, err := g.PlayCard(Seat2, parseCard(t, "KS")); err != nil {
		t.Fatalf("seat2 follow failed: %v", err)
	}
	// seat3 must also follow (holds spades).
	if _, err := g.PlayCard(Seat3, parseCard(t, "4H")); !errors.Is(err, ErrMustFollowSuit) {
		t.Fatalf("expected ErrMustFollowSuit, got %v", err)
	}
	if _, err := g.PlayCard(Seat3, parseCard(t, "3S")); err != nil {
		t.Fatalf("seat3 follow failed: %v", err)
	}
	// Trick is full now; further plays rejected.
	if _, err := g.PlayCard(Seat0, parseCard(t, "2H")); !errors.Is(err, ErrTrickNotFull) {
		t.Fatalf("expected ErrTrickNotFull, got %v", err)
	}
	// Winner: AS beats KS (same lead suit, higher rank).
	evs, err := g.CompleteTrick()
	if err != nil {
		t.Fatalf("CompleteTrick: %v", err)
	}
	if len(evs) != 1 || evs[0].Kind != EventTrickCompleted {
		t.Fatalf("expected trick_completed, got %+v", evs)
	}
	data := evs[0].Data.(TrickCompletedData)
	if data.Trick.Winner != Seat0 || data.Trick.WinnerTeam != TeamA {
		t.Fatalf("winner = %v/%v, want seat0/teamA", data.Trick.Winner, data.Trick.WinnerTeam)
	}
	// Winner leads next trick.
	if _, err := g.PlayCard(Seat0, parseCard(t, "2H")); err != nil {
		t.Fatalf("winner should lead next trick: %v", err)
	}
}

// --- trump selection ---

func TestTrumpValidation(t *testing.T) {
	hands := makeHands(t, [4][]string{
		{"AS", "2H", "3D", "4C", "5S"},
		{"2S", "3H", "4D", "5C", "6S"},
		{"7S", "8H", "9D", "10C", "JS"},
		{"QS", "KH", "AD", "2C", "3S"},
	})
	g := newTestGame(t, buildDeck(t, Seat0, hands), Options{})
	if _, err := g.StartGame(); err != nil {
		t.Fatalf("StartGame: %v", err)
	}
	if _, err := g.SelectTrump(Hearts); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("trump before hakem should fail, got %v", err)
	}
	if _, err := g.SelectHakem(); err != nil {
		t.Fatalf("SelectHakem: %v", err)
	}
	if _, err := g.SelectTrump(Clubs); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("trump before initial deal should fail, got %v", err)
	}
	if _, err := g.DealInitialCards(); err != nil {
		t.Fatalf("DealInitialCards: %v", err)
	}
	if _, err := g.SelectTrump(Suit("bananas")); !errors.Is(err, ErrInvalidTrump) {
		t.Fatalf("expected ErrInvalidTrump, got %v", err)
	}
	// Non-hakem cannot select: engine has no per-seat command, so verify
	// hakem identity instead.
	if g.Hakem() != Seat0 {
		t.Fatalf("hakem = %v, want seat0", g.Hakem())
	}
	if _, err := g.SelectTrump(Hearts); err != nil {
		t.Fatalf("SelectTrump: %v", err)
	}
	// Double selection rejected.
	if _, err := g.SelectTrump(Spades); !errors.Is(err, ErrWrongPhase) {
		t.Fatalf("double trump should fail, got %v", err)
	}
	if _, err := g.DealRemainingCards(); err != nil {
		t.Fatalf("DealRemainingCards: %v", err)
	}
	if g.Trump() != Hearts {
		t.Fatalf("trump = %v, want hearts", g.Trump())
	}
}

// --- round & match completion ---

func TestRoundCompletionAndMatchWin(t *testing.T) {
	hands := makeHands(t, [4][]string{
		{"AS", "2H", "3D", "4C", "5S"},
		{"2S", "3H", "4D", "5C", "6S"},
		{"7S", "8H", "9D", "10C", "JS"},
		{"QS", "KH", "AD", "2C", "3S"},
	})
	g := newTestGame(t, buildDeck(t, Seat0, hands), Options{RoundsToWin: 1})
	setupToPlay(t, g, Seat0)

	// Play all 13 tricks with a legal bot.
	playRoundLegally(t, g)
	if g.Phase() != PhaseRoundComplete {
		t.Fatalf("phase = %s, want round_complete", g.Phase())
	}
	if g.tricksA+g.tricksB != 13 {
		t.Fatalf("tricks sum = %d, want 13", g.tricksA+g.tricksB)
	}
	// Someone must have ≥7 tricks.
	if g.tricksA < 7 && g.tricksB < 7 {
		t.Fatalf("no team reached 7 tricks: A=%d B=%d", g.tricksA, g.tricksB)
	}
	evs, err := g.CompleteRound()
	if err != nil {
		t.Fatalf("CompleteRound: %v", err)
	}
	rc := evs[0].Data.(RoundCompletedData)
	if !rc.GameComplete {
		t.Fatal("RoundsToWin=1 should complete the match")
	}
	if g.Phase() != PhaseGameComplete {
		t.Fatalf("phase = %s, want game_complete", g.Phase())
	}
	fin, err := g.CompleteGame()
	if err != nil {
		t.Fatalf("CompleteGame: %v", err)
	}
	gd := fin[0].Data.(GameCompletedData)
	want := TeamA
	if rc.WinnerTeam == TeamA {
		want = TeamA
	} else {
		want = TeamB
	}
	if gd.WinnerTeam != want {
		t.Fatalf("winner = %v, want %v", gd.WinnerTeam, want)
	}
	// Idempotent second call.
	if _, err := g.CompleteGame(); err != nil {
		t.Fatalf("CompleteGame should be idempotent, got %v", err)
	}
}

// playRoundLegally plays all 13 tricks of the current round choosing the
// first legal card per seat, asserting no command ever errors.
func playRoundLegally(t *testing.T, g *Game) {
	t.Helper()
	for tricks := 0; tricks < tricksPerRound; tricks++ {
		for play := 0; play < playerCount; play++ {
			turn := g.turn
			hand := g.hands[turn]
			played := false
			for _, c := range hand {
				if _, err := g.PlayCard(turn, c); err == nil {
					played = true
					break
				} else if !errors.Is(err, ErrMustFollowSuit) {
					t.Fatalf("unexpected error playing %v: %v", c, err)
				}
			}
			if !played {
				t.Fatalf("no legal card for seat %d (hand %v, lead %s)", turn, hand, g.trick.LeadSuit)
			}
		}
		if _, err := g.CompleteTrick(); err != nil {
			t.Fatalf("CompleteTrick: %v", err)
		}
	}
}
