# Security

## Server authority (ADR-0004)

The browser is untrusted. Every gameplay command passes: authentication →
room membership → phase → turn → card ownership → rule legality. Clients
render; they never compute.

## Hidden-information protection

- `game.SeatView` is the only per-recipient state projection; it contains
  the recipient's own hand and others' hand counts only.
- Private card-delivery events are filtered before broadcast
  (`publicEvents`); only completed-trick and public events cross the wire.
- AI reasoning uses `ai.BuildInformationSet` — provably free of other
  hands (`TestInformationSetFairness`).

## Authentication (ADR-0008)

- bcrypt (cost 12) password hashing; 8–128 char policy.
- JWT HS256 access tokens (30-day default, ADR-0012), issuer-checked.
- Refresh tokens: 256-bit random, stored **hashed** (SHA-256), single-use
  with rotation; expired/unknown/reused → 401.
- Login responses are identical for unknown user vs wrong password (no
  enumeration).

## Transport

- WS upgrade authenticates the token; 64 KiB read limit per message.
- Requests body-capped (1 MiB) via `http.MaxBytesReader`.
- Security headers: `X-Content-Type-Options: nosniff`,
  `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`.
- Panic recovery converts handler panics into 500s (stack logged).

## Rate limiting

- Redis fixed-window counters per client IP on auth endpoints and the WS
  handshake (default 60/min), and per-user on chat (5/10 s).
- Fail-open on Redis errors (availability-first; documented trade-off).

## Secrets

- Configuration only via environment; `.env` gitignored, `.env.example`
  documents variables.
- `APP_ENV=production` refuses to start with a JWT secret shorter than 32
  characters; the dev fallback secret is rejected in production.

## Anti-cheat status

- Illegal moves are impossible by construction (engine rejects and the
  simulator/property tests enforce termination + legality).
- Score forging is impossible (server-computed only).
- Known limitation: collusion between partners is part of the game; no
  behavioral detection yet (future work).
