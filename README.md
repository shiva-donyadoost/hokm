# HOKM AI PLATFORM

Online Iranian Hokm (حکم) platform: multiplayer rooms, real-time play, and
professional-level AI opponents.

- **Backend**: Go 1.27, clean architecture, server-authoritative game engine,
  WebSocket realtime, PostgreSQL + Redis.
- **Frontend**: React + TypeScript (Vite), mobile-first game table.
- **AI**: strategy ladder (Easy → Pro) reasoning strictly over public
  information; Monte Carlo ready; RL-ready interface.

## Quick start

```bash
cp .env.example .env   # fill in secrets locally
docker compose up --build
```

Status: work in progress — see [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md).

## Documentation

- [Architecture](docs/ARCHITECTURE.md)
- [Game rules](docs/GAME_RULES.md)
- [Implementation plan](docs/IMPLEMENTATION_PLAN.md)
- [Architecture decisions (ADRs)](docs/decisions/)
- [Agent rules](agents.md)

## Development (Windows host)

Toolchains live on E: (see `agents.md` HARD RULES): Go at `E:\tools\go`,
Node at `E:\tools\nodejs`. Go caches: `GOPATH/GOMODCACHE/GOCACHE` under
`E:\tools\go\...`; npm cache `E:\tools\npm-cache`. Docker Desktop WSL2 data
lives on `E:\Docker\data`.

```bash
# backend tests (from repo root)
cd backend && go test ./...

# frontend (once Phase 8 lands)
cd frontend && npm install && npm run dev
```
