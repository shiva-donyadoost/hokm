# ADR-0014: Client diagnostics and hand playability lock

- Status: Accepted
- Date: 2026-09-01
- Deciders: engineering

## Context

Real players intermittently could not play a card even though the UI showed
it was their turn ("your turn" badge / amber seat), and the card would not
visually move under drag. Diagnosis required client-side evidence that the
server logs cannot provide (pointer handlers, UI lock flags, WS readyState,
client-side legal-move filter).

Two concrete defects were identified in code:

1. **Transform ownership clash (primary "card cannot move")**: each hand
   card wrapper used the same DOM node for (a) the CSS `deal-in` animation
   (`animation-fill-mode: both`, animating `transform`), (b) the fan/select
   inline `transform`, and (c) imperative drag `transform` updates. After
   the deal animation, the animation fill keeps owning `transform` in the
   cascade, so drag updates appear to do nothing and the fan layout fights
   the animation. Symptom matches "UI shows turn but card cannot move".

2. **Non-playable overlapping cards swallow pointers**: illegal (dimmed)
   cards kept default `pointer-events` while omitting handlers, so they
   still hit-tested above partially covered legal cards in the overlapping
   hand and silently dropped taps/drags.

Secondary gaps that amplify "stuck turn" reports:

- `playCard` / `selectTrump` silently no-op when the WebSocket is not OPEN
  while the last `SeatView` still shows the local turn (Room reconnect
  effect was a stub).
- `broadcast` pushed `deadline_*` **before** `rescheduleTimerLocked`, so
  clients often rendered a stale countdown after turn changes.

## Decision

1. Split hand card DOM: outer node owns the deal animation; inner node owns
   fan/select/drag transforms and pointer handlers. Deal keyframes end with
   an explicit `transform: none` on the outer node only.
2. Set `pointer-events: none` on non-playable hand wrappers so covered legal
   cards remain reachable.
3. Add a small structured client diagnostic logger (`clientLog`): uncaught
   errors, unhandled rejections, play-card attempts with reason, turn /
   playability snapshots, and high-signal game/WS events. Ring buffer +
   `window.__HOKM_DIAG__` dump for support; console JSON lines, no secrets.
4. Surface WS-not-open play failures via `lastError`, and auto-reconnect the
   room subscription with bounded backoff after unexpected close.
5. Reschedule table deadlines **before** sending `STATE` so countdown fields
   match the turn the client just received.

## Alternatives considered

- **Opacity-only deal animation**: simpler, but drops the fly-in motion that
   the UX spec calls for; rejected in favor of a two-node split.
- **Remove client legal-move filtering**: would let illegal taps round-trip
   to the server; rejected (extra ERROR noise, worse UX). Keep filter; make
   it observable via diagnostics.
- **Server-only logging**: cannot see pointer/transform/WS readyState on the
   browser; rejected as insufficient for this class of bug.

## Consequences

- Hand markup is one wrapper deeper; CSS deal animation must not share a
  node with drag transforms (lesson recorded in agents.md).
- Diagnostic buffer is in-memory only (cleared on reload); not a telemetry
  backend. Do not log tokens or full chat bodies.
- Reconnect may briefly duplicate SUBSCRIBE; server already re-sends ROOM /
  STATE on subscribe (existing behavior).
- Deadline fix slightly changes STATE ordering relative to timer arming
  (timers arm before the push); clients see consistent countdowns.
