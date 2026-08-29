# ADR-0006: PostgreSQL + Redis Strategy

Date: 2026-08-29
Status: Accepted

## Context

We need durable identity/history/statistics and low-latency live state for
running games. Two very different workloads.

## Decision

**PostgreSQL** — system of record:

- users, profiles, rooms (registry), games, game_players, results,
  statistics, ratings, chat logs, game event logs.
- Migrations via `golang-migrate`, versioned SQL files in
  `backend/migrations/`, applied by compose `backend` entrypoint and CI.
- Access through `database/sql` + `pgx` driver; repository interfaces live
  in `internal/app`, implementations in `internal/infra/postgres`.

**Redis** — ephemeral + coordination:

- active room index (room code → server instance), presence (WS session
  heartbeats), refresh-token denylist, rate-limit counters, pub/sub channel
  for cross-instance room events (future horizontal scale).
- Live in-game state stays **in process memory** (authoritative); Redis
  snapshots allow recovery after restart (best-effort, Phase 14).

**Why not Postgres for live state**: turn latency budget is < 50 ms;
row-level locking and JSONB churn per trick would add avoidable pressure.
**Why not Redis alone**: durability requirements (users, history, money-
adjacent ratings) demand ACID.

## Consequences

- Horizontal scaling later = sticky WS sessions per room + Redis pub/sub;
  schema already models per-room ownership.
- Docker compose provides both services with healthchecks and named
  volumes; host bind data for dev volumes lives on E:.
