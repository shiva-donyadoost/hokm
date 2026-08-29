# AGENTS.md — HOKM AI PLATFORM

This file governs every agent and contributor working in this repository.
Treat every HARD RULE as mandatory. Violations must be corrected before any
other work continues.

---

## HARD RULES

### H1. E: drive is the primary development drive

- The repository lives at `E:\project\hokm`. Never clone/copy it to C:.
- All toolchains, caches, and build artifacts must live on E:
  (Go: `E:\tools\go`, Node: `E:\tools\nodejs`, `GOPATH/GOMODCACHE/GOCACHE`
  under `E:\tools\go\...`, npm cache under `E:\tools\npm-cache`).
- Docker Desktop data (WSL2 vhdx) lives on `E:\Docker\data`.
- Never claim storage has moved unless verified (locked-file checks, sizes).
- C: has very little free space (~3 GB). Avoid writing anything large to C:.

### H2. Phased workflow

- Implementation follows `docs/IMPLEMENTATION_PLAN.md` phase order.
- For every phase: implement → test → lint/format → fix → document lessons →
  update docs/ADRs → review diff → conventional commit → mark complete.
- Never implement all phases first and test later.

### H3. Server authority

- The Go backend is the single source of truth for game state.
- The browser/client is untrusted. Every command is validated server-side
  (auth, membership, phase, turn, card ownership, rule legality).
- No hidden information (opponent hands, undealt cards) may ever be sent to
  a client or made accessible to AI decision logic.

### H4. Game engine purity

- The Hokm game engine (domain layer) must not import HTTP, WebSocket,
  database, or UI packages. It must be independently testable.
- Game state changes only through explicit commands
  (StartGame, SelectTrump, PlayCard, ...). No arbitrary mutation.

### H5. AI fairness

- AI strategies reason only over an Information Set: their own hand, played
  cards, trump, lead suit, trick history, and public events.
- AI must never access hidden information through engine internals.

### H6. Error → Lesson workflow

When anything fails (compile, test, runtime, Docker, DB, E2E):
1. Identify the problem. 2. Reproduce it. 3. Determine root cause.
4. Fix it. 5. Add a regression test where feasible. 6. Add a Lesson Learned
below. 7. Commit the fix.

### H7. Git policy

- Conventional Commits only (`feat(game): ...`, `fix(ws): ...`).
- Commit every coherent change. No vague messages.
- Never commit secrets, `.env` (only `.env.example`), or build artifacts.

### H8. Documentation

- Keep README and `docs/` current as code changes; do not defer to the end.
- Create an ADR **before** significant architectural decisions; ADRs must
  document a real decision with real trade-offs, never placeholders.

### H9. Testing discipline

- Tests are written with the implementation, not after.
- Engine changes require engine tests (unit + invariant/property style).
- AI-vs-AI simulations must complete with zero illegal moves / crashes.
- `gofmt`/`go vet` must pass for Go; lint must pass for TypeScript.

---

## LESSONS LEARNED

### Environment (Windows host, 2026-08-29)

1. **Docker Desktop WSL data relocation**: The `DataFolder` key in
   `settings-store.json` only affects the Sailor VMM backend (default
   `C:\ProgramData\DockerDesktop\vm-data`); the GUI "Disk image location" is
   unavailable in WSL2 mode. The working path is the **`WslDataFolder`**
   (plus legacy `CustomWslDistroDir`) key — Docker then runs its internal
   `moveWSLDisk` flow on next start (unregister → move → re-register).
   Verify success by vhdx file-lock checks and the
   `HKCU\...\Lxss` `BasePath` registry value, not by settings file content.
2. **dl.google.com is blocked** on this network (all /go/*.zip return 404
   while go.dev/dl/ metadata works). Go zips must be fetched from the Aliyun
   mirror: `https://mirrors.aliyun.com/golang/go<ver>.windows-amd64.zip`.
   Node zips from nodejs.org work fine.
3. **Toolchains on E:**: Go 1.27.0 at `E:\tools\go`, Node 24.20.0 LTS at
   `E:\tools\nodejs`. User PATH updated. Because already-running processes
   do not see PATH changes, every new shell used by tooling must prepend
   `E:\tools\go\bin;E:\tools\nodejs` and set GOPATH/GOMODCACHE/GOCACHE.
4. **Docker on Windows 10 Pro**: WSL2 backend requires
   `wsl --install --no-distribution` + reboot. `wsl --import --vhd` fails
   with ERROR_FILE_EXISTS if the target folder already contains the vhdx;
   move it out first, and normalize `VhdFileName` in the Lxss registry key
   afterwards.
5. **PowerShell 5.1 quirks**: `Invoke-RestMethod` pipelines over large JSON
   arrays can flatten unexpectedly; use explicit sub-expressions and index
   results (e.g. `@(... | Where-Object ...)[0]`). HEAD requests are
   unreliable for CDN existence checks — use ranged GETs (`--range 0-100`).

### Engine (2026-08-29)

6. **Lead-suit must be captured on the first card of a trick**: the initial
   `PlayCard` implementation appended the card but never set
   `Trick.LeadSuit`, so follow-suit validation silently never fired (empty
   suit matched no hand). Caught by a scripted unit test, not by the random
   simulation — legal bots happened to follow suit anyway. Lesson: invariant
   simulations pass even when *optional* rules (like following suit) are
   unenforced, because random/first-card play rarely violates them. Always
   pair property tests with targeted negative tests for each rule.

### Transport (2026-08-29)

7. **Middleware must preserve `http.Hijacker` for WebSocket upgrades**:
   wrapping the mux with a logging `ResponseWriter` broke gorilla's
   handshake (`response does not implement http.Hijacker`) — only visible
   through a real upgrade test, never with plain httptest handler tests.
   Fix: delegate `Hijack()`/`Flush()` on the recorder. Lesson: any custom
   `ResponseWriter` wrapper must implement `http.Hijacker` and
   `http.Flusher` pass-through, and WebSocket support requires an
   end-to-end upgrade test, not just handler tests.
8. **gorilla/websocket forbids repeated reads after a failed read**: the
   first E2E client polled `ReadJSON` after a deadline error and panicked
   with "repeated read on failed websocket connection". Fix: one dedicated
   reader goroutine per client feeding a channel; assertions consume the
   channel with deadlines.

---

## PROJECT CONVENTIONS

- Language: Go 1.27 backend (`backend/`), TypeScript + React frontend
  (`frontend/`), simulation tooling under `backend/cmd/simulator`.
- Go module path: `github.com/hokm/platform` (module root `backend/`).
- Formatting: `gofmt` (Go), Prettier (TS). Vet + ESLint must pass.
- Config via environment variables, loaded in `internal/config`
  (see `.env.example`). 12-factor style.
- Logging: structured JSON logs (slog). No fmt.Println in server code.
- Errors: wrapped with `%w`, mapped to typed API errors at the HTTP edge.
- Commit style: Conventional Commits (see H7).
