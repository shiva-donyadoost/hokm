# AI architecture

The AI module (`backend/internal/ai`) is decoupled from transport and never
accesses hidden information (ADR-0005, HARD RULE H5).

## Fairness boundary

Strategies receive only an **InformationSet**, constructed by
`BuildInformationSet(view, publicEvents)` from:

- the seat's own hand (`SeatView.YourHand`),
- trump, lead suit, current trick,
- per-seat hand counts, tricks won,
- all publicly played cards and completed-trick history.

A dedicated test (`TestInformationSetFairness`) proves that no card still
held by another seat can appear in the projection.

## Strategy ladder

| Difficulty | Implementation |
|---|---|
| Easy | uniformly random legal card; arbitrary trump |
| Medium | rule-based: most-trumps trump; cheap winners; dump low behind a winning partner |
| Hard | + void inference from trick history; risk-aware aces; avoids over-trumping low tricks |
| Expert | Monte Carlo over plausible hands (uniform unseen-card deal respecting voids), random-legal rollouts, ~40 samples/decision |
| Pro | same with ~120 samples and partner-aware baseline heuristics |

`LegalCards(is)` mirrors the engine's legality rules (follow suit), so AI
decisions are engine-legal by construction; any mismatch is a hard test
failure.

## Integration

- Room AI seats are stored with `ai_difficulty`; the table manager builds
  one strategy per AI seat at match start.
- After every human action the table loops AI decisions until a human turn
  or match end, timing each decision into `hokm_ai_decisions_total`.
- Disconnected humans are played by a `medium` fallback strategy after the
  grace period (Phase 14).

## Simulation

`backend/cmd/simulator` runs batches of AI-vs-AI matches:

```bash
go run ./cmd/simulator -games 1000 -rounds 7 -seed 1234 -strategy expert
```

It aborts on the first illegal move or non-terminating game and prints a
report (rounds, tricks, decisions, team balance). Validation runs:
1000 expert games → 139,464 tricks, 568,584 decisions, 0 illegal moves,
team wins 504/496.

## Future RL

An RL policy only needs to implement `PlayerStrategy` (or be wrapped in one
that maps policy outputs to legal actions from the same InformationSet) —
no engine or transport changes required.
