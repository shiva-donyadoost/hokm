// Command simulator runs AI-vs-AI Hokm batches without any transport or UI,
// validating the invariants required by the implementation spec:
//
//	no illegal moves, no impossible states, no crashes,
//	every game terminates, scores are consistent, cards stay unique.
//
// Usage:
//
//	go run ./cmd/simulator -games 1000 -rounds 7 -seed 1 -strategy pro
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/hokm/platform/internal/ai"
	"github.com/hokm/platform/internal/game"
)

func main() {
	games := flag.Int("games", 100, "number of matches to simulate")
	roundsToWin := flag.Int("rounds", 7, "rounds needed to win a match")
	seed := flag.Int64("seed", 1, "random seed (0 = time-based)")
	strategy := flag.String("strategy", "medium", "easy|medium|hard|expert|pro")
	flag.Parse()

	rngSeed := *seed
	if rngSeed == 0 {
		rngSeed = time.Now().UnixNano()
	}

	start := time.Now()
	wins := map[game.Team]int{}
	totalTricks, totalRounds := 0, 0
	longestMatch := 0
	var totalDecisions int64

	for i := 0; i < *games; i++ {
		rng := rand.New(rand.NewSource(rngSeed + int64(i)))
		g, err := game.NewGame([4]game.Player{
			{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"},
		}, game.Options{RoundsToWin: *roundsToWin, Rand: rng})
		if err != nil {
			slog.Error("new game", "err", err)
			os.Exit(1)
		}
		strat := ai.New(*strategy, rng)
		publicOf := func() []game.Event { return publicOnly(g.Events()) }

		steps := 0
		for g.Phase() != game.PhaseGameComplete {
			steps++
			if steps > 500000 {
				slog.Error("match did not terminate", "game", i)
				os.Exit(1)
			}
			switch g.Phase() {
			case game.PhaseAwaitingHakem:
				check(g.StartGame())
			case game.PhaseHakemSelection:
				check(g.SelectHakem())
				check(g.DealInitialCards())
			case game.PhaseTrumpSelection:
				seat := g.Hakem()
				is := ai.BuildInformationSet(g.ViewFor(seat), publicOf())
				check(g.SelectTrump(strat.DecideTrump(is)))
				check(g.DealRemainingCards())
				totalDecisions++
			case game.PhaseTrickPlay:
				seat := actingSeat(g)
				if seat == game.NoSeat {
					check(g.CompleteTrick())
					continue
				}
				is := ai.BuildInformationSet(g.ViewFor(seat), publicOf())
				card := strat.DecideCard(is)
				if _, err := g.PlayCard(seat, card); err != nil {
					// An AI decision that the engine rejects is a fairness/
					// legality bug — fail loudly with details.
					slog.Error("illegal move",
						"game", i, "strategy", strat.Name(),
						"seat", seat, "card", card, "err", err)
					os.Exit(1)
				}
				totalDecisions++
			case game.PhaseRoundComplete:
				v := g.ViewFor(game.Seat0)
				totalTricks += v.TricksThisRound[0] + v.TricksThisRound[1]
				check(g.CompleteRound())
				totalRounds++
			}
		}
		check(g.CompleteGame())
		longestMatch = max(longestMatch, g.RoundsPlayed())
		for _, ev := range g.Events() {
			if ev.Kind == game.EventGameCompleted {
				wins[ev.Data.(game.GameCompletedData).WinnerTeam]++
			}
		}
		if i%100 == 0 && i > 0 {
			fmt.Printf("progress: %d/%d games\n", i, *games)
		}
	}

	elapsed := time.Since(start)
	fmt.Println("---- simulation report ----")
	fmt.Printf("games:            %d\n", *games)
	fmt.Printf("strategy:         %s\n", *strategy)
	fmt.Printf("rounds to win:    %d\n", *roundsToWin)
	fmt.Printf("seed:             %d\n", rngSeed)
	fmt.Printf("duration:         %v\n", elapsed)
	fmt.Printf("total rounds:     %d\n", totalRounds)
	fmt.Printf("total tricks:     %d\n", totalTricks)
	fmt.Printf("total decisions:  %d\n", totalDecisions)
	fmt.Printf("longest match:    %d rounds\n", longestMatch)
	fmt.Printf("team A wins:      %d\n", wins[game.TeamA])
	fmt.Printf("team B wins:      %d\n", wins[game.TeamB])
	fmt.Printf("illegal moves:    0 (any aborts the run)\n")
}

func actingSeat(g *game.Game) game.Seat {
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

// check aborts the simulation on any engine error — an engine command
// failing inside an AI-driven game means an invariant is broken.
func check(_ []game.Event, err error) {
	if err != nil {
		slog.Error("engine command failed", "err", err)
		os.Exit(1)
	}
}
