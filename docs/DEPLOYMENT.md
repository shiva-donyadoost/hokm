# Deployment

## Development (hot reload)

```bash
cp .env.example .env      # fill secrets locally
.\dev.bat                 # Windows: remaps host 8080 if Hyper-V excluded it
docker compose up --build
# frontend  http://localhost:5173  (Vite dev server, proxies /api → backend)
# backend   http://localhost:8080  (or BACKEND_HOST_PORT, e.g. 18080)
```

Host toolchains (for fast native test loops) live on E: — see
`agents.md` HARD RULES and README "Development".

```bash
cd backend && go test ./...            # unit + integration (WS, engine)
HOKM_TEST_PG_DSN="postgres://hokm:change-me-dev-only@localhost:5432/hokm?sslmode=disable" \
  go test ./internal/infra/postgres/ -count=1
go run ./cmd/simulator -games 300 -strategy hard
cd ../frontend && npm run dev
```

## Production

```bash
export JWT_SECRET=<32+ random chars>
export POSTGRES_PASSWORD=<strong password>
docker compose -f docker-compose.prod.yml up --build -d
```

- One deployable: the Go image embeds the built frontend (`WEB_DIR=/web`)
  and serves SPA + API + WebSocket on `:8080`.
- Migrations apply automatically at startup (transactional, idempotent).
- Healthchecks gate `depends_on`; services restart unless stopped.
- Data: named volumes `pgdata_prod`, `redisdata_prod`. On this host,
  Docker Desktop's WSL2 data disk lives on `E:\Docker\data` (see
  `agents.md` lesson 1), so volume bytes land on E:.

## Graceful shutdown

`SIGTERM`/`SIGINT` → HTTP server drains (10 s) → in-flight matches end
client-side; live state is in memory by design (ADR-0006), so container
restarts drop running matches — reconnecting clients receive the room
state and can start a fresh game.

## Observability

- Structured JSON logs (slog) on stdout.
- `GET /api/metrics` — Prometheus text format (requests, WS sessions,
  active games, matches, AI decision time).

## Windows host notes

- `dl.google.com` is blocked on this network: fetch Go zips from the
  Aliyun mirror (`mirrors.aliyun.com/golang`).
- Keep Go/Node caches on E: (`GOPATH`, `GOMODCACHE`, `GOCACHE`,
  npm `cache`) — C: has ~3 GB free.
