# IMPLEMENTATION PLAN — HOKM AI PLATFORM

Living document. Update the Status column as phases complete.
Rules: phases are executed strictly in order; each phase is implemented,
tested, linted, documented, committed (Conventional Commits), and marked
complete before the next begins. See `agents.md` HARD RULES.

## Phases

| # | Phase | Scope | Status |
|---|-------|-------|--------|
| 0 | Environment & Repository Foundation | git init, repo layout, AGENTS.md, docs structure, docker-compose (postgres/redis), .env.example, E: verification | ✅ done |
| 1 | Domain Model & Hokm Game Engine | Card, Deck, Player, Team, Game, Round, Trick, Trump, Hakem, state machine, rule engine, scoring — pure Go, zero infra imports | ✅ done |
| 2 | Comprehensive Game Engine Testing | 52-card invariants, dealing, hakem-by-ace, follow-suit, trick winner, turn rotation, scoring, completion; property tests + full-game invariants | ✅ done |
| 3 | Go Backend Architecture | Clean architecture: domain / application / infrastructure / HTTP layers, config (env), DI wiring, slog logging, typed errors | ✅ done |
| 4 | Authentication & Users | Register/login, bcrypt, JWT access+refresh, profile, authorization middleware | ✅ done |
| 5 | Room & Lobby System | create/join/leave, public/private/friends, room code, lobby, ready, host, kick, AI slots | ✅ done |
| 6 | WebSocket Infrastructure | single authenticated WS endpoint, room subscription, command/event envelope, heartbeat, reconnect, disconnect handling | planned |
| 7 | Multiplayer Gameplay | engine wired to rooms + WS; full real-time Hokm flow (hakem → trump → deal → 13 tricks → score) | planned |
| 8 | Frontend | React + TypeScript (Vite): home, auth, profile, room browser, create/join, lobby, game table, results, leaderboard, history, settings | planned |
| 9 | Mobile-First Game UI | player positions, hand, trick area, trump indicator, scores, turn/team indicators, animations, touch, responsive desktop | planned |
| 10 | AI Engine | PlayerStrategy interface; Easy→Pro strategies; information-set reasoning (played cards, remaining, probabilities); RL-ready | planned |
| 11 | PostgreSQL & Redis | schema + migrations (users, profiles, rooms, games, players, teams, results, stats, ratings, chat, events); Redis active rooms/state/presence/sessions/rate limits | planned |
| 12 | Chat & Social | room chat, system messages, presence, moderation, rate limiting | planned |
| 13 | Ranking & Statistics | Elo-ready rating service, leaderboard, win rate, streaks, rank | planned |
| 14 | Reconnection & Resilience | session recovery, state replay, disconnect timeout, optional AI takeover | planned |
| 15 | Security & Anti-Cheat | server-authoritative validation, rate limits, secure headers, hidden-info protection, WS validation, secret management | planned |
| 16 | Full Testing | unit + integration + E2E (Playwright) + multiplayer E2E + AI simulation batches (zero illegal moves) | planned |
| 17 | Observability & Performance | structured logs, metrics (WS, engine, AI timing), error hooks | planned |
| 18 | Production Docker | Dockerfiles, dev/prod compose, health checks, migrations, graceful shutdown | planned |
| 19 | Documentation & Release | README, ARCHITECTURE, GAME_RULES, API, WEBSOCKET, AI, DATABASE, SECURITY, DEPLOYMENT, ADRs, release checklist | planned |

## Game Rules (authoritative reference)

Standard Iranian Hokm, 4 players, fixed teams (2v2, partners face each other):
0. **Hakem selection**: deal one card at a time to each player in turn; the
   first player to receive an **Ace** becomes Hakem (the dealer rotates each
   game; Hakem role rotates to the winning team's next player each round-win
   as configured). Ambiguities are documented in `docs/GAME_RULES.md` and
   made configurable.
1. **Initial deal**: 5 cards to each player (starting left of Hakem).
2. **Trump**: Hakem views their 5 cards and selects trump suit.
3. **Remaining deal**: remaining 8 cards each → 13 cards per player.
4. **Play**: 13 tricks per round. Lead suit must be followed if possible;
   otherwise any card (including trump). Highest trump beats highest lead
   suit; otherwise highest lead-suit card wins. Trick winner leads next.
5. **Round scoring**: a team taking ≥7 tricks wins the round.
6. **Game scoring**: first team to win the configured number of rounds
   (default 7) wins the match.

## Command Surface (engine)

`StartGame, SelectHakem, DealInitialCards, SelectTrump, DealRemainingCards,
PlayCard, CompleteTrick, CompleteRound, CompleteGame` — command-driven only;
no arbitrary state mutation.

## Phase 0 acceptance checklist

- [ ] git repo initialized on E: with initial commits
- [ ] agents.md present
- [ ] docs/ structure + IMPLEMENTATION_PLAN.md + ADRs 0001–0009
- [ ] docker-compose.yml with postgres, redis, healthchecks
- [ ] .env.example (no secrets committed)
- [ ] E: drive / toolchain verification documented
