package ai

import (
	"math/rand"

	"github.com/hokm/platform/internal/game"
)

// MonteCarloStrategy samples plausible opponent hands from the unseen-card
// pool, completes the current trick with a heuristic baseline, and averages
// team-trick outcomes over random-legal rollouts.
type MonteCarloStrategy struct {
	base         *HardStrategy
	rng          *rand.Rand
	samples      int
	partnerAware bool
}

func (m *MonteCarloStrategy) Name() string { return "expert/pro" }

func (m *MonteCarloStrategy) DecideTrump(is InformationSet) game.Suit {
	return mostTrumpsSuit(is)
}

func (m *MonteCarloStrategy) DecideCard(is InformationSet) game.Card {
	legal := LegalCards(is)
	if len(legal) == 1 {
		return legal[0]
	}
	if len(is.Trick) == 0 && len(is.History) < 2 {
		// Early leads: heuristic is fine and cheap.
		return m.base.DecideCard(is)
	}
	best := legal[0]
	bestScore := -1.0
	for _, c := range legal {
		score := m.evaluate(is, c)
		if score > bestScore {
			bestScore, best = score, c
		}
	}
	return best
}

// evaluate estimates expected team tricks if we play candidate.
func (m *MonteCarloStrategy) evaluate(is InformationSet, candidate game.Card) float64 {
	unseen := is.Unseen()
	// Seats with unknown hands (not me, not exhausted).
	var unknown []game.Seat
	for s := game.Seat0; s <= game.Seat3; s++ {
		if s == is.Me || is.HandsLeft[s] == 0 {
			continue
		}
		unknown = append(unknown, s)
	}
	if len(unknown) == 0 || len(unseen) == 0 {
		// Fully determined: just simulate deterministically once.
		return m.rollout(is, candidate, nil)
	}
	total := 0.0
	valid := 0
	for i := 0; i < m.samples; i++ {
		dealt, ok := dealUnknowns(is, unseen, unknown, m.rng)
		if !ok {
			continue
		}
		total += m.rollout(is, candidate, dealt)
		valid++
	}
	if valid == 0 {
		return 0
	}
	return total / float64(valid)
}

// dealUnknowns distributes unseen cards to unknown seats respecting counts
// and known voids.
func dealUnknowns(is InformationSet, unseen []game.Card, unknown []game.Seat, rng *rand.Rand) (map[game.Seat][]game.Card, bool) {
	voids := is.voidSeats()
	pool := append([]game.Card(nil), unseen...)
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	out := make(map[game.Seat][]game.Card, len(unknown))
	need := 0
	for _, s := range unknown {
		need += is.HandsLeft[s]
	}
	if need > len(pool) {
		return nil, false
	}
	// Assign card-by-card, respecting voids where possible.
	remaining := make(map[game.Seat]int, len(unknown))
	for _, s := range unknown {
		remaining[s] = is.HandsLeft[s]
	}
	for len(pool) > 0 {
		progressed := false
		for _, s := range unknown {
			if remaining[s] == 0 {
				continue
			}
			pick := -1
			for i, c := range pool {
				if !voids[s][c.Suit] {
					pick = i
					break
				}
			}
			if pick < 0 {
				pick = 0 // void constraints impossible; fall back
			}
			out[s] = append(out[s], pool[pick])
			pool = append(pool[:pick], pool[pick+1:]...)
			remaining[s]--
			progressed = true
		}
		if !progressed {
			break
		}
	}
	return out, true
}

// rollout simulates the rest of the current trick and subsequent random-legal
// play, returning our team's tricks from this point on.
func (m *MonteCarloStrategy) rollout(is InformationSet, candidate game.Card, hands map[game.Seat][]game.Card) float64 {
	trick := append([]game.PlayedCard(nil), is.Trick...)
	lead := is.Lead
	if len(trick) == 0 {
		lead = candidate.Suit
	}
	trick = append(trick, game.PlayedCard{Seat: is.Me, Card: candidate})

	live := map[game.Seat][]game.Card{is.Me: handWithout(is.Hand, candidate)}
	for s, cards := range hands {
		live[s] = append([]game.Card(nil), cards...)
	}

	turn := (is.Me + 1) % 4
	for len(trick) < 4 {
		card := randomLegal(live[turn], lead, m.rng)
		if card.Rank == 0 {
			return 0
		}
		live[turn] = handWithout(live[turn], card)
		trick = append(trick, game.PlayedCard{Seat: turn, Card: card})
		turn = (turn + 1) % 4
	}

	// Resolve this trick.
	winner := trick[0].Seat
	for _, pc := range trick[1:] {
		if pc.Card.Beats(trickWinnerCard(trick, winner), lead, is.Trump) {
			winner = pc.Seat
		}
	}
	myTeam := game.TeamOf(is.Me)
	gain := 0.0
	if game.TeamOf(winner) == myTeam {
		gain++
	}
	wonA, wonB := is.TricksWon[0], is.TricksWon[1]
	if game.TeamOf(winner) == game.TeamA {
		wonA++
	} else {
		wonB++
	}
	need := game.TricksNeededToWinRound()
	if wonA >= need || wonB >= need {
		return gain
	}
	myTricks, oppTricks := wonA, wonB
	if myTeam == game.TeamB {
		myTricks, oppTricks = wonB, wonA
	}
	nextLead := winner
	gain += float64(simulateRest(live, is.Trump, nextLead, myTeam, myTricks, oppTricks, m.rng))
	return gain
}

func trickWinnerCard(trick []game.PlayedCard, winner game.Seat) game.Card {
	for _, pc := range trick {
		if pc.Seat == winner {
			return pc.Card
		}
	}
	return trick[0].Card
}

// simulateRest plays random-legal cards until the round ends, counting the
// team's tricks.
func simulateRest(live map[game.Seat][]game.Card, trump game.Suit, lead game.Seat, myTeam game.Team, myTricks, oppTricks int, rng *rand.Rand) int {
	count := 0
	totalLeft := 0
	for s := game.Seat0; s <= game.Seat3; s++ {
		totalLeft += len(live[s])
	}
	need := game.TricksNeededToWinRound()
	for totalLeft >= 4 {
		if myTricks >= need || oppTricks >= need {
			break
		}
		var trick []game.PlayedCard
		turn := lead
		var leadSuit game.Suit
		for i := 0; i < 4; i++ {
			card := randomLegal(live[turn], leadSuit, rng)
			if card.Rank == 0 {
				break
			}
			if i == 0 {
				leadSuit = card.Suit
			}
			live[turn] = handWithout(live[turn], card)
			trick = append(trick, game.PlayedCard{Seat: turn, Card: card})
			turn = (turn + 1) % 4
		}
		if len(trick) < 4 {
			break
		}
		winner := trick[0].Seat
		for _, pc := range trick[1:] {
			if pc.Card.Beats(trickWinnerCard(trick, winner), leadSuit, trump) {
				winner = pc.Seat
			}
		}
		if game.TeamOf(winner) == myTeam {
			count++
			myTricks++
		} else {
			oppTricks++
		}
		lead = winner
		totalLeft -= 4
	}
	return count
}

func handWithout(hand []game.Card, c game.Card) []game.Card {
	for i, hc := range hand {
		if hc == c {
			out := append([]game.Card(nil), hand[:i]...)
			return append(out, hand[i+1:]...)
		}
	}
	return hand
}

func randomLegal(hand []game.Card, lead game.Suit, rng *rand.Rand) game.Card {
	if len(hand) == 0 {
		return game.Card{}
	}
	if lead != "" {
		for _, c := range hand {
			if c.Suit == lead {
				// Play a random card of the lead suit.
				var opts []game.Card
				for _, hc := range hand {
					if hc.Suit == lead {
						opts = append(opts, hc)
					}
				}
				return opts[rng.Intn(len(opts))]
			}
		}
	}
	return hand[rng.Intn(len(hand))]
}
