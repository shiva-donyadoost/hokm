package game

import "errors"

// Domain errors returned by engine commands. The transport layer maps these
// to API/WebSocket error codes; it must never leak internal state details.
var (
	ErrWrongPhase         = errors.New("game: command not allowed in current phase")
	ErrNotHakem           = errors.New("game: only the hakem may select trump")
	ErrInvalidTrump       = errors.New("game: invalid trump suit")
	ErrNotYourTurn        = errors.New("game: not this player's turn")
	ErrCardNotOwned       = errors.New("game: player does not hold this card")
	ErrMustFollowSuit     = errors.New("game: must follow the lead suit")
	ErrTrickNotFull       = errors.New("game: trick is not complete")
	ErrRoundNotComplete   = errors.New("game: round tricks are not finished")
	ErrGameNotComplete    = errors.New("game: match is not decided yet")
	ErrGameAlreadyOver    = errors.New("game: match is already complete")
	ErrDuplicatePlayer    = errors.New("game: duplicate player id")
	ErrInvalidPlayerCount = errors.New("game: exactly four players are required")
)
