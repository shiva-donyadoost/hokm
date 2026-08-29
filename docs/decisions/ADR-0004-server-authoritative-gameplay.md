# ADR-0004: Server-Authoritative Gameplay

Date: 2026-08-29
Status: Accepted

## Context

The browser is untrusted. Cheating (peeking hands, playing out of turn,
forging scores) would destroy the product. Clients also lag or desync.

## Decision

- The Go backend owns **all** game state. Clients render; they never
  compute.
- Every command passes five checks before touching the engine:
  1. **Authentication** — valid session (JWT).
  2. **Membership** — sender is a seated participant of the room's game.
  3. **Phase** — command legal for current engine phase.
  4. **Turn** — command issued by the player whose turn it is.
  5. **Legality** — card ownership + follow-suit etc., enforced by engine.
- **Views, not state**: WS events carry a *view* of state per recipient:
  the sender's own hand in full; others reduced to counts; deck never sent;
  only completed tricks are public.
- Any hidden-information leak (e.g., echoing a full state object) is a
  security bug; a unit test asserts private-view serialization for every
  event type.
- Server timestamps + monotonic message ids order events; clients never
  infer "who won" locally except as an optimistic hint that the server
  event corrects.

## Alternatives considered

- **Client authority with server spot-checks**: trivially cheatable.
- **Deterministic lockstep from a shared seed**: exposes the deck seed to
  all clients; rejected.

## Consequences

- Slightly more server CPU per action (negligible: validation is O(1)).
- Client code is simpler; one source of truth for disputes.
