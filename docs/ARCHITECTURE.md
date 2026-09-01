# Architecture

Monorepo with a Go backend and a React frontend. See ADRs in
`docs/decisions/` for the decisions behind this structure.

```
backend/
  cmd/server       composition root: config -> stores -> http server
  cmd/simulator    AI-vs-AI batch runner (validation + statistics)
  internal/
    game           pure Hokm engine (no I/O imports) — the domain core
    ai             strategies over Information Sets (never hidden info)
    room           lobby domain: rooms, membership, host controls
    app            use-cases: users, tables (game sessions), chat
    auth           bcrypt passwords, JWT access, opaque refresh tokens
    rating         Elo math + ScoreStore (memory/postgres)
    config         env-driven configuration
    httpapi        REST transport: handlers, middleware, errors
    ws             WebSocket transport: hub, sessions, envelope protocol
    infra/postgres durable stores + embedded migrations
    infra/redisx   rate limiting, presence (fail-open)
    infra/memory   in-memory stores (dev fallback)
    metrics        Prometheus-text metrics registry
  migrations/      versioned SQL, embedded, applied at startup

frontend/
  src/api          REST client with transparent token refresh
  src/protocol     TS mirror of the WS envelope + game types
  src/state        zustand stores (auth, live game)
  src/components   Card, PlayerSeat, TrickArea, GameTable, ChatPanel
  src/pages        Auth, Rooms, Room (lobby+table), Profile, Leaderboard
```

## Request flow (gameplay)

1. REST: register/login → JWT access + refresh tokens.
2. WS upgrade `/api/ws?token=` → authenticated session.
3. Client `SUBSCRIBE {room_id}` → membership checked → room snapshot +
   chat history (+ live game view if a match is running).
4. Host `START_GAME` → engine created for the room's four seats →
   random first hakem (ADR-0012) → initial deal → clients receive
   per-seat `STATE`.
5. Hakem `SELECT_TRUMP` → remaining deal → `trick_play`.
6. Players `PLAY_CARD`; the server resolves tricks/rounds/matches
   automatically, projects a per-seat `STATE` view, and fans out public
   `EVENTS`.
7. On completion the match is recorded; ratings/statistics update.
   Host may `REPLAY_GAME` to deal a new match with the same seats
   (ADR-0010). Profile (`GET /api/stats`) and Leaderboard
   (`GET /api/leaderboard`) read those stats.
8. Disconnect → grace deadline (`TURN_TIMEOUT_SECONDS`) → fallback AI plays
   for the absent human; reconnect rebinds and replays state.

## Authority boundaries

- The engine is the only state mutator; transport layers validate
  identity/membership and delegate (ADR-0004).
- `SeatView` is the only state projection that leaves the engine toward a
  client; hidden information never crosses it.
- AI strategies see `InformationSet`, built exclusively from `SeatView` +
  public events (ADR-0005).
