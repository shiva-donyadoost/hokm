# Game Rules (implemented)

Standard Iranian Hokm, 4 players, fixed 2v2 teams. Seats 0–3 alternate
teams: seats 0+2 = Team A, seats 1+3 = Team B. Partners sit opposite.

## Hakem selection

- A full deck is shuffled; starting **left of the dealer**, one card is
  turned to each player in turn order.
- The **first player to receive an Ace** becomes Hakem; the deciding card
  is part of the public `hakem_selected` event.
- The drawn cards are folded back before the real deal.

## Dealing

1. **Initial deal**: 5 cards per player, card-by-card starting left of the
   Hakem.
2. The Hakem sees their 5 cards and selects the **trump** suit.
3. **Remaining deal**: the last 8 cards per player are dealt the same way
   (5 + 8 = 13 cards each).

## Play

- The Hakem leads the first trick; the winner of each trick leads the next.
- 13 tricks per round.
- **Follow suit**: a player holding a card of the lead suit must play one.
  Otherwise any card (including trump) may be played.
- **Winning a trick**: the highest trump played wins; if no trump was
  played, the highest card of the lead suit wins.

## Scoring

- **Round**: a team wins the round by taking **7 or more** of the 13
  tricks.
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
