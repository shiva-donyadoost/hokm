# IMPLEMENTATION PROMPT — HOKM AI PLATFORM

You are the lead software architect and senior full-stack engineer responsible for implementing this project.

Build a production-oriented, scalable online Iranian Hokm platform with multiplayer support and professional-level AI players.

The project must be designed as a real product, not a prototype or demo.

---

# 0. READ THE RULES FIRST

Before doing anything:

1. Read `agents.md`.
2. Treat every HARD RULE in `agents.md` as mandatory.
3. Do not start implementation until the repository and environment have been inspected.
4. Respect the E: drive storage requirement.
5. Do not bypass the required phased workflow.

If `agents.md` does not exist, create it using the required hard rules and then continue.

---

# 1. FIRST ACTION — ANALYZE AND PLAN

Immediately after receiving this prompt:

DO NOT start implementing random features.

First inspect:

* Repository
* Existing files
* Existing Git history
* Go installation
* Node.js installation
* Docker
* Docker Compose
* PostgreSQL availability/configuration
* Redis availability/configuration
* Current project location
* Available disk space
* E: drive
* Docker Desktop storage configuration
* WSL configuration if applicable

Verify that the project can be developed primarily from E:.

If the project is currently located on C:, do not create a second copy silently.

Report the issue and provide a safe migration plan.

---

# 2. CREATE THE IMPLEMENTATION ROADMAP

Before implementation, create:

`docs/IMPLEMENTATION_PLAN.md`

The plan must divide the entire project into phases.

At minimum use the following phases:

## Phase 0 — Environment and Repository Foundation

* Repository setup
* Git configuration
* AGENTS.md
* Documentation structure
* Docker foundation
* Docker Compose
* Environment configuration
* E: drive verification

## Phase 1 — Domain Model and Hokm Game Engine

Implement the complete core game engine without HTTP/WebSocket/UI.

Include:

* Card
* Deck
* Player
* Team
* Room-independent Game
* Round
* Trick
* Trump
* Hakem
* Game State
* State Machine
* Rule Engine
* Scoring

The Game Engine must be independently testable.

## Phase 2 — Comprehensive Game Engine Testing

Build extensive unit tests and property/invariant tests.

Test:

* 52-card deck
* No duplicate cards
* Shuffle
* Deal
* Hakem selection
* Trump
* Follow Suit
* Invalid moves
* Trick winner
* Turn rotation
* Scoring
* Round completion
* Game completion

## Phase 3 — Go Backend Architecture

Implement:

* Clean Architecture
* Domain layer
* Application layer
* Infrastructure layer
* HTTP layer
* Configuration
* Dependency injection
* Logging
* Error handling

## Phase 4 — Authentication and Users

Implement:

* Registration
* Login
* Password hashing
* JWT
* Refresh tokens
* User profile
* Avatar
* Authorization

## Phase 5 — Room and Lobby System

Implement:

* Create Room
* Join Room
* Leave Room
* Public Room
* Private Room
* Friends-only Room
* Room Code
* Invite Link
* Lobby
* Ready
* Host
* Kick
* AI slots

## Phase 6 — WebSocket Real-Time Infrastructure

Implement:

* WebSocket connection
* Authentication
* Room subscription
* Commands
* Events
* Broadcast
* Private player events
* Reconnection
* Disconnect handling
* Heartbeat
* Timeout

## Phase 7 — Multiplayer Gameplay

Connect the Game Engine to the Room and WebSocket layers.

Implement the complete real-time Hokm experience.

## Phase 8 — Frontend

Use a modern frontend framework.

Preferred:

React + TypeScript

but evaluate the repository and architecture before selecting the final stack.

Implement:

* Home
* Login
* Register
* Profile
* Room browser
* Create Room
* Join Room
* Lobby
* Game Table
* Result
* Leaderboard
* History
* Settings

## Phase 9 — Mobile-First Game UI

Create a professional mobile-first Hokm table.

Implement:

* Player positions
* Cards
* Hand
* Trick area
* Trump indicator
* Score
* Turn indicator
* Team indicator
* Animations
* Touch interactions
* Responsive desktop layout

## Phase 10 — AI Engine

Implement professional-level AI.

Architecture:

Rule-Based
→ Heuristic
→ Probability/Card Inference
→ Monte Carlo
→ Future RL integration

Implement AI difficulty levels:

* Easy
* Medium
* Hard
* Expert
* Pro

AI must never access hidden information.

## Phase 11 — PostgreSQL and Redis

Implement persistence.

PostgreSQL:

* Users
* Profiles
* Rooms
* Games
* Game Players
* Teams
* Results
* Statistics
* Ratings
* Chat
* Game Events

Redis:

* Active Rooms
* Active Game State
* Presence
* Sessions
* Rate Limiting
* Temporary state

## Phase 12 — Chat and Social Features

Implement:

* Room Chat
* System Messages
* Player Presence
* Basic moderation
* Rate limiting

## Phase 13 — Ranking and Statistics

Implement:

* Rating
* Leaderboard
* Win Rate
* Games Played
* Wins
* Losses
* Streak
* Rank

Design the rating system so Elo/Glicko can be supported.

## Phase 14 — Reconnection and Resilience

Implement:

* Reconnection
* Session recovery
* Game state recovery
* Disconnect timeout
* Temporary AI takeover if enabled
* Graceful recovery

## Phase 15 — Security and Anti-Cheat

Implement:

* Server authoritative gameplay
* Input validation
* Authorization
* Rate limiting
* Secure headers
* Hidden-information protection
* WebSocket validation
* Abuse prevention
* Secret management

## Phase 16 — Full Testing

Implement:

* Unit tests
* Integration tests
* E2E tests
* Multiplayer E2E
* AI simulation tests
* Reconnection tests
* Security tests

At minimum run AI-vs-AI simulations and verify that thousands of games can complete without illegal moves or crashes.

## Phase 17 — Observability and Performance

Implement:

* Structured logs
* Metrics
* Error tracking hooks
* Game performance monitoring
* WebSocket monitoring
* AI decision timing
* Active game metrics

## Phase 18 — Production Docker Environment

Finalize:

* Dockerfile
* Docker Compose
* Development configuration
* Production configuration
* Health checks
* Database migrations
* Redis configuration
* Environment variables
* Graceful shutdown

## Phase 19 — Documentation and Release

Finalize:

* README
* Architecture documentation
* API documentation
* WebSocket documentation
* Game Rules
* AI documentation
* Database documentation
* Security documentation
* Deployment documentation
* ADRs

Then create a release checklist.

---

# 3. PHASE EXECUTION RULE

After creating the plan:

Start with Phase 0.

For every phase:

1. Read relevant previous lessons from `agents.md`.
2. Read relevant ADRs.
3. Implement the phase.
4. Write tests immediately with the implementation.
5. Run tests.
6. Run lint/format/static analysis.
7. Run Docker tests where applicable.
8. Run E2E tests where applicable.
9. Fix failures.
10. Document meaningful failures as Lessons Learned in `agents.md`.
11. Update documentation.
12. Update ADRs when required.
13. Review Git diff.
14. Create a meaningful Git commit.
15. Mark the phase complete.
16. Only then continue to the next phase.

Never implement all phases first and test later.

---

# 4. GIT COMMIT POLICY

Commit every coherent change.

Use Conventional Commits.

Examples:

feat(game): implement card model

feat(game): implement trick validation

test(game): add trick winner tests

feat(room): implement room manager

feat(ws): implement authenticated websocket sessions

fix(game): prevent invalid off-suit card

feat(ai): implement heuristic card evaluation

test(e2e): add four-player game flow

docs(adr): document authoritative game state

chore(docker): configure local development stack

Do not create vague commit messages.

---

# 5. GAME RULES

Implement the standard Iranian Hokm rules.

The known requirement for Hakem selection is:

The first player who receives an Ace during the Hakem-selection process becomes Hakem.

Initial dealing:

* Deal 5 cards to each player.
* Hakem sees their initial 5 cards.
* Hakem chooses Trump.
* Deal the remaining cards.
* Each player ends with 13 cards.

Gameplay:

* 13 tricks per round.
* The lead Suit must be followed if possible.
* If a player does not have the lead Suit, they may play another Suit or Trump.
* Highest Trump wins if Trump is played.
* Otherwise highest card of Lead Suit wins.
* Trick winner leads the next Trick.

Do not silently invent ambiguous rules.

If a specific Iranian Hokm rule is ambiguous, document the ambiguity and make the rule configurable where appropriate.

---

# 6. GAME ENGINE REQUIREMENTS

The Game Engine must be deterministic where possible.

The Engine must expose clean commands such as:

* StartGame
* SelectHakem
* DealInitialCards
* SelectTrump
* DealRemainingCards
* PlayCard
* CompleteTrick
* CompleteRound
* CompleteGame

Do not allow arbitrary mutation of Game State.

Prefer command-driven state transitions.

---

# 7. SERVER AUTHORITATIVE ARCHITECTURE

The Go backend is the authority.

The browser is untrusted.

Every gameplay command must be validated server-side.

Examples:

PLAY_CARD

CHOOSE_TRUMP

READY

SURRENDER

The server must validate:

* Authentication
* Authorization
* Player membership
* Game phase
* Turn
* Card ownership
* Rule legality

---

# 8. AI

Build the AI as an independent module.

Do not couple AI implementation to HTTP, WebSocket, or UI.

AI interface should support multiple strategies.

Example conceptual interface:

```text
PlayerStrategy
    DecideTrump()
    DecideCard()
```

Implement:

* EasyStrategy
* MediumStrategy
* HardStrategy
* ExpertStrategy
* ProStrategy

AI should reason using an Information Set.

AI should track:

* Played cards
* Remaining possible cards
* Trump
* Lead Suit
* Trick history
* Partner behavior
* Opponent probabilities

Prepare the architecture for future Reinforcement Learning.

---

# 9. FRONTEND

Use TypeScript.

Prefer React unless repository analysis provides a strong reason to use another framework.

The frontend must be:

* Mobile-first
* Responsive
* Accessible
* Touch-friendly
* Performance-conscious

Do not place game authority in frontend code.

---

# 10. E2E TESTING

Use a real browser-based E2E framework such as Playwright if appropriate.

Test actual user flows.

At minimum:

### Authentication

Register → Login → Profile

### Room

Create Room → Join → Lobby → Ready

### Game

Four Players → Start → Hakem → Trump → Deal → Tricks → Result

### Multiplayer

Multiple browser contexts should participate in the same Room.

### Invalid Action

Attempt invalid card → Server rejects → UI shows appropriate feedback.

### Reconnection

Disconnect Player → Reconnect → Restore state → Continue game.

### AI

Human + AI → Start → Complete game.

---

# 11. AI SIMULATION

Create a simulation environment that can run games without UI.

Support:

AI vs AI

Run large batches of games.

Validate:

* No illegal moves
* No impossible states
* No crashes
* Game always terminates
* Scores are consistent
* Cards remain unique
* Each player ends with correct card counts

Record useful statistics.

---

# 12. DOCKER

Docker Compose must provide the complete local development environment.

The developer should be able to run the project using a documented command such as:

```bash
docker compose up --build
```

Services should include at minimum:

* backend
* frontend
* postgres
* redis

Add health checks.

Use persistent volumes where necessary.

Ensure development data is stored on E: where the host configuration permits.

---

# 13. E DRIVE REQUIREMENT

Treat E: as the primary development drive.

Before generating large files or installing dependencies:

verify disk location.

Avoid unnecessary files on C:.

Pay special attention to:

* Docker Desktop
* WSL
* npm cache
* Go cache
* model files
* PostgreSQL volumes
* Redis volumes
* build output
* test artifacts

Document any Windows/Docker configuration that must be manually changed.

Never claim storage has moved unless it has actually been verified.

---

# 14. DOCUMENTATION

Maintain documentation continuously.

Do not wait until the end.

Required:

`README.md`

`docs/ARCHITECTURE.md`

`docs/GAME_RULES.md`

`docs/API.md`

`docs/WEBSOCKET.md`

`docs/AI.md`

`docs/DATABASE.md`

`docs/SECURITY.md`

`docs/DEPLOYMENT.md`

`docs/IMPLEMENTATION_PLAN.md`

`docs/decisions/`

---

# 15. ADR POLICY

Create ADRs before significant architectural decisions.

Examples:

ADR-0001-project-architecture

ADR-0002-game-state-management

ADR-0003-websocket-architecture

ADR-0004-server-authoritative-gameplay

ADR-0005-ai-architecture

ADR-0006-postgresql-redis-strategy

ADR-0007-frontend-framework

ADR-0008-authentication

ADR-0009-docker-development-environment

Do not create ADRs merely as empty placeholders.

Each ADR must explain an actual decision.

---

# 16. ERROR → LESSON WORKFLOW

Whenever an implementation fails:

Do not simply fix the error and move on.

Instead:

1. Identify the problem.
2. Reproduce it.
3. Determine root cause.
4. Fix it.
5. Add a regression test.
6. Add a Lesson Learned to `agents.md`.
7. Commit the fix.

This applies to:

* Compilation errors
* Runtime errors
* Test failures
* E2E failures
* Docker errors
* Database errors
* WebSocket errors
* Race conditions
* AI bugs
* UI bugs
* Deployment issues
* Incorrect assumptions

---

# 17. FINAL QUALITY GATE

The project is considered complete only when:

* All phases are complete.
* All required tests pass.
* E2E flows pass.
* Docker Compose starts successfully.
* PostgreSQL works.
* Redis works.
* WebSocket multiplayer works.
* Reconnection works.
* AI can complete games.
* AI does not access hidden information.
* Game rules are server authoritative.
* Security checks pass.
* Documentation is complete.
* ADRs are complete.
* Lessons Learned are documented.
* Git history contains meaningful commits.
* No secrets are committed.
* No unnecessary project data is stored on C:.
* The application is reproducible from the documented setup.

---

# START NOW

Your first response/action must be:

1. Inspect the repository and environment.
2. Read/create `agents.md`.
3. Verify E: drive constraints.
4. Analyze the architecture.
5. Create `docs/IMPLEMENTATION_PLAN.md`.
6. Create the initial ADRs.
7. Present the complete phased implementation plan.
8. Start Phase 0.
9. Implement Phase 0.
10. Test Phase 0.
11. Commit Phase 0.
12. Continue sequentially through the phases.

Do not stop after creating the plan unless a blocking environment problem requires user intervention.

Do not ask for confirmation between phases.

Proceed autonomously according to the plan and the HARD RULES.
