# IMPLEMENTATION PLAN — HOKM AI PLATFORM

Living document. Update the Status column as phases complete.
Rules: phases are executed strictly in order; each phase is implemented,
tested, linted, documented, committed (Conventional Commits), and marked
complete before the next begins. See `agents.md` HARD RULES.

## Enhancement Wave 5 — Session Restore, Lobby Teams, Hand & Collect

Source: ADR-0013 (2026-09-01).

| # | Phase | Scope | Status |
|---|-------|-------|--------|
| X | Session restore | refresh must not log out; boot `/me` (refresh JWT first); stay on lobby/table URL | done |
| Y | Mobile hand + drag | overlapping row (no rotation); pointer capture on wrapper; drag-to-play and tap-to-play | done |
| Z | Lobby teams + seats | Team A (0+2) / Team B (1+3); host moves any occupant on desktop and mobile | done |
| AA | Seat score | number by each name is tricks this round; top bar still shows match rounds | done |
| AB | Collect from felt | last-trick cards stay at rest, then travel to the winner; no remount from origin | done |
| AC | Delete lobby | host `DELETE /api/rooms/{id}` lobby only, with confirm | done |

## Enhancement Wave 4 — Table Rules, Stats Fix, Session & Chat

Source: ADR-0012 (2026-09-01).

| # | Phase | Scope | Status |
|---|-------|-------|--------|
| R | Hand order + legal dim | trump-led alternating-color sort; legal bright / illegal dim; illegal not clickable/draggable, no toast | done |
| S | Mobile hand | hide opponent hand-backs; keep names/scores; local arc inside viewport | done |
| T | Stats + leaderboard | rank by wins; fix `games.room_id` FK (migration 0004); profile wins/losses after each match | done |
| U | Round cut + hakem | first to 7 tricks ends the round; round-1 hakem random; later keep-on-win / NextSeat on loss | done |
| V | Seat score + Back to rooms | team rounds won by the name; Back to rooms only if all four seats occupied | done |
| W | Session + turn bar + chat | access JWT 720h; deadline bar on every acting seat (human and AI); player-typed chat only | done |

## Enhancement Wave 3 — Lobby, Table Feel, Replay & Stats UI

Source: ADR-0010 (2026-08-30).

| # | Phase | Scope | Status |
|---|-------|-------|--------|
| M | Lobby defaults | room creator seated ready; host `POST /rooms/{id}/ai/fill` random AI | done |
| N | Trump from hand | hakem taps a dealt card; no suit-button overlay | done |
| O | Table presentation | overlapping card arc (no opacity), trump glyph, directional play + collect, no yellow winner box, fill-up turn bar | done |
| P | Replay | `REPLAY_GAME` host-only after match complete; same room/seats | done |
| Q | Stats UI | Profile shows wins/losses/rating; Leaderboard page | done |

## Enhancement Wave 2 — Gameplay UX, Timing, Persistence & Chat

Source: `impliment.md` (2026-08-30). Hard rule: **no hard-coded gameplay
values** — every timing/limit lives in the configuration layer.

| # | Phase | Scope | Status |
|---|-------|-------|--------|
| A | Configuration Architecture | GameConfig (hakem timeout 10s, card timeouts fast 5s / medium 10s / slow 15s, reconnect grace 30s, trick-winner display 3s, card-play animation 0.5s, round options) + Room settings (RoundCount 1/3/5, GameSpeed, ChatEnabled) | ? done |
| B | Timers & State Machine | server-authoritative deadlines (trump + card), auto actions on expiry (deterministic trump = most trumps; card = lowest legal), Automatic flags, race-safe single table timer | ? done |
| C | Dealing Animation | animated deal, private visibility (existing views) | ? done |
| D | Sequential Card Presentation | card-play animation 0.5s (config) | ? done |
| E | Trick Winner Animation | winner reveal 3s + collect-toward-winner animation | ? done |
| F | Round Score Display | previous round winners, match score from server (round history in view) | ? done |
| G | Card Interaction | arc hand, tap-to-select + tap-to-play, pointer drag & drop, drag cancellation | ? done |
| H | Persistent Session | refresh keeps auth + room + game state (tokens + subscribe replay + return-to-game banner) | ? done |
| I | AI Takeover | configurable grace period + takeover scheduling on disconnect | ? done |
| J | Chat Improvements | room chat toggle (server-enforced), emoji picker, unread badge | ? done |
| K | SVG Card Assets | 52 faces + card-back asset, resolver + build-time validation | ? done |
| L | Testing | unit/integration for timers, auto-play, room settings; regression suite | ? done (Playwright browser E2E remains follow-up) |

| # | Phase | Scope | Status |
|---|-------|-------|--------|
| 0 | Environment & Repository Foundation | git init, repo layout, AGENTS.md, docs structure, docker-compose (postgres/redis), .env.example, E: verification | ✅ done |
| 1 | Domain Model & Hokm Game Engine | Card, Deck, Player, Team, Game, Round, Trick, Trump, Hakem, state machine, rule engine, scoring — pure Go, zero infra imports | ✅ done |
| 2 | Comprehensive Game Engine Testing | 52-card invariants, dealing, hakem-by-ace, follow-suit, trick winner, turn rotation, scoring, completion; property tests + full-game invariants | ✅ done |
| 3 | Go Backend Architecture | Clean architecture: domain / application / infrastructure / HTTP layers, config (env), DI wiring, slog logging, typed errors | ✅ done |
| 4 | Authentication & Users | Register/login, bcrypt, JWT access+refresh, profile, authorization middleware | ✅ done |
| 5 | Room & Lobby System | create/join/leave, public/private/friends, room code, lobby, ready, host, kick, AI slots | ✅ done |
| 6 | WebSocket Infrastructure | single authenticated WS endpoint, room subscription, command/event envelope, heartbeat, reconnect, disconnect handling | ✅ done |
| 7 | Multiplayer Gameplay | engine wired to rooms + WS; full real-time Hokm flow (hakem → trump → deal → 13 tricks → score) | ✅ done |
| 8 | Frontend | React + TypeScript (Vite): home, auth, profile, room browser, create/join, lobby, game table, results, leaderboard, history, settings | ✅ done (leaderboard/history land with phases 11/13) |
| 9 | Mobile-First Game UI | player positions, hand, trick area, trump indicator, scores, turn/team indicators, animations, touch, responsive desktop | ✅ done |
| 10 | AI Engine | PlayerStrategy interface; Easy→Pro strategies; information-set reasoning (played cards, remaining, probabilities); RL-ready | ✅ done (moved ahead of 8-9: completes the backend loop required by human+AI E2E) |
| 11 | PostgreSQL & Redis | schema + migrations (users, profiles, rooms, games, players, teams, results, stats, ratings, chat, events); Redis active rooms/state/presence/sessions/rate limits | ✅ done (core: users/refresh durable + migrations + redis rate limiting/presence; game history writes land with 13) |
| 12 | Chat & Social | room chat, system messages, presence, moderation, rate limiting | ✅ done |
| 13 | Ranking & Statistics | Elo-ready rating service, leaderboard, win rate, streaks, rank | ✅ done (streaks land with game-history replay) |
| 14 | Reconnection & Resilience | session recovery, state replay, disconnect timeout, optional AI takeover | ✅ done |
| 15 | Security & Anti-Cheat | server-authoritative validation, rate limits, secure headers, hidden-info protection, WS validation, secret management | ✅ done (built-in from phases 4-14: auth at upgrade, membership/phase/turn checks, per-seat views, input caps, rate limits, headers; fair-projection test) |
| 16 | Full Testing | unit + integration + E2E (protocol-level in Go: 4-player match, human+AI, invalid actions, reconnection) + AI simulation batches (zero illegal moves); Playwright browser E2E = follow-up | done (browser E2E pending) |
| 17 | Observability & Performance | structured logs, /api/metrics (http, ws sessions, active games, matches, AI decision time), error hooks | done |
| 18 | Production Docker | single deployable (frontend baked into go image), prod compose with required secrets, healthchecks, migrations, graceful shutdown | done |
| 19 | Documentation & Release | README, ARCHITECTURE, GAME_RULES, API, WEBSOCKET, AI, DATABASE, SECURITY, DEPLOYMENT, ADRs 0001-0012, release checklist | done |

## Game Rules (authoritative reference)

Standard Iranian Hokm, 4 players, fixed teams (2v2, partners face each other):
0. **Hakem selection**: round 1 is a uniform random seat (ADR-0012). Later
   rounds keep hakem on team win and pass to `NextSeat` on loss. Ambiguities
   are documented in `docs/GAME_RULES.md` and made configurable.
1. **Initial deal**: 5 cards to each player (starting left of Hakem).
2. **Trump**: Hakem views their 5 cards and selects trump suit.
3. **Remaining deal**: remaining 8 cards each → 13 cards per player.
4. **Play**: up to 13 tricks per round; first team to 7 tricks wins and
   remaining tricks are skipped (ADR-0012). Lead suit must be followed if possible;
   otherwise any card (including trump). Highest trump beats highest lead
   suit; otherwise highest lead-suit card wins. Trick winner leads next.
5. **Round scoring**: a team taking 7 tricks wins the round immediately;
   leftover tricks of that round are not played (ADR-0012).
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
