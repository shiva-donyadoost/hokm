# ADR-0002: Game State Management (Command-Driven State Machine)

Date: 2026-08-29
Status: Accepted

## Context

Card game state is highly concurrency-sensitive and rule-sensitive. Bugs
like double-played cards, skipped turns, or illegal trumps are fatal to
trust. The engine must be independently testable and deterministic where
possible.

## Decision

The engine is a finite state machine mutated **only** through explicit
commands:

`StartGame, SelectHakem, DealInitialCards, SelectTrump, DealRemainingCards,
PlayCard, CompleteTrick, CompleteRound, CompleteGame`

- Each command returns a domain error (`ErrNotYourTurn`, `ErrCardNotOwned`,
  `ErrFollowSuit`, `ErrWrongPhase`, ...) or produces `Event`s.
- State machine phases: `AwaitingHakem → TrumpSelection → Dealing →
  TrickPlay → RoundComplete → GameComplete`.
- Randomness (shuffle, first-round hakem seat) is injected via an `rand.Rand`
  interface so tests can seed determinism.
- The engine holds a `sync.Mutex` per game; commands are serialized.
- Events emitted by commands are the *only* data exposed to transport; the
  serializer explicitly omits hidden information (opponent hands, deck).

## Alternatives considered

- **Free-for-all mutable state**: rejected — impossible to test or reason
  about legality.
- **Event sourcing within engine**: rejected for now — full event sourcing
  adds replay infrastructure we do not need at current scale; commands+events
  give us the audit trail we need. Persisted game logs (Phase 11) record
  events for history/replay anyway.

## Consequences

- All legality lives in one place; the HTTP/WS layer performs identity and
  membership checks, then delegates to the engine.
- Engine tests can drive entire games deterministically with seeded decks.
