# Database

## Engines

- **PostgreSQL** — system of record: users, refresh tokens, rooms registry,
  games, game players, statistics, ratings, chat messages.
- **Redis** — ephemeral: fixed-window rate limits, presence. Fail-open by
  design (a Redis outage logs a warning and disables limiting).

Live in-game state is **in process memory** (authoritative, latency-bound);
PostgreSQL records durable outcomes (ADR-0006).

## Migrations

Plain SQL in `backend/migrations/`, embedded into the binary
(`migrations.FS`) and applied at startup by `internal/infra/postgres.Migrate`
inside per-file transactions, tracked in `schema_migrations`. Idempotent:
re-running is a no-op.

| Version | Contents |
|---|---|
| 0001_init | users, refresh_tokens, rooms, games, game_players, game_events, statistics |
| 0002_chat | chat_messages |
| 0003_rating | statistics.rating (default 1000) |

## Key tables

- `users` — text PK (app-generated crypto IDs), unique username/email,
  bcrypt password hash, guest flag.
- `refresh_tokens` — SHA-256 token hash PK, user FK, expiry; **single use**
  (consumed = deleted inside a transaction with `FOR UPDATE`).
- `games` / `game_players` — one row per recorded match + per-seat
  participants (AI flagged with difficulty).
- `statistics` — per-user games/wins/losses/rounds and Elo rating.
- `chat_messages` — durable chat log (the live room history is in memory).

## Local access

The dev compose exposes `localhost:5432` (hokm/change-me-dev-only/hokm).
Integration tests opt in with:

```bash
HOKM_TEST_PG_DSN="postgres://hokm:change-me-dev-only@localhost:5432/hokm?sslmode=disable" \
  go test ./internal/infra/postgres/ -count=1
```
