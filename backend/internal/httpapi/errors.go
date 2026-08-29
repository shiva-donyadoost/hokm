// Package httpapi provides the REST transport layer: routing, middleware,
// and typed error mapping. It delegates all business rules to the
// application layer and never mutates game state directly.
package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/hokm/platform/internal/game"
)

// APIError is the transport-facing error with a stable machine code.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

func apiError(status int, code, msg string) *APIError {
	return &APIError{Status: status, Code: code, Message: msg}
}

// mapDomainError converts engine errors to client-safe API errors.
// Messages never leak hidden information.
func mapDomainError(err error) *APIError {
	switch {
	case errors.Is(err, game.ErrNotYourTurn):
		return apiError(http.StatusConflict, "not_your_turn", "it is not your turn")
	case errors.Is(err, game.ErrMustFollowSuit):
		return apiError(http.StatusUnprocessableEntity, "must_follow_suit", "you must follow the lead suit")
	case errors.Is(err, game.ErrCardNotOwned):
		return apiError(http.StatusUnprocessableEntity, "card_not_owned", "you do not hold that card")
	case errors.Is(err, game.ErrTrickNotFull):
		return apiError(http.StatusConflict, "trick_not_full", "trick is not complete")
	case errors.Is(err, game.ErrWrongPhase):
		return apiError(http.StatusConflict, "wrong_phase", "command not allowed right now")
	case errors.Is(err, game.ErrNotHakem):
		return apiError(http.StatusForbidden, "not_hakem", "only the hakem may select trump")
	case errors.Is(err, game.ErrInvalidTrump):
		return apiError(http.StatusUnprocessableEntity, "invalid_trump", "invalid trump suit")
	case errors.Is(err, game.ErrDuplicatePlayer):
		return apiError(http.StatusBadRequest, "duplicate_player", "duplicate player")
	case errors.Is(err, game.ErrInvalidPlayerCount):
		return apiError(http.StatusBadRequest, "invalid_players", "exactly four players required")
	default:
		return apiError(http.StatusInternalServerError, "internal", "internal error")
	}
}

// writeJSON writes v as JSON with the given status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError maps err to an APIError and writes it.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var ae *APIError
	if !errors.As(err, &ae) {
		ae = mapDomainError(err)
	}
	if ae.Status >= 500 {
		slog.ErrorContext(r.Context(), "request error", "err", err, "path", r.URL.Path)
	}
	writeJSON(w, ae.Status, ae)
}
