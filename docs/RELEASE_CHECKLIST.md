# Release checklist

Verify every item before tagging a release.

## Tests

- [ ] `cd backend && go test ./... -count=3` — all green (flaky-free)
- [ ] Postgres integration: `HOKM_TEST_PG_DSN=... go test ./internal/infra/postgres/ -count=1`
- [ ] Frontend: `npm run build` and `npx eslint src` clean
- [ ] Simulator validation batch: `go run ./cmd/simulator -games 1000 -strategy expert`
      → 0 illegal moves, 0 non-terminating games, team balance reported

## Runtime

- [ ] `docker compose up --build` — dev stack healthy (postgres, redis,
      backend, frontend)
- [ ] `docker compose -f docker-compose.prod.yml up --build -d` with real
      `JWT_SECRET`/`POSTGRES_PASSWORD` — healthy
- [ ] Manual flow: register → login → create room → 4th seat AI → start →
      trump → tricks → result → leaderboard updated
- [ ] Disconnection mid-match → AI takeover completes match → reconnect
      shows the result
- [ ] `/api/health` and `/api/metrics` respond

## Security

- [ ] No secrets committed (`git grep` for passwords/keys; `.env` ignored)
- [ ] Production refuses weak `JWT_SECRET` (starts and logs clearly, or
      fails fast — verified)
- [ ] Refresh-token reuse rejected (rotation works)

## Documentation

- [ ] README, docs/* and ADRs reflect the current behavior
- [ ] Lessons Learned updated in `agents.md` for any new incident
- [ ] `docs/IMPLEMENTATION_PLAN.md` statuses current

## Known limitations (documented, acceptable)

- Playwright browser E2E suite not yet added (protocol-level E2E exists in
  Go tests); follow-up in `docs/IMPLEMENTATION_PLAN.md`.
- Chat history is in-memory per room; durable log written to Postgres only
  when the `chat_messages` pipeline is wired to the sink (schema ready).
- Leaderboard/history/streak UI pages pending final frontend polish.
