# ADR-0013: Session Restore, Lobby Teams, Mobile Hand, Collect

Date: 2026-09-01
Status: Accepted

Supersedes in part:

- ADR-0012 item 3 (mobile fan arc) — mobile hand is an overlapping row,
  not a rotated arc.
- ADR-0012 item 7 (seat label = team rounds won) — seat label is tricks
  this round.

## Context

Seven follow-ups after Wave 4: refresh dumps the user on login, the
mobile fan fights drag, the lobby cannot rearrange seats, seat scores
show the wrong counter, trick collect jumps from above the felt, mobile
drag does not fire, and the host cannot delete a lobby room.

## Decision

1. **Refresh**: never log the user out while a refresh token exists.
   Boot restores `/me` (refreshing the access JWT first if it is expired)
   before protected routes render. The current URL (lobby or table)
   stays. Logout is only the explicit button.
2. **Mobile hand**: no rotation. Cards sit in one overlapping row
   (negative margin), same suit/rank order as desktop, so a finger can
   drag a legal card toward the table.
3. **Lobby teams**: two columns (Team A seats 0+2, Team B seats 1+3).
   The host may move any occupant (human or AI) to any seat, including
   swaps, on both viewports. Lobby only; in-game rejected.
4. **Seat score**: the number next to each name is that team's tricks
   this round. The top bar still shows match rounds.
5. **Collect**: when the fourth card lands, the four cards travel from
   their felt slots toward the winner. They must not remount from the
   off-table origin (that was the jump/glitch). The existing pause
   before collect stays.
6. **Mobile drag**: same as desktop — pointer down on a legal card,
   drag toward the table, release to play. Tap-to-select / tap-to-play
   remains. `touch-action: none` and capture on the wrapper, not the
   inner `<button>`.
7. **Delete room**: host-only, lobby-only, with a confirm. Members are
   notified (`status: closed`) and the room is removed.

## Alternatives considered

- **Keep the mobile arc and only raise overlap**: rejected — product
  wants a row for drag.
- **HTML5 drag-and-drop for seats**: rejected on mobile; pointer drag
  plus tap-source / tap-target.
- **Any member may move themselves**: rejected — only the host reorders
  (original request: room creator).
- **Delete during a match**: rejected — lobby only.

## Consequences

- `POST /api/rooms/{id}/seats` and `DELETE /api/rooms/{id}` are new.
- Access tokens issued under the 30-day TTL still need a boot-time
  refresh if the tab was backgrounded past expiry.
- Scripted tests for seat moves and lobby delete live next to existing
  room tests.
