package game

import "strings"

// Phase is the engine's finite state machine state.
type Phase string

const (
	// PhaseAwaitingHakem: the match is set up; the hakem-selection deal
	// (card-by-card until an Ace appears) has not run yet.
	PhaseAwaitingHakem Phase = "awaiting_hakem"
	// PhaseHakemSelection: hakem selection is in progress.
	PhaseHakemSelection Phase = "hakem_selection"
	// PhaseTrumpSelection: initial cards are dealt; hakem must pick trump.
	PhaseTrumpSelection Phase = "trump_selection"
	// PhaseTrickPlay: all cards dealt; tricks are being played.
	PhaseTrickPlay Phase = "trick_play"
	// PhaseRoundComplete: a round's 13 tricks finished; scores recorded.
	PhaseRoundComplete Phase = "round_complete"
	// PhaseGameComplete: a team won the match.
	PhaseGameComplete Phase = "game_complete"
)

func (p Phase) Valid() bool {
	switch p {
	case PhaseAwaitingHakem, PhaseHakemSelection, PhaseTrumpSelection,
		PhaseTrickPlay, PhaseRoundComplete, PhaseGameComplete:
		return true
	}
	return false
}

func (p Phase) String() string { return strings.TrimSpace(string(p)) }
