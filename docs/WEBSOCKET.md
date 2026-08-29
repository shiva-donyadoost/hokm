# WebSocket protocol

Single endpoint: `GET /api/ws?token=<access_token>` (authenticated at
upgrade; session bound to the user for its lifetime).

## Envelope

```json
{ "type": "PLAY_CARD", "id": "client-msg-id", "payload": { ... } }
{ "type": "STATE", "payload": { ... SeatView ... } }
{ "type": "EVENTS", "name": "trick_completed", "payload": { ... } }
{ "type": "ERROR", "id": "client-msg-id", "payload": { "code": "...", "message": "..." } }
```

## Client → server commands

| Type | Payload | Notes |
|---|---|---|
| `PING` | — | keepalive; server answers `PONG` |
| `SUBSCRIBE` | `{room_id}` | membership required; sends ROOM snapshot, chat history, and live STATE if a match runs; rebinds reconnects |
| `START_GAME` | `{room_id}` | host only; 4 seated members, all ready |
| `SELECT_TRUMP` | `{room_id, suit}` | hakem only, during trump phase |
| `PLAY_CARD` | `{room_id, card:{suit, rank}}` | seat's turn; ownership + follow-suit enforced |
| `CHAT` | `{room_id, body}` | member only; 1–500 chars, 5 per 10 s |

## Server → client messages

| Type | Payload |
|---|---|
| `STATE` | full per-seat `SeatView` (own hand in full, others as counts) |
| `EVENTS` | one public event per message (`name`): `hakem_selected`, `trump_selected`, `card_played`, `trick_completed`, `round_completed`, `game_completed`, `next_round_started` |
| `ROOM` | lobby snapshot after any room mutation |
| `CHAT` | `{id, room_id, user_id, username, body, is_system, at}` |
| `ERROR` | rejected command reason |
| `PONG` | heartbeat reply |

## Ordering, heartbeat, reconnection

- Per-room game updates are applied under the table lock; each client's
  messages arrive in server-apply order.
- Server pings every 20 s; a session is dropped after ~45 s without pong.
- On reconnect: open a new socket with a valid token and `SUBSCRIBE` the
  room again — the server replays the room + chat + current game view and
  rebinds the seat. After `TURN_TIMEOUT_SECONDS` (default 60) the fallback
  AI may act for the absent player.
