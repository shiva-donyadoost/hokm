package ai

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/hokm/platform/internal/game"
)

// aiDriver runs a complete match where every seat is played by the given
// strategy through the sanctioned public interface: per-seat views plus
// public events only.
func aiDriver(t *testing.T, strat PlayerStrategy, roundsToWin int, seed int64) *game.Game {
	t.Helper()
	g, err := game.NewGame([4]game.Player{
		{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}, {ID: "d", Name: "D"},
	}, game.Options{RoundsToWin: roundsToWin, Rand: rand.New(rand.NewSource(seed))})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	publicOf := func() []game.Event { return publicOnly(g.Events()) }
	steps := 0
	for g.Phase() != game.PhaseGameComplete {
		steps++
		if steps > 200000 {
			t.Fatal("AI game did not terminate")
		}
		switch g.Phase() {
		case game.PhaseAwaitingHakem:
			if _, err := g.StartGame(); err != nil {
				t.Fatalf("StartGame: %v", err)
			}
		case game.PhaseHakemSelection:
			if _, err := g.SelectHakem(); err != nil {
				t.Fatalf("SelectHakem: %v", err)
			}
			if _, err := g.DealInitialCards(); err != nil {
				t.Fatalf("DealInitialCards: %v", err)
			}
		case game.PhaseTrumpSelection:
			seat := g.Hakem()
			is := BuildInformationSet(g.ViewFor(seat), publicOf())
			suit := strat.DecideTrump(is)
			if _, err := g.SelectTrump(suit); err != nil {
				t.Fatalf("SelectTrump(%s): %v", suit, err)
			}
			if _, err := g.DealRemainingCards(); err != nil {
				t.Fatalf("DealRemainingCards: %v", err)
			}
		case game.PhaseTrickPlay:
			if g.ViewFor(game.Seat0).MatchOver {
				continue
			}
			seat := currentTurn(g)
			if seat == game.NoSeat {
				if _, err := g.CompleteTrick(); err != nil {
					t.Fatalf("CompleteTrick: %v", err)
				}
				continue
			}
			is := BuildInformationSet(g.ViewFor(seat), publicOf())
			card := strat.DecideCard(is)
			if _, err := g.PlayCard(seat, card); err != nil {
				t.Fatalf("%s played %v from seat %d: %v", strat.Name(), card, seat, err)
			}
		case game.PhaseRoundComplete:
			if _, err := g.CompleteRound(); err != nil {
				t.Fatalf("CompleteRound: %v", err)
			}
		}
	}
	if _, err := g.CompleteGame(); err != nil {
		t.Fatalf("CompleteGame: %v", err)
	}
	return g
}

// currentTurn reads the acting seat from public views only.
func currentTurn(g *game.Game) game.Seat {
	for s := game.Seat0; s <= game.Seat3; s++ {
		v := g.ViewFor(s)
		if v.Turn == s && len(v.CurrentTrick) < 4 {
			return s
		}
	}
	return game.NoSeat
}

func publicOnly(evs []game.Event) []game.Event {
	var out []game.Event
	for _, ev := range evs {
		switch ev.Kind {
		case game.EventHakemSelected, game.EventTrumpSelected, game.EventCardPlayed,
			game.EventTrickCompleted, game.EventRoundCompleted, game.EventGameCompleted,
			game.EventNextRoundStarted:
			out = append(out, ev)
		}
	}
	return out
}

func TestStrategiesPlayCompleteGamesLegally(t *testing.T) {
	difficulties := []string{"easy", "medium", "hard", "expert", "pro"}
	for _, d := range difficulties {
		for seed := int64(1); seed <= 3; seed++ {
			strat := New(d, rand.New(rand.NewSource(seed)))
			g := aiDriver(t, strat, 1, seed*100+int64(len(d)))
			if g.Phase() != game.PhaseGameComplete {
				t.Fatalf("%s seed %d: phase = %s", d, seed, g.Phase())
			}
			evs := g.Events()
			var completed int
			for _, ev := range evs {
				if ev.Kind == game.EventTrickCompleted {
					completed++
				}
			}
			if completed != game.TricksPerRound() {
				t.Fatalf("%s seed %d: tricks = %d, want 13", d, seed, completed)
			}
		}
	}
}

// TestInformationSetFairness verifies the projection never leaks hidden
// cards: nothing in an InformationSet may come from another seat's hand
// unless it was publicly played.
func TestInformationSetFairness(t *testing.T) {
	g, err := game.NewGame([4]game.Player{
		{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"},
	}, game.Options{RoundsToWin: 1, Rand: rand.New(rand.NewSource(9))})
	if err != nil {
		t.Fatalf("NewGame: %v", err)
	}
	if _, err := g.StartGame(); err != nil {
		t.Fatal(err)
	}
	if _, err := g.SelectHakem(); err != nil {
		t.Fatal(err)
	}
	if _, err := g.DealInitialCards(); err != nil {
		t.Fatal(err)
	}
	if _, err := g.SelectTrump(game.Spades); err != nil {
		t.Fatal(err)
	}
	if _, err := g.DealRemainingCards(); err != nil {
		t.Fatal(err)
	}
	// Play two tricks so some cards become public.
	for i := 0; i < 8; i++ {
		seat := currentTurn(g)
		if seat == game.NoSeat {
			if _, err := g.CompleteTrick(); err != nil {
				t.Fatal(err)
			}
			continue
		}
		v := g.ViewFor(seat)
		if _, err := g.PlayCard(seat, v.YourHand[0]); err != nil {
			if errors.Is(err, game.ErrMustFollowSuit) {
				for _, c := range v.YourHand {
					if c.Suit == v.CurrentTrick[0].Card.Suit {
						_, _ = g.PlayCard(seat, c)
						break
					}
				}
				continue
			}
			t.Fatalf("play: %v", err)
		}
	}
	// Build IS for seat 0 from public info only.
	is := BuildInformationSet(g.ViewFor(game.Seat0), publicOnly(g.Events()))
	// Every other seat's current hand must be invisible.
	for s := game.Seat1; s <= game.Seat3; s++ {
		for _, c := range g.ViewFor(s).YourHand {
			for _, known := range is.Hand {
				if known == c {
					t.Fatalf("IS leaked card %v from seat %d's hand", c, s)
				}
			}
			if _, played := is.PlayedSet()[c]; played {
				t.Fatalf("IS claims %v played but it is still in seat %d's hand", c, s)
			}
		}
	}
	// Own hand must be complete.
	if len(is.Hand) != len(g.ViewFor(game.Seat0).YourHand) {
		t.Fatal("IS lost own-hand cards")
	}
}
