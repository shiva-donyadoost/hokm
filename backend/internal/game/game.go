package game

import (
	"math/rand"
	"sync"
	"time"
)

// Player is a seat participant identified by an opaque ID owned by the
// upper layers (user ID or AI id). The engine never interprets it.
type Player struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Options configures a match. Zero values select defaults.
type Options struct {
	// RoundsToWin is how many round wins decide the match (default 7).
	RoundsToWin int
	// Rand drives all shuffles/draws. Nil means a time-seeded source.
	// Inject a seeded source for deterministic tests.
	Rand *rand.Rand
	// KeepHakemOnLoss: when false (default, standard Iranian rule), the
	// hakem passes to the next seat if their team loses a round; when true
	// the original hakem keeps hakemship for the whole match. The
	// traditional rule has regional variants (docs/GAME_RULES.md).
	KeepHakemOnLoss bool
	// DeckShuffler produces the ordered deck for each shuffle (hakem draw,
	// initial deal). Nil uses a standard shuffled 52-card deck. Tests inject
	// fixed orderings for determinism.
	DeckShuffler func(r *rand.Rand) []Card
}

func (o *Options) roundsToWin() int {
	if o.RoundsToWin <= 0 {
		return 7
	}
	return o.RoundsToWin
}

// NoSeat marks "no turn / no seat".
const NoSeat Seat = -1

// Game is a full Hokm match: hakem selection, trump, rounds of 13 tricks,
// and match scoring. It is room-independent: transport layers attach
// player IDs to seats and serialize events.
//
// All state changes flow through the exported command methods. Each command
// validates its preconditions and returns the events it produced.
type Game struct {
	mu      sync.Mutex
	opts    Options
	rng     *rand.Rand
	players [playerCount]Player

	phase    Phase
	dealer   Seat // seat that starts the hakem-selection draw
	hakem    Seat // NoSeat until selected
	trump    Suit
	trumpSet bool

	hands [playerCount][]Card
	deck  []Card // undealt cards

	turn Seat

	trick            Trick
	tricksPlayed     int
	trickHistory     []CompletedTrick
	tricksA, tricksB int
	roundNumber      int
	roundsA, roundsB int

	gameCompleted bool
	eventLog      []Event
}

// NewGame creates a match for four distinct players. The first player
// listed is dealer for the hakem selection; play order is seat0 → 1 → 2 → 3.
func NewGame(players [playerCount]Player, opts Options) (*Game, error) {
	seen := make(map[string]struct{}, playerCount)
	for _, p := range players {
		if p.ID == "" {
			return nil, ErrInvalidPlayerCount
		}
		if _, dup := seen[p.ID]; dup {
			return nil, ErrDuplicatePlayer
		}
		seen[p.ID] = struct{}{}
	}
	if opts.Rand == nil {
		opts.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return &Game{
		opts:    opts,
		rng:     opts.Rand,
		players: players,
		phase:   PhaseAwaitingHakem,
		dealer:  Seat0,
		hakem:   NoSeat,
		turn:    NoSeat,
	}, nil
}

// Phase returns the current phase.
func (g *Game) Phase() Phase {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.phase
}

// Hakem returns the current hakem seat or NoSeat.
func (g *Game) Hakem() Seat {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.hakem
}

// Trump returns the selected trump suit (empty string if unset).
func (g *Game) Trump() Suit {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.trump
}

// Events returns a copy of the full event log since the match started.
func (g *Game) Events() []Event {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := make([]Event, len(g.eventLog))
	copy(out, g.eventLog)
	return out
}

// RoundsPlayed returns the total number of completed rounds.
func (g *Game) RoundsPlayed() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.roundsA + g.roundsB
}

// record appends events to the log and passes them through.
func (g *Game) record(evs []Event) []Event {
	g.eventLog = append(g.eventLog, evs...)
	return evs
}

// SeatOf maps a player ID to its seat.
func (g *Game) SeatOf(playerID string) (Seat, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for s, p := range g.players {
		if p.ID == playerID {
			return Seat(s), true
		}
	}
	return NoSeat, false
}

// StartGame moves the match from setup into hakem selection and shuffles
// the deck for the first round.
func (g *Game) StartGame() ([]Event, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != PhaseAwaitingHakem {
		return nil, ErrWrongPhase
	}
	g.roundNumber = 1
	g.deck = g.nextDeck()
	g.phase = PhaseHakemSelection
	evs := []Event{{Kind: EventNextRoundStarted, Data: NextRoundStartedData{Number: 1}}}
	return g.record(evs), nil
}

// SelectHakem runs the ace-draw: cards are turned one at a time starting
// left of the dealer until an Ace appears; that seat becomes hakem.
// In later rounds the hakem is already known (rotation) and this call
// is a no-op returning no events.
func (g *Game) SelectHakem() ([]Event, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != PhaseHakemSelection {
		return nil, ErrWrongPhase
	}
	if g.hakem != NoSeat {
		// Hakem already determined by rotation at round start.
		return nil, nil
	}
	var aceCard Card
	hakem := NoSeat
	seat := NextSeat(g.dealer)
	for hakem == NoSeat {
		if len(g.deck) == 0 {
			g.deck = g.nextDeck()
		}
		c := g.deck[0]
		g.deck = g.deck[1:]
		if c.Rank == RankAce {
			hakem, aceCard = seat, c
		}
		seat = NextSeat(seat)
	}
	g.hakem = hakem
	evs := []Event{{Kind: EventHakemSelected, Data: HakemSelectedData{Seat: hakem, Card: aceCard}}}
	return g.record(evs), nil
}

// DealInitialCards deals five cards to each player starting left of the
// hakem, card by card.
func (g *Game) DealInitialCards() ([]Event, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != PhaseHakemSelection || g.hakem == NoSeat {
		return nil, ErrWrongPhase
	}
	g.deck = g.nextDeck() // fold any hakem-draw cards back in
	events := g.dealLocked(EventInitialCardsDealt, initialDealCount)
	g.phase = PhaseTrumpSelection
	return g.record(events), nil
}

// SelectTrump lets the hakem choose the trump suit before the remaining
// cards are dealt.
func (g *Game) SelectTrump(suit Suit) ([]Event, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != PhaseTrumpSelection {
		return nil, ErrWrongPhase
	}
	if len(g.hands[g.hakem]) == 0 || g.trumpSet {
		return nil, ErrWrongPhase
	}
	if !suit.Valid() {
		return nil, ErrInvalidTrump
	}
	g.trump = suit
	g.trumpSet = true
	evs := []Event{{Kind: EventTrumpSelected, Data: TrumpSelectedData{Seat: g.hakem, Suit: suit}}}
	return g.record(evs), nil
}

// DealRemainingCards deals the remaining eight cards to each player.
func (g *Game) DealRemainingCards() ([]Event, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != PhaseTrumpSelection || !g.trumpSet {
		return nil, ErrWrongPhase
	}
	events := g.dealLocked(EventCardsDealt, remainingDealCount)
	g.phase = PhaseTrickPlay
	g.turn = g.hakem // hakem leads the first trick
	return g.record(events), nil
}

// PlayCard plays a card from the seat's hand into the current trick.
func (g *Game) PlayCard(s Seat, c Card) ([]Event, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != PhaseTrickPlay {
		return nil, ErrWrongPhase
	}
	if g.trick.Full() {
		return nil, ErrTrickNotFull
	}
	if g.turn != s {
		return nil, ErrNotYourTurn
	}
	if !c.Valid() || !g.handContains(s, c) {
		return nil, ErrCardNotOwned
	}
	if g.trick.Plays > 0 && c.Suit != g.trick.LeadSuit && g.handHasSuit(s, g.trick.LeadSuit) {
		return nil, ErrMustFollowSuit
	}
	if !g.removeCard(s, c) {
		return nil, ErrCardNotOwned
	}
	if g.trick.Plays == 0 {
		g.trick.LeadSuit = c.Suit // first card of the trick sets the lead
	}
	g.trick.Cards[g.trick.Plays] = PlayedCard{Seat: s, Card: c}
	g.trick.Plays++
	if g.trick.Full() {
		g.turn = NoSeat
	} else {
		g.turn = NextSeat(s)
	}
	evs := []Event{{Kind: EventCardPlayed, Data: CardPlayedData{Seat: s, Card: c}}}
	return g.record(evs), nil
}

// CompleteTrick resolves a full trick, records it, and either hands the
// lead to the winner or — after the 13th trick — moves the game to
// PhaseRoundComplete.
func (g *Game) CompleteTrick() ([]Event, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != PhaseTrickPlay {
		return nil, ErrWrongPhase
	}
	if !g.trick.Full() {
		return nil, ErrTrickNotFull
	}
	winner := g.trick.Winner(g.trump)
	played := make([]PlayedCard, playerCount)
	copy(played, g.trick.Cards[:])
	ct := CompletedTrick{
		Number:     g.tricksPlayed + 1,
		LeadSuit:   g.trick.LeadSuit,
		Cards:      played,
		Winner:     winner,
		WinnerTeam: TeamOf(winner),
	}
	g.trickHistory = append(g.trickHistory, ct)
	g.tricksPlayed++
	if ct.WinnerTeam == TeamA {
		g.tricksA++
	} else {
		g.tricksB++
	}
	events := []Event{{Kind: EventTrickCompleted, Data: TrickCompletedData{Trick: ct}}}
	g.record(events)
	g.trick = Trick{}
	if g.tricksPlayed == tricksPerRound {
		g.phase = PhaseRoundComplete
		g.turn = NoSeat
	} else {
		g.turn = winner
	}
	return events, nil
}

// CompleteRound closes the round, updates match score, rotates hakem per
// options, and either starts the next round or completes the match.
func (g *Game) CompleteRound() ([]Event, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != PhaseRoundComplete {
		return nil, ErrWrongPhase
	}
	winner := TeamA
	if g.tricksB > g.tricksA {
		winner = TeamB
	}
	if winner == TeamA {
		g.roundsA++
	} else {
		g.roundsB++
	}
	matchOver := g.roundsA >= g.opts.roundsToWin() || g.roundsB >= g.opts.roundsToWin()
	events := []Event{{
		Kind: EventRoundCompleted,
		Data: RoundCompletedData{
			Number:       g.roundNumber,
			WinnerTeam:   winner,
			TricksA:      g.tricksA,
			TricksB:      g.tricksB,
			RoundsWonA:   g.roundsA,
			RoundsWonB:   g.roundsB,
			GameComplete: matchOver,
		},
	}}
	if matchOver {
		g.phase = PhaseGameComplete
		return g.record(events), nil
	}
	// Hakem rotation: hakem keeps the role while their team wins; on a
	// loss it passes to the next seat (standard rule, configurable).
	if !g.opts.KeepHakemOnLoss && winner != TeamOf(g.hakem) {
		g.hakem = NextSeat(g.hakem)
	}
	g.dealer = g.hakem
	g.roundNumber++
	g.resetRoundLocked()
	g.phase = PhaseHakemSelection
	events = append(events, Event{
		Kind: EventNextRoundStarted,
		Data: NextRoundStartedData{Number: g.roundNumber, Hakem: g.hakem},
	})
	return g.record(events), nil
}

// CompleteGame finalizes a decided match and emits the terminal event.
// It is idempotent once the game is complete.
func (g *Game) CompleteGame() ([]Event, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.phase != PhaseGameComplete {
		return nil, ErrGameNotComplete
	}
	if g.gameCompleted {
		return nil, nil
	}
	winner := TeamA
	if g.roundsB > g.roundsA {
		winner = TeamB
	}
	g.gameCompleted = true
	return g.record([]Event{{
		Kind: EventGameCompleted,
		Data: GameCompletedData{WinnerTeam: winner, RoundsWonA: g.roundsA, RoundsWonB: g.roundsB},
	}}), nil
}

// --- internals ---

// nextDeck produces the next deck ordering via the injected shuffler or a
// standard shuffle.
func (g *Game) nextDeck() []Card {
	if g.opts.DeckShuffler != nil {
		return g.opts.DeckShuffler(g.rng)
	}
	return shuffledStandard(g.rng)
}

func shuffledStandard(r *rand.Rand) []Card {
	d := NewDeck().Shuffle(r)
	out := make([]Card, len(d))
	copy(out, d[:])
	return out
}

func (g *Game) handContains(s Seat, c Card) bool {
	for _, hc := range g.hands[s] {
		if hc == c {
			return true
		}
	}
	return false
}

func (g *Game) handHasSuit(s Seat, suit Suit) bool {
	for _, hc := range g.hands[s] {
		if hc.Suit == suit {
			return true
		}
	}
	return false
}

func (g *Game) removeCard(s Seat, c Card) bool {
	h := g.hands[s]
	for i, hc := range h {
		if hc == c {
			g.hands[s] = append(h[:i], h[i+1:]...)
			return true
		}
	}
	return false
}

// dealLocked deals n cards to each player card-by-card starting left of the
// hakem, emitting one private CardsDealt event per seat.
func (g *Game) dealLocked(kind EventKind, n int) []Event {
	for i := 0; i < n; i++ {
		for k := 0; k < playerCount; k++ {
			seat := Seat((int(g.hakem) + 1 + k) % playerCount)
			c := g.deck[0]
			g.deck = g.deck[1:]
			g.hands[seat] = append(g.hands[seat], c)
		}
	}
	events := make([]Event, 0, playerCount)
	for k := 0; k < playerCount; k++ {
		seat := Seat((int(g.hakem) + 1 + k) % playerCount)
		hand := make([]Card, len(g.hands[seat]))
		copy(hand, g.hands[seat])
		events = append(events, Event{
			Kind: kind,
			Data: CardsDealtData{Seat: seat, Count: len(hand), Cards: hand},
		})
	}
	return events
}

func (g *Game) resetRoundLocked() {
	g.hands = [playerCount][]Card{}
	g.deck = nil
	g.trump = ""
	g.trumpSet = false
	g.trick = Trick{}
	g.tricksPlayed = 0
	g.trickHistory = nil
	g.tricksA, g.tricksB = 0, 0
	g.turn = NoSeat
}
