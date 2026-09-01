# ADR-0010: Gameplay UX, Replay, Statistics Surface, Turn Timer

Date: 2026-08-30
Status: Accepted

## Context

The lobby and table already work end-to-end (phases 5-13, Wave 2), but
play-feel and post-match surfaces lag behind:

1. The room creator must click Ready even though they intend to play.
2. Filling three empty seats requires three Add-AI clicks with a chosen
   difficulty.
3. Hakem trump is a four-button overlay, not a choice from the dealt hand.
4. The hand fan leaves gaps; overlapping cards were faded with opacity.
5. Trump is shown as English text, not a suit mark.
6. After a match the room stays `in_game` with no way to play again.
7. Played cards flash onto the table; trick-win uses a yellow highlight
   box instead of cards travelling to the winner.
8. Statistics and Elo already persist (`rating.ScoreStore`, `/api/stats`,
   `/api/leaderboard`) but the profile/leaderboard UI never reads them.
9. The acting player's deadline is a thin bar only on *your* seat, using
   a hard-coded 10s width.

Server authority (ADR-0004) and engine purity (ADR-0002) still apply:
presentation may animate, but legality, membership, and scoring stay on
the server.

## Decision

### Lobby

- Room **creator is seated ready**. Other humans still ready themselves.
  AI seats remain auto-ready.
- Host-only `POST /api/rooms/{id}/ai/fill` fills every empty seat with an
  AI whose difficulty is chosen uniformly at random from the existing
  set (easy/medium/hard/expert/pro). The per-seat Add-AI control stays.

### Trump

- No suit buttons. Hakem taps one of their five cards; the **suit of that
  card** is sent as `SELECT_TRUMP`. Server validation is unchanged
  (hakem, phase, legal suit).

### Presentation (client-only)

- Hand fan: overlapping cards, rotation around the bottom edge, **no
  opacity** on stacked or illegal cards. Illegal cards are simply not
  submitted (pointer disabled), not faded.
- Trump indicator: suit glyph (ASCII unicode escapes in source, per
  lesson 9) instead of the word "hearts".
- Played card: animates from that seat's screen position to the trick
  slot. Direction is a function of relative seat (bottom/left/top/right).
- Trick win: the four cards travel as a group toward the winner's seat.
  The yellow reveal box is removed; the motion is the reveal.
- Turn timer: every acting seat (you and opponents, trump or card)
  shows a progress bar that **fills** toward the server deadline. Width
  is derived from the first-seen remaining ms for that deadline, never a
  hard-coded gameplay timeout.

### Replay

- New WS command `REPLAY_GAME {room_id}`. Host only, and only when the
  table's engine is `PhaseGameComplete`.
- Same frozen membership, same room id; a **new** `game.Game` is dealt.
  Timers/step gens are bumped so stale fires are no-ops. Stats for the
  finished match have already been recorded; the new match records
  separately.
- Room status stays `in_game`. Clients keep the GameTable and receive a
  fresh `STATE`.

### Statistics

- No new persistence. Profile reads `GET /api/stats`; a Leaderboard page
  reads `GET /api/leaderboard`. Only human seats are scored (existing
  `ApplyMatch` skip-AI rule).

## Alternatives considered

- **Auto-start when 4 ready**: rejected — host still owns kick/AI/fill
  and must confirm the table.
- **Replay returns to lobby**: rejected — same four players want another
  hand without re-readying; going lobby would fight H3 (status flicker
  and re-subscribe races).
- **Client-side random AI fill (three Add-AI calls)**: rejected — not
  atomic; two hosts/tabs could overfill. One host-only command under the
  room lock.
- **Keep yellow winner box plus motion**: rejected — the box fights the
  collect animation and was the reported clutter.
- **Hard-code 5/10/15s on the client progress bar**: rejected — Wave 2
  rule (no hard-coded gameplay values). Deadline unix ms from the server
  plus first-seen remaining is presentation-only.

## Consequences

- Hosts can sit down and fill a bot table in two clicks (Fill AI + Start).
- Replay is a gameplay command, so it is tested at the WS layer like
  `START_GAME`, not as a page reload.
- Animation timings stay in `frontend/src/config.ts` (presentation) and
  must not affect engine legality.
- Profile/leaderboard stay empty until at least one human-vs-* match
  has been recorded (in-memory store resets on process restart unless
  Postgres is up).
