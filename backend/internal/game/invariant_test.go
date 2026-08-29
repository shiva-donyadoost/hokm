package game

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
)

// --- property / invariant tests ---

// legalBot plays a complete game using ONLY the public SeatView projection
// (its own hand + public info). If any command errors, the invariant is
// broken: view-based legality must always match engine legality.
type simReport struct {
	games            int
	tricks           int
	plays            int
	rounds           int
	events           int
	trumpsBySuit     map[Suit]int
	winners          map[Team]int
	roundsPerGameSum int
}

func runOneGame(g *Game, rep *simReport) error {
	const maxSteps = 200000
	for step := 0; step < maxSteps; step++ {
		switch g.Phase() {
		case PhaseAwaitingHakem:
			if _, err := g.StartGame(); err != nil {
				return fmt.Errorf("StartGame: %w", err)
			}
		case PhaseHakemSelection:
			if _, err := g.SelectHakem(); err != nil {
				return fmt.Errorf("SelectHakem: %w", err)
			}
			if g.hakem == NoSeat {
				return errors.New("hakem not selected after SelectHakem")
			}
			if _, err := g.DealInitialCards(); err != nil {
				return fmt.Errorf("DealInitialCards: %w", err)
			}
		case PhaseTrumpSelection:
			view := g.ViewFor(g.hakem)
			if view.Trump == "" {
				if len(view.YourHand) == 0 {
					return errors.New("hakem has no cards for trump choice")
				}
				if _, err := g.SelectTrump(view.YourHand[0].Suit); err != nil {
					return fmt.Errorf("SelectTrump: %w", err)
				}
				rep.trumpsBySuit[view.YourHand[0].Suit]++
			} else {
				if _, err := g.DealRemainingCards(); err != nil {
					return fmt.Errorf("DealRemainingCards: %w", err)
				}
			}
		case PhaseTrickPlay:
			if g.trick.Full() {
				if _, err := g.CompleteTrick(); err != nil {
					return fmt.Errorf("CompleteTrick: %w", err)
				}
				rep.tricks++
				continue
			}
			view := g.ViewFor(g.turn)
			card, err := chooseLegalCard(view, g.trick)
			if err != nil {
				return err
			}
			if _, err := g.PlayCard(g.turn, card); err != nil {
				return fmt.Errorf("PlayCard %v by seat %d: %w", card, g.turn, err)
			}
			rep.plays++
		case PhaseRoundComplete:
			before := g.roundNumber
			if _, err := g.CompleteRound(); err != nil {
				return fmt.Errorf("CompleteRound: %w", err)
			}
			rep.rounds++
			// Invariants at round boundary.
			if g.tricksA+g.tricksB != 0 && g.Phase() == PhaseHakemSelection {
				return errors.New("round state not reset")
			}
			if before == g.roundNumber && g.Phase() != PhaseGameComplete {
				return errors.New("round number did not advance")
			}
			for s := Seat0; s <= Seat3; s++ {
				if len(g.hands[s]) != 0 {
					return fmt.Errorf("seat %d still holds %d cards after round", s, len(g.hands[s]))
				}
			}
		case PhaseGameComplete:
			evs, err := g.CompleteGame()
			if err != nil {
				return fmt.Errorf("CompleteGame: %w", err)
			}
			if len(evs) > 0 {
				gd := evs[0].Data.(GameCompletedData)
				rep.winners[gd.WinnerTeam]++
				need := g.opts.roundsToWin()
				if gd.RoundsWonA >= need && gd.RoundsWonB >= need {
					return errors.New("both teams reached roundsToWin")
				}
				winnerRounds := gd.RoundsWonA
				if gd.WinnerTeam == TeamB {
					winnerRounds = gd.RoundsWonB
				}
				if winnerRounds < need {
					return fmt.Errorf("winner has %d rounds, need %d", winnerRounds, need)
				}
			}
			return nil
		}
	}
	return errors.New("game did not terminate within step budget")
}

// chooseLegalCard decides using only the seat view: own hand, current trick
// (public), and lead suit.
func chooseLegalCard(v SeatView, trick Trick) (Card, error) {
	if len(v.YourHand) == 0 {
		return Card{}, fmt.Errorf("seat %d has empty hand but must play", v.You)
	}
	var lead Suit
	if trick.Plays > 0 {
		lead = trick.LeadSuit
		hasLead := false
		for _, c := range v.YourHand {
			if c.Suit == lead {
				hasLead = true
				break
			}
		}
		if hasLead {
			for _, c := range v.YourHand {
				if c.Suit == lead {
					return c, nil
				}
			}
		}
	}
	return v.YourHand[0], nil
}

func newSimGame(seed int64, roundsToWin int) (*Game, error) {
	rng := rand.New(rand.NewSource(seed))
	players := [playerCount]Player{
		{ID: fmt.Sprintf("p%d", seed), Name: "A"},
		{ID: fmt.Sprintf("q%d", seed), Name: "B"},
		{ID: fmt.Sprintf("r%d", seed), Name: "C"},
		{ID: fmt.Sprintf("s%d", seed), Name: "D"},
	}
	return NewGame(players, Options{RoundsToWin: roundsToWin, Rand: rng})
}

func TestSimulationRandomLegalGames(t *testing.T) {
	const games = 200
	rep := &simReport{trumpsBySuit: map[Suit]int{}, winners: map[Team]int{}}
	for i := 0; i < games; i++ {
		g, err := newSimGame(int64(1000+i), 7)
		if err != nil {
			t.Fatalf("game %d: NewGame: %v", i, err)
		}
		if err := runOneGame(g, rep); err != nil {
			t.Fatalf("game %d broke an invariant: %v", i, err)
		}
		// Post-game invariants.
		if g.Phase() != PhaseGameComplete {
			t.Fatalf("game %d: phase = %s after termination", i, g.Phase())
		}
	}
	rep.games = games
	if rep.plays == 0 || rep.tricks == 0 || rep.rounds == 0 {
		t.Fatalf("simulation did nothing: %+v", rep)
	}
	// Every round's tricks must sum to 13 across all games: rounds == sum of
	// trick completions / 13.
	if rep.tricks != rep.rounds*tricksPerRound {
		t.Fatalf("tricks=%d rounds=%d: tricks != rounds*13", rep.tricks, rep.rounds)
	}
	if rep.plays != rep.tricks*playerCount {
		t.Fatalf("plays=%d tricks=%d: plays != tricks*4", rep.plays, rep.tricks)
	}
	t.Logf("simulation: %d games, %d rounds, %d tricks, %d plays, winners=%v",
		rep.games, rep.rounds, rep.tricks, rep.plays, rep.winners)
}

func TestSimulationShortMatches(t *testing.T) {
	const games = 50
	rep := &simReport{trumpsBySuit: map[Suit]int{}, winners: map[Team]int{}}
	for i := 0; i < games; i++ {
		g, err := newSimGame(int64(5000+i), 1)
		if err != nil {
			t.Fatalf("game %d: NewGame: %v", i, err)
		}
		if err := runOneGame(g, rep); err != nil {
			t.Fatalf("game %d broke an invariant: %v", i, err)
		}
	}
	if rep.rounds != games {
		t.Fatalf("RoundsToWin=1: rounds=%d, want %d", rep.rounds, games)
	}
}

// driveOneRound plays from the current phase until the round completes,
// choosing legal actions only from public/own-hand information.
func driveOneRound(g *Game) error {
	const maxSteps = 50000
	for step := 0; step < maxSteps; step++ {
		switch g.Phase() {
		case PhaseHakemSelection:
			if _, err := g.SelectHakem(); err != nil {
				return err
			}
			if _, err := g.DealInitialCards(); err != nil {
				return err
			}
		case PhaseTrumpSelection:
			view := g.ViewFor(g.hakem)
			if view.Trump == "" {
				if _, err := g.SelectTrump(view.YourHand[0].Suit); err != nil {
					return err
				}
			} else if _, err := g.DealRemainingCards(); err != nil {
				return err
			}
		case PhaseTrickPlay:
			if g.trick.Full() {
				if _, err := g.CompleteTrick(); err != nil {
					return err
				}
				continue
			}
			view := g.ViewFor(g.turn)
			card, err := chooseLegalCard(view, g.trick)
			if err != nil {
				return err
			}
			if _, err := g.PlayCard(g.turn, card); err != nil {
				return err
			}
		case PhaseRoundComplete:
			return nil
		default:
			return fmt.Errorf("unexpected phase %s", g.Phase())
		}
	}
	return errors.New("round did not finish within budget")
}

// TestHakemRotation verifies the rotation rule round by round: hakem keeps
// the role on a round win and passes left on a loss.
func TestHakemRotation(t *testing.T) {
	for seed := int64(0); seed < 30; seed++ {
		g, err := newSimGame(seed, 2)
		if err != nil {
			t.Fatalf("NewGame: %v", err)
		}
		if _, err := g.StartGame(); err != nil {
			t.Fatalf("StartGame: %v", err)
		}
		if _, err := g.SelectHakem(); err != nil {
			t.Fatalf("SelectHakem: %v", err)
		}
		prev := g.hakem
		for g.Phase() != PhaseGameComplete {
			if err := driveOneRound(g); err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
			evs, err := g.CompleteRound()
			if err != nil {
				t.Fatalf("seed %d: CompleteRound: %v", seed, err)
			}
			rc := evs[0].Data.(RoundCompletedData)
			if rc.GameComplete {
				break
			}
			want := prev
			if rc.WinnerTeam != TeamOf(prev) {
				want = NextSeat(prev)
			}
			if g.hakem != want {
				t.Fatalf("seed %d: hakem = %v, want %v (winner %v, prev %v)",
					seed, g.hakem, want, rc.WinnerTeam, prev)
			}
			prev = g.hakem
		}
	}
}
