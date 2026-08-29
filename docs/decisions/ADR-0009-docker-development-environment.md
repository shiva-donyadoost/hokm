# ADR-0009: Docker Development Environment

Date: 2026-08-29
Status: Accepted

## Context

Developers/agents run on a constrained Windows host (C: ~3 GB free, E: is
the primary drive). The full stack needs PostgreSQL + Redis + backend +
frontend with reproducible startup and health checks.

## Decision

- `docker-compose.yml` (dev default): `postgres:17-alpine`,
  `redis:7-alpine`, `backend` (Go, built via `backend/Dockerfile`),
  `frontend` (Vite dev server in container for parity). All with
  healthchecks; `depends_on: condition: service_healthy`.
- **Named Docker volumes** for postgres/redis data. Docker Desktop's WSL2
  data disk lives on `E:\Docker\data` (host-level config, see
  `docs/DEPLOYMENT.md`), so volume bytes land on E: without bind-mount
  permission complexity.
- `docker compose up --build` is the documented dev entrypoint.
- Backend reads config from env (12-factor); `.env.example` documents every
  variable; compose consumes `.env` (gitignored).
- Go builds in Docker use `GOMODCACHE`/`GOCACHE` in named volumes so module
  downloads persist between builds.
- Host-side Go tests run natively (toolchains on `E:\tools\go`,
  `E:\tools\nodejs`) with caches on E:.

## Alternatives considered

- **Host bind mounts for DB data (E:\...)**: rejected — Windows bind-mount
  filesystem semantics (inotify, permissions) degrade Postgres and dev DX.
- **No Docker, host services**: unreproducible; contradicts
  `docker compose up --build` requirement.
- **Kubernetes/kind**: unjustified complexity for this stage.

## Consequences

- First `compose up` pulls images (postgres/redis/node) — one-time cost.
- CI (future) reuses the same compose for integration tests.
