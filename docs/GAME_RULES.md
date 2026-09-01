# Game Rules (implemented)

Standard Iranian Hokm, 4 players, fixed 2v2 teams. Seats 0–3 alternate
teams: seats 0+2 = Team A, seats 1+3 = Team B. Partners sit opposite.

## Hakem selection

- **Round 1**: the engine picks a hakem uniformly at random among the
  four seats (human or AI). The public `hakem_selected` event names that
  seat. There is no ace-draw (ADR-0012).
- **Later rounds**: hakem stays if their team won the previous round;
  if the hakem's team lost, hakemship passes to the next seat
  (`NextSeat`). See Hakem rotation below.

## Dealing

1. **Initial deal**: 5 cards per player, card-by-card starting left of the
   Hakem.
2. The Hakem sees their 5 cards and selects the **trump** suit.
3. **Remaining deal**: the last 8 cards per player are dealt the same way
   (5 + 8 = 13 cards each).

## Play

- The Hakem leads the first trick; the winner of each trick leads the next.
- Up to 13 tricks per round. The round ends as soon as one team has
  **7 tricks**; leftover cards are not played (ADR-0012).
- **Follow suit**: a player holding a card of the lead suit must play one.
  Otherwise any card (including trump) may be played.
- **Winning a trick**: the highest trump played wins; if no trump was
  played, the highest card of the lead suit wins.

## Scoring

- **Round**: a team wins the round by taking **7 tricks** (play stops
  at that point; remaining tricks are skipped).
- **Match**: the first team to win the configured number of rounds
  (`ROUNDS_TO_WIN`, default 7) wins the match.

## Hakem rotation (configurable, documented ambiguity)

Regional rules vary. The engine default: the Hakem **keeps the role while
their team wins rounds**; when the Hakem's team loses a round, hakemship
passes to the next seat in turn order (`NextSeat`). Set
`KeepHakemOnLoss: true` in `game.Options` for a fixed Hakem across the
whole match (the engine-level option is exposed; the server uses the
default).

## Determinism

All shuffles/draws go through an injected `rand.Rand`. Tests and the
simulator pass seeded sources; a `DeckShuffler` option can fix exact deck
orders for scripted scenarios.
