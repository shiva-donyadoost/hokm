# ADR-0001: Project Architecture

Date: 2026-08-29
Status: Accepted

## Context

HOKM is a real-time 4-player card game with professional AI opponents. The
product must scale to many concurrent rooms, remain server-authoritative,
and be developed primarily on a constrained Windows host (E: drive).

## Decision

Monorepo with two deployables and clean-architecture layering:

```
backend/    Go 1.27 — domain, application, infrastructure, HTTP, WS
frontend/   React + TypeScript (Vite)
docs/       architecture, ADRs, game rules, deployment
docker-compose.yml (dev), production overrides
```

Backend layering (dependencies point inward only):

- `internal/game`   — pure Hokm engine (no I/O imports) [H4]
- `internal/app`    — use-cases orchestrating engine + repositories
- `internal/infra`  — postgres, redis, persistence adapters
- `internal/httpapi`— REST handlers, middleware
- `internal/ws`     — WebSocket hub, sessions, event fan-out
- `internal/ai`     — strategies operating on Information Sets [H5]

## Alternatives considered

- **Two repos**: simpler CI per service, but cross-cutting protocol changes
  require coordinated PRs; monorepo keeps API contracts and engine tests
  atomic with backend changes.
- **Microservices**: unwarranted at this scale; the game loop is
  latency-sensitive and in-memory. Room/game state lives in process (with
  Redis snapshots); PostgreSQL for durable identity/history.
- **Node full-stack**: single language, but Go gives better concurrency
  primitives for thousands of persistent WS connections and predictable
  memory per session.

## Consequences

- Engine is testable without any infrastructure.
- Protocol (WS envelope) is defined once in Go and mirrored in TS types.
- Future extraction of services (e.g., matchmaking) is possible because
  application-layer use-cases do not depend on transport.
