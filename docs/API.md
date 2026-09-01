# REST API

Base path `/api`. JSON bodies. Errors use a stable envelope:

```json
{ "code": "not_your_turn", "message": "it is not your turn" }
```

Authentication: `Authorization: Bearer <access_token>` (JWT, **30 days**
default / `JWT_ACCESS_TTL=720h`, ADR-0012). On `401 token_expired` the
client should POST `/api/auth/refresh`.

## Health & ops

| Method | Path | Description |
|---|---|---|
| GET | `/api/health` | liveness (used by Docker healthchecks) |
| GET | `/api/metrics` | Prometheus text metrics |

## Auth

| Method | Path | Body | Notes |
|---|---|---|---|
| POST | `/api/auth/register` | `{username, email, password}` | 201 → `{user, tokens}`; username 3–24 `[a-zA-Z0-9_]`, password 8–128 |
| POST | `/api/auth/login` | `{username, password}` | 200 → `{user, tokens}`; identical 401 for unknown user / wrong password |
| POST | `/api/auth/refresh` | `{refresh_token}` | rotates; old token single-use |
| GET | `/api/me` | — | current profile |

## Rooms

| Method | Path | Body | Notes |
|---|---|---|---|
| POST | `/api/rooms` | `{name, visibility}` | host = caller, seat 0, **host ready by default** |
| GET | `/api/rooms` | — | public rooms in lobby status |
| GET | `/api/rooms/{id}` | — | room snapshot |
| POST | `/api/rooms/join` | `{code}` | join by 6-char code |
| POST | `/api/rooms/{id}/leave` | — | host transfer on leave; empty room deleted |
| POST | `/api/rooms/{id}/ready` | `{ready}` | toggle own readiness |
| POST | `/api/rooms/{id}/kick` | `{user_id}` | host only, humans only |
| POST | `/api/rooms/{id}/ai` | `{difficulty}` | host only; easy/medium/hard/expert/pro; AI auto-ready |
| POST | `/api/rooms/{id}/ai/fill` | — | host only; fill every empty seat with a random-difficulty AI |
| POST | `/api/rooms/{id}/ai/remove` | `{user_id}` | host only |
| POST | `/api/rooms/{id}/seats` | `{from_seat, to_seat}` | host, lobby only; swap or move onto an empty seat (0–3) |
| DELETE | `/api/rooms/{id}` | — | host, lobby only; notifies `status: closed` then removes the room |

## Stats

| Method | Path | Description |
|---|---|---|
| GET | `/api/leaderboard` | top 50 by wins, then rating (ADR-0012) |
| GET | `/api/stats` | caller's rating/games/wins/losses/rounds |

## Domain error codes

`unauthorized, token_expired, bad_credentials, username_taken, email_taken,
validation, room_not_found, room_full, already_in_room, not_in_room,
not_host, game_in_progress, not_your_turn, must_follow_suit,
card_not_owned, trick_not_full, wrong_phase, not_hakem, invalid_trump,
rate_limited, internal`
