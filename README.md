# HOKM AI PLATFORM

Online Iranian Hokm (حکم) platform: multiplayer rooms, real-time play, and
professional-level AI opponents.

- **Backend**: Go 1.27, clean architecture, server-authoritative game
  engine, WebSocket realtime, PostgreSQL + Redis, Elo ratings.
- **Frontend**: React + TypeScript (Vite), mobile-first game table.
- **AI**: strategy ladder (Easy → Pro) reasoning strictly over public
  information; Monte Carlo for Expert/Pro; RL-ready interface.

## Quick start

```bash
cp .env.example .env   # fill in secrets locally
.\dev.bat              # or: .\dev.ps1   (Windows)
docker compose up --build
# open http://localhost:5173
```

Production (single deployable, frontend baked into the Go image):

```bash
export JWT_SECRET=<32+ chars> POSTGRES_PASSWORD=<strong>
docker compose -f docker-compose.prod.yml up --build -d   # serves :8080
```

## Documentation

- [Architecture](docs/ARCHITECTURE.md) · [Game rules](docs/GAME_RULES.md)
- [REST API](docs/API.md) · [WebSocket protocol](docs/WEBSOCKET.md)
- [AI](docs/AI.md) · [Database](docs/DATABASE.md)
- [Security](docs/SECURITY.md) · [Deployment](docs/DEPLOYMENT.md)
- [Implementation plan](docs/IMPLEMENTATION_PLAN.md) · [ADRs](docs/decisions/)
- [Agent rules](agents.md)

## Development

Toolchains live on E: (see `agents.md` HARD RULES): Go at `E:\tools\go`,
Node at `E:\tools\nodejs`; caches under `E:\tools\*`. Docker Desktop WSL2
data lives on `E:\Docker\data`.

```bash
cd backend && go test ./...      # engine, ws multiplayer e2e, ai, auth
cd backend && go run ./cmd/simulator -games 300 -strategy hard
cd frontend && npm install && npm run dev
```

## Validation status

- Engine: 52-card invariants, random first hakem, follow-suit, trick winner,
  scoring — unit + property tests, 200-game random simulation.
- Multiplayer: full-stack E2E — 4 WS clients complete a match; 1 human +
  3 AI completes a match; disconnect → AI takeover → reconnect.
- AI: 1,000 expert matches → 139k tricks, 568k decisions, 0 illegal moves.
- Production compose verified end-to-end (SPA, API, auth, metrics).

