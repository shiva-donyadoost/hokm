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

### H10. ADR for every change

- An ADR **MUST** be considered for every change, not only "large" ones.
- Before implementing: check `docs/decisions/` for an existing ADR that
  already covers the decision. If none does, write one first.
- The ADR must record the problem, the chosen approach, alternatives
  rejected, and trade-offs. Never ship a behavior change without this.
- Trivial typo/format-only diffs may reuse the latest related ADR and
  note that reuse in the commit body; they still may not invent
  architecture in code comments instead of an ADR.

### H11. Debug results belong in this file

- After every debug cycle (compile, test, runtime, Docker, DB, E2E, UI),
  append a Lesson Learned below **in the same change**. The lesson must
  name the symptom, the root cause, and the rule that prevents repeating
  it. "Fixed it" without a lesson is incomplete work.
- Search LESSONS LEARNED before retrying a failed approach. Do not
  re-introduce a bug that already has a lesson.
- Lessons are the project's memory: write them so a future agent who
  never saw the incident can avoid it.

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
6. **Vite dev server in Docker on Windows goes stale**: file-change events
   do not propagate through Docker Desktop bind mounts, so a long-running
   vite process keeps serving its old module graph even after the host
   files change (symptom: new UI code "missing" despite hard refresh).
   Fix: `CHOKIDAR_USEPOLLING=true` (+ interval) in the frontend compose
   service, and recreate the container after large changes. Verify by
   diffing the served module (`curl /src/<file>`) against disk, not by
   trusting the browser.

### Dev tooling (2026-08-30)

12. **Windows Hyper-V excluded ports vs Docker publish**: `docker compose up` failed with `listen tcp 0.0.0.0:8080: bind: An attempt was made to access a socket in a way forbidden by its access permissions` while netstat showed nothing on 8080. Root cause: WinNAT/Hyper-V `excludedportrange` (here 8061-8160) includes 8080; this is not "port in use". Fix: keep container `APP_ADDR=:8080`, publish `${BACKEND_HOST_PORT:-8080}:8080`, and have `dev.ps1` probe bindability (TcpListener on 0.0.0.0) then pick 8080/18080/28080/8000. Rule: diagnose Windows Docker publish failures with `netsh interface ipv4 show excludedportrange protocol=tcp` before assuming a leftover process; never restart WinNAT as the project-level fix (admin, races on reboot). `dev.bat` is a wrapper around `dev.ps1`.
7. **PowerShell `$args` is a reserved automatic variable**: declaring
   `function F($args)` and splatting an array (`F @("up","-d")`) binds only
   the first element — the rest land in the automatic variable. Symptom:
   `docker compose` printed its usage help. Fix: named parameter with array
   type (`param([string[]]$ComposeArgs)` + `docker compose @ComposeArgs`).
8. **Compose build contexts must match the Dockerfile's COPY paths**: the
   prod Dockerfile (repo-root context, `COPY backend/...`) silently broke
   the dev compose build (`build: ./backend` → backend-relative context).
   A running container from the previous image masked the failure until the
   next rebuild. Unify on repo-root context + a `target:` stage selector
   (`base` for dev, `prod` with baked frontend), and make launcher scripts
   abort on failed builds instead of serving stale containers.

### Frontend encoding (2026-08-30)

9. **PowerShell `Get-Content` corrupts BOM-less UTF-8 source files**: PS 5.1
   reads BOM-less files with the ANSI codepage, so editing frontend sources
   through `Get-Content`/`Set-Content` or `WriteAllText` after such a read
   double-encodes every multibyte character (middle dots, suit symbols,
   emojis) into visible mojibake in the UI. Symptom: garbled text on screen
   while the dev server happily serves it. Fix: rewrite the affected files
   as pure ASCII with the write/edit tools; verify with a non-ASCII byte
   scan of `src/` AND of the served bundle before committing. Rule: never
   edit BOM-less UTF-8 sources through PS 5.1 string pipelines.

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

### Frontend lint (2026-08-31)

10. **ESLint flat config rejects `--ext`**: `npm run lint` was `eslint src --ext .ts,.tsx`, which ESLint 9 (`eslint.config.js`) rejects as `Invalid option '--ext'`. Symptom: lint "fails" before any file is checked. Fix: `eslint src` only; file filtering lives in `eslint.config.js` `files`. Rule: never pass legacy CLI flags that the flat config already covers.
11. **`requestAnimationFrame` is not a default ESLint global**: adding a fly-in animation produced `no-undef` even though TypeScript `lib: DOM` knows the API. Symptom: `npx eslint src` errors on TrickArea while `tsc` is clean. Fix: declare `requestAnimationFrame` / `cancelAnimationFrame` next to `setTimeout` in `eslint.config.js` globals. Rule: any new browser API used in `src/` must be listed in that globals block.

### Wave 4 / ADR-0012 (2026-09-01)

13. **First-to-7 round cut breaks "exactly 13 tricks" assertions**: after `CompleteTrick` ends the round at 7, `table_ai_test`, `table_ws_test`, `table_reconnect_test`, and `ai_test` still expected `LastTrick.Number == 13` / `completed == 13`. Symptom: E2E and AI strategy tests fail even though the engine is correct. Fix: assert 7-13 tricks and leftover cards discarded on `CompleteRound`. Rule: when a round-termination rule changes, grep tests and the Monte Carlo rollout (`mc.go` stopped only on trick 13) — property tests that allow a range are not a substitute for updating the scripted E2E checks.
14. **System chat tests are not covered by the unit no-op**: `ChatService.System` returning without storing still left `TestTwoClientsChat` waiting for `"chatter2 joined the room"` over WS. Symptom: 5s timeout on join line. Fix: WS test waits only for player `Send` text. Rule: a unit no-op is incomplete until the protocol-level test that previously observed the message is inverted.
15. **`noUncheckedIndexedAccess` on tuple construction**: `opp[0]` from a `Suit[]` is `Suit | undefined`, so `suitOrder` failed `tsc` even though the arrays are length-2. Fix: branch on trump and return a `[Suit, Suit, Suit, Suit]` literal. Rule: do not index short arrays to build a fixed tuple; name the four suits explicitly.
16. **`matchMedia` follows the same ESLint-global rule as rAF (lesson 11)**: the mobile hand layout hook used `window.matchMedia` and needed `matchMedia` in `eslint.config.js` globals. Rule: any new browser API in `src/` is an ESLint global, not only a TypeScript lib type.

### Wave 5 / ADR-0013 (2026-09-01)

17. **Refresh logged the user out because Room treated `user === null` as "go to login"**: `RequireAuth` started with `loading: false` and `Room.tsx` called `navigate('/login')` while `/me` was still in flight. Symptom: F5 on lobby/table dumps you on the login page even with a valid refresh token. Fix: `booting` gate until `ensureFreshAccess` + `/me`; Room never navigates to login while tokens exist. Rule: protected routes must not redirect on a null user until session restore has finished, and a network blip must not clear tokens.
18. **Mobile drag missed pointer events because capture was on `e.target`**: the inner card `<img>`/`<button>` received the touch, so the wrapper's `onPointerMove`/`onPointerUp` never ran. Combined with a rotated fan, fingers could not drag. Fix: `touch-action: none`, capture on `e.currentTarget`, `pointer-events: none` on the Card, overlapping row with no rotation. Rule: pointer capture belongs on the element that owns the handlers; never assume the event target is that element on touch devices.

### Wave 6 / ADR-0014 (2026-09-01)

19. **Deal animation and drag shared one `transform` owner**: hand cards used a
    single DOM node for CSS `deal-in` (`animation-fill-mode: both`, animating
    `transform`), fan/select inline transform, and imperative drag updates.
    After the deal animation the fill kept owning `transform`, so drag appeared
    to do nothing while the UI still showed "your turn". Fix: outer node owns
    deal animation (keyframes end with `transform: none`); inner node owns
    fan/drag transforms and pointer handlers. Rule: never put a CSS animation
    that touches `transform` on the same element that must accept JS/React
    transform updates (drag, fan, select lift).
20. **Dimmed/illegal overlapping cards still hit-tested**: non-playable hand
    wrappers omitted handlers but kept default `pointer-events`, swallowing
    taps on partially covered legal cards. Fix: `pointer-events: none` on
    non-playable wrappers. Rule: in an overlapping hand, disabled cards must
    not participate in hit-testing.
21. **`playCard` silently no-oped on a dead WebSocket**: UI kept the last
    SeatView ("your turn") after `onclose`, while `playCard` required
    `readyState === OPEN` with no error and Room's reconnect effect was a
    stub. Fix: bounded auto-reconnect in `useGame.connect`, surface
    `lastError` + reconnect banner, structured client diagnostics
    (`window.__HOKM_DIAG__`). Rule: never drop a user command on a closed
    socket without visible feedback and a reconnect path.
22. **STATE carried stale `deadline_*`**: `broadcast` pushed views then called
    `rescheduleTimerLocked`, so the armed deadline was not in the message
    clients just received. Fix: reschedule before the STATE send. Rule: any
    field derived during broadcast must be computed before the payload is
    marshaled.

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

### Player avatars / ADR-0015 (2026-09-05)

23. Prefer CDN avatar SVG with stable user_id seed and initials fallback.
24. When patching Windows sources match CRLF; LF-only search fails.
