package ai

import (
	"github.com/hokm/platform/internal/game"
	"math/rand"
)

// PlayerStrategy decides trump and cards for one seat.
type PlayerStrategy interface {
	Name() string
	DecideTrump(is InformationSet) game.Suit
	DecideCard(is InformationSet) game.Card
}

// New returns the strategy for a difficulty level.
func New(difficulty string, rng *rand.Rand) PlayerStrategy {
	switch difficulty {
	case "easy":
		return &EasyStrategy{rng: rng}
	case "medium":
		return &MediumStrategy{rng: rng}
	case "hard":
		return &HardStrategy{rng: rng}
	case "expert":
		return &MonteCarloStrategy{base: &HardStrategy{rng: rng}, rng: rng, samples: 40, partnerAware: false}
	case "pro":
		return &MonteCarloStrategy{base: &HardStrategy{rng: rng}, rng: rng, samples: 120, partnerAware: true}
	default:
		return &MediumStrategy{rng: rng}
	}
}

// --- Easy: uniformly random legal card, arbitrary trump ---

type EasyStrategy struct{ rng *rand.Rand }

func (e *EasyStrategy) Name() string { return "easy" }

func (e *EasyStrategy) DecideTrump(is InformationSet) game.Suit {
	if len(is.Hand) == 0 {
		return game.Spades
	}
	return is.Hand[e.rng.Intn(len(is.Hand))].Suit
}

func (e *EasyStrategy) DecideCard(is InformationSet) game.Card {
	legal := LegalCards(is)
	return legal[e.rng.Intn(len(legal))]
}

// LegalCards returns the subset of the hand that is legal to play now.
func LegalCards(is InformationSet) []game.Card {
	if len(is.Trick) == 0 || !is.HasSuit(is.Lead) {
		return is.Hand
	}
	var out []game.Card
	for _, c := range is.Hand {
		if c.Suit == is.Lead {
			out = append(out, c)
		}
	}
	return out
}

// --- Medium: rule-based heuristics ---

type MediumStrategy struct{ rng *rand.Rand }

func (m *MediumStrategy) Name() string { return "medium" }

func (m *MediumStrategy) DecideTrump(is InformationSet) game.Suit {
	return mostTrumpsSuit(is)
}

// mostTrumpsSuit picks the suit with the most cards (tie → highest card).
func mostTrumpsSuit(is InformationSet) game.Suit {
	count := map[game.Suit]int{}
	high := map[game.Suit]game.Rank{}
	for _, c := range is.Hand {
		count[c.Suit]++
		if c.Rank > high[c.Suit] {
			high[c.Suit] = c.Rank
		}
	}
	best := game.Suit("")
	bestCount, bestHigh := -1, game.Rank(0)
	for _, s := range game.Suits {
		if count[s] > bestCount || (count[s] == bestCount && high[s] > bestHigh) {
			best, bestCount, bestHigh = s, count[s], high[s]
		}
	}
	return best
}

func (m *MediumStrategy) DecideCard(is InformationSet) game.Card {
	legal := LegalCards(is)
	if len(legal) == 1 {
		return legal[0]
	}
	if len(is.Trick) == 0 {
		return m.lead(is, legal)
	}
	return m.follow(is, legal)
}

func (m *MediumStrategy) lead(is InformationSet, legal []game.Card) game.Card {
	// Lead an Ace of a non-trump suit if held; otherwise lowest non-trump.
	var ace game.Card
	lowest := game.Card{Rank: game.RankAce + 1}
	lowestIsTrump := true
	for _, c := range legal {
		if c.Rank == game.RankAce && c.Suit != is.Trump {
			ace = c
			break
		}
		if c.Suit != is.Trump && c.Rank < lowest.Rank {
			lowest, lowestIsTrump = c, false
		}
	}
	if ace.Rank != 0 {
		return ace
	}
	if !lowestIsTrump {
		return lowest
	}
	// All trumps: lead lowest.
	return lowestOf(legal)
}

func (m *MediumStrategy) follow(is InformationSet, legal []game.Card) game.Card {
	bestSeat, bestCard, ok := is.CurrentBest()
	partnerWinning := ok && bestSeat == is.Partner
	// If partner already wins, dump the lowest.
	if partnerWinning {
		return lowestOf(legal)
	}
	// Try the cheapest winning card; when void in lead, consider trumping.
	var cheapest game.Card
	found := false
	for _, c := range legal {
		if c.Beats(bestCard, is.Lead, is.Trump) {
			if !found || c.Rank < cheapest.Rank {
				cheapest, found = c, true
			}
		}
	}
	if found {
		// Avoid burning the Ace of trump early for a low-value trick.
		if cheapest.Suit == is.Trump && is.Trump != is.Lead && bestCard.Rank < game.RankJack {
			return lowestOf(legal)
		}
		return cheapest
	}
	return lowestOf(legal)
}

func lowestOf(cards []game.Card) game.Card {
	low := cards[0]
	for _, c := range cards[1:] {
		if c.Rank < low.Rank {
			low = c
		}
	}
	return low
}

func highestOf(cards []game.Card) game.Card {
	high := cards[0]
	for _, c := range cards[1:] {
		if c.Rank > high.Rank {
			high = c
		}
	}
	return high
}

// --- Hard: heuristics + void inference from played cards ---

type HardStrategy struct{ rng *rand.Rand }

func (h *HardStrategy) Name() string { return "hard" }

func (h *HardStrategy) DecideTrump(is InformationSet) game.Suit {
	return mostTrumpsSuit(is)
}

func (h *HardStrategy) DecideCard(is InformationSet) game.Card {
	legal := LegalCards(is)
	if len(legal) == 1 {
		return legal[0]
	}
	if len(is.Trick) == 0 {
		return h.leadHard(is, legal)
	}
	return h.followHard(is, legal)
}

// voidSuites infers suits a seat cannot hold (they played off-suit earlier).
func (is *InformationSet) voidSeats() map[game.Seat]map[game.Suit]bool {
	voids := make(map[game.Seat]map[game.Suit]bool, 4)
	for _, tr := range is.History {
		for _, pc := range tr.Cards {
			if pc.Card.Suit != tr.LeadSuit {
				if voids[pc.Seat] == nil {
					voids[pc.Seat] = make(map[game.Suit]bool)
				}
				voids[pc.Seat][tr.LeadSuit] = true
			}
		}
	}
	return voids
}

func (h *HardStrategy) leadHard(is InformationSet, legal []game.Card) game.Card {
	voids := is.voidSeats()
	// Prefer Aces of suits where opponents are not void (more likely to win).
	var ace game.Card
	aceSafe := false
	var highNonTrump game.Card
	highSafe := false
	for _, c := range legal {
		if c.Suit == is.Trump {
			continue
		}
		risk := false
		for _, opp := range []game.Seat{(is.Me + 1) % 4, (is.Me + 3) % 4} {
			if voids[opp][c.Suit] {
				risk = true
			}
		}
		if c.Rank == game.RankAce && !risk {
			ace, aceSafe = c, true
		}
		if c.Rank > highNonTrump.Rank && !risk {
			highNonTrump, highSafe = c, true
		}
	}
	if aceSafe {
		return ace
	}
	if highSafe {
		return highNonTrump
	}
	return lowestOf(legal)
}

func (h *HardStrategy) followHard(is InformationSet, legal []game.Card) game.Card {
	bestSeat, bestCard, ok := is.CurrentBest()
	if !ok {
		return lowestOf(legal)
	}
	if bestSeat == is.Partner {
		// Partner wins: dump low, but keep aces.
		low := legal[0]
		for _, c := range legal[1:] {
			if c.Rank < low.Rank && !(c.Rank == game.RankAce && c.Suit != is.Trump) {
				low = c
			}
		}
		return low
	}
	// Cheapest card that currently wins the trick.
	var winners []game.Card
	for _, c := range legal {
		if c.Beats(bestCard, is.Lead, is.Trump) {
			winners = append(winners, c)
		}
	}
	if len(winners) > 0 {
		toPlay := 3 - len(is.Trick) // players still to act this trick
		if toPlay == 0 {
			return lowestOf(winners) // last to play: winning is safe
		}
		cheap := lowestOf(winners)
		// Don't over-trump a low lead unless the trick looks valuable.
		if cheap.Suit == is.Trump && is.Lead != is.Trump && bestCard.Rank <= game.Rank9 {
			return lowestOf(legal)
		}
		return cheap
	}
	return lowestOf(legal)
}
