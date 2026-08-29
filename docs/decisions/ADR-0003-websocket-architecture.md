# ADR-0003: WebSocket Architecture

Date: 2026-08-29
Status: Accepted

## Context

Gameplay is turn-based but latency-sensitive (players expect sub-second
updates). Rooms have exactly 4 participants plus optional spectators.
We need per-room broadcast, per-player private events (their hand), and
reconnection.

## Decision

- **One WS endpoint** `/api/ws` (upgraded after REST auth), not per-room
  endpoints.
- **Envelope protocol** (JSON):
  ```json
  { "type": "PLAY_CARD", "id": "client-msg-id", "payload": { ... } }
  { "type": "EVENT", "id": "server-msg-id", "name": "TrickCompleted", "room": "r_k3x", "payload": { ... } }
  { "type": "ERROR", "id": "client-msg-id", "code": "not_your_turn", "message": "..." }
  ```
- **Hub → Room → Session** model: each connection is a `Session` (auth'd
  user id); rooms register sessions; hub dispatches commands to room
  actors. Room processing is serialized through a per-room goroutine
  (actor pattern) — no locks inside room logic.
- **Heartbeat**: server pings every 20s; session dropped after 2 missed
  pongs. Client sends `PING` command as keepalive fallback.
- **Reconnection**: on reconnect with valid token + active room, server
  replays: full room snapshot + last game state (private view).
- Writes to a WS connection are marshalled through a per-session buffered
  channel to avoid concurrent write races.

## Alternatives considered

- **Socket.IO**: adds a protocol/runtime dependency; native WS + our own
  envelope is smaller, typed, and easier to load-test in Go.
- **SSE + REST**: no client→server push path; card plays need low-latency
  bidirectional messaging.
- **Per-room endpoints**: complicates auth and connection migration when
  moving between lobby and rooms.

## Consequences

- TS client mirrors the envelope types; a single `protocol.ts` is the
  contract (kept in sync manually, verified by E2E tests).
- Message ordering per room is guaranteed by the room actor.
