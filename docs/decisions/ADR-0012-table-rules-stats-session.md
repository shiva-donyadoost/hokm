# ADR-0012: Table Rules, Hand UX, Stats, Session, Chat

Date: 2026-09-01
Status: Accepted

Supersedes in part:

- ADR-0002 (hakem-by-ace draw) — first hakem is now random.
- ADR-0008 (15-minute access JWT) — access TTL default is 30 days.
- ADR-0010 (illegal cards not faded; leaderboard UI only) — legal cards
  are highlighted by dimming the rest; leaderboard rank is by wins.

ADR-0011 is the Windows host-port remap and is unrelated.

## Context

Twelve play-feel and rule gaps after Wave 3 (ADR-0010). Product answers
below are from a clarifying round; this ADR is the source of truth.

## Decision

### 1. Hand order (client)

Group by suit, ranks Ace → King → Queen → Jack → 10 → 9 … → 2 inside
each suit.

Suit order **alternates color**, trump first:

- Color groups: red = hearts, diamonds; black = spades, clubs.
- Within a color, fixed order: hearts then diamonds; spades then clubs.
- Position 1: trump.
- Position 2: first suit of the **opposite** color (that color's fixed
  order).
- Position 3: the remaining suit of trump's color.
- Position 4: the last opposite-color suit.

Example, trump = hearts: hearts, spades, diamonds, clubs.

### 2. Legal-card highlight (client)

Legal cards stay at full brightness. Illegal cards are dimmed. No extra
ring or glow.

### 3. Mobile layout (client)

Keep opponent names and scores. Hide opponent hand-backs (they overflow
the viewport). Tighten the local-player arc so it stays inside the
screen. Only the local player's hand is shown on small viewports.

### 4. Leaderboard and profile stats

Rank `GET /api/leaderboard` by **wins** (then rating, then username).
Wins and losses must update after every finished match. Investigate and
fix the production deploy (`http://37.32.9.231`) where both surfaces
show zeros, then land the same fix locally. Root cause already identified
in the working tree: `games.room_id` FK pointed at an unused `rooms`
table, so `ApplyMatch` rolled back (migration `0004_games_room_fk`).

### 5. Round end (engine)

A team that reaches **7 tricks** wins the round immediately. Remaining
tricks of that round are not played. Leftover hands are discarded on
`CompleteRound`. Match scoring is unchanged (first team to the configured
round count wins).

### 6. Illegal cards (client + server)

Illegal cards are not clickable or draggable. No error toast. The server
follow-suit check stays (H3 / ADR-0004); a cheating client still gets a
protocol error, not a UI message.

### 7. Seat label

Show that player's **team rounds won** (match score, not tricks this
round) next to the name.

### 8. Back to rooms

The post-match **Back to rooms** button is shown only when all four seats
are still occupied. If anyone has left, hide it. Host `REPLAY_GAME`
(ADR-0010) is unchanged.

### 9. Session length

Access JWT default TTL is **30 days** (`JWT_ACCESS_TTL=720h`), matching
the refresh token. Users stay signed in for a month without logging in
again. Override remains env-driven.

### 10. Hakem

- **Round 1**: uniform random among the four seats (human and AI).
- **Later rounds**: not re-randomized. Hakem stays if their team won the
  previous round; if the hakem's team lost, hakemship passes to the next
  seat (`NextSeat`). This is the existing engine rotation
  (`KeepHakemOnLoss` default false).

### 11. Turn progress bar

Every acting seat — human and AI, trump and card — shows the fill-up bar
from the server `deadline_unix_ms`. AI still plays on the step timer; the
deadline is display-only for AI so the bar always has a server-authored
end time (Wave 2: no hard-coded gameplay values on the client).

### 12. Chat

Only text a player types. `ChatService.System` is a no-op: no timeout,
join/leave, auto-play, or AI-takeover lines.

## Alternatives considered

- **Fixed suit order (hearts, diamonds, clubs, spades)**: rejected —
  product wants trump-led, alternating opposite color.
- **Highlight as a ring/glow**: rejected — legal = bright, illegal = dim.
- **Hide opponent names on mobile**: rejected — names and scores stay;
  only hand-backs go.
- **Rank by Elo/rating**: rejected — product wants wins.
- **Play all 13 tricks**: rejected — first to 7 already decided the round.
- **Click illegal card, show error**: rejected — no pointer, no toast.
- **Show tricks this round (3 of 13) by the name**: rejected — show team
  rounds won for the match.
- **Always show Back to rooms**: rejected — hide when a seat has left.
- **Keep 15-minute access JWT**: rejected — one-month session.
- **Ace-draw first hakem / random every round / humans-only draw**:
  rejected — first round random among all four seats; later rounds use
  standard keep-on-win / pass-on-loss.
- **Turn bar only for the local human**: rejected — every acting seat,
  including AI.
- **Keep system chat lines**: rejected — player text only.

## Consequences

- Engine tests that assumed ace-draw hakem or exactly 13 tricks per round
  must follow the new rules.
- Existing Postgres deploys need migration `0004` before stats write.
- Access tokens issued before this change still expire at the old TTL
  until the user logs in again.
- Hand sort and dimming are presentation-only; legality remains on the
  server.
- `CHAT` payloads may still carry `is_system`; the server simply never
  emits those rows after this change.
