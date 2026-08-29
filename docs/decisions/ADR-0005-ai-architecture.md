# ADR-0005: AI Architecture

Date: 2026-08-29
Status: Accepted

## Context

Hokm has hidden information (hands). AI must be competitive at five
difficulty levels and must never cheat. Future RL training is planned.

## Decision

- `internal/ai` defines a strategy interface decoupled from the engine:

  ```go
  type PlayerStrategy interface {
      DecideTrump(is InformationSet) game.Suit
      DecideCard(is InformationSet) game.Card
  }
  ```

- **InformationSet** is the *only* input: own hand, trump, lead suit,
  played-card history, tricks won, seat/partner, and derived probabilities
  (remaining unseen cards distributed over unknown hands, weighted by
  follow-suit constraints). The engine package exposes a projection
  function `NewInformationSet(view)` that provably contains no hidden data
  (test: projection of a view never includes other hands).
- Ladder of implementations:
  - **Easy**: random-legal.
  - **Medium**: rule-based (follow suit, dump lowest, win cheaply).
  - **Hard**: heuristics + card counting (track voids, high remaining).
  - **Expert**: heuristic + Monte Carlo rollouts over plausible hands.
  - **Pro**: Monte Carlo with partner-signaling heuristics and trump
    management; pluggable evaluator so an RL policy can replace it later.
- Strategies are stateless per decision; per-game memory (counted cards)
  lives in a `CardTracker` the AI owns, fed only from public events.
- The simulator (`backend/cmd/simulator`) runs N games AI-vs-AI, asserting
  termination, legality, uniqueness of cards, and score consistency.

## Alternatives considered

- **AI reading engine internals**: rejected — H5 violation; the Information
  Set boundary is enforced by type (projection takes a public view).
- **Single configurable AI**: rejected — distinct strategies are easier to
  test, benchmark, and later replace with learned policies.

## Consequences

- Same interface supports humans-optional AI takeover (Phase 14).
- RL integration later = new `RLStrategy` implementing the same interface.
