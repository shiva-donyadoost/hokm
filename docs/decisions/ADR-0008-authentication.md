# ADR-0008: Authentication

Date: 2026-08-29
Status: Accepted

## Context

We need stateless auth for REST + long-lived WebSocket connections, plus
session revocation and refresh flows suitable for mobile and web clients.

## Decision

- **JWT access token** (default TTL **30 days** / `720h` per ADR-0012;
  was 15 min; HS256 or RS256 via env config) carried in
  `Authorization: Bearer` and as a one-time ticket query param for the WS
  upgrade (`/api/ws?ticket=<short-lived ws ticket>`); the ticket is single-
  use and bound to the user id to avoid long-lived tokens in URLs.
- **Refresh token**: opaque 256-bit random, stored hashed (SHA-256) in
  Postgres with rotation + reuse detection; revoked on logout; denylist
  cache in Redis for fast checks.
- **Passwords**: bcrypt cost 12. Registration validates username/email/
  password policy server-side; all inputs size-limited.
- Middleware: `RequireAuth` for REST; WS session authenticated once at
  upgrade, user id attached to the session for its lifetime.
- Guests (AI-only tables) get ephemeral accounts marked `is_guest`, no
  password, upgradeable later.

## Alternatives considered

- **Server sessions only (cookie)**: simplest, but breaks for the WS ticket
  flow across origins and mobile clients; hybrid chosen instead.
- **OAuth-only**: defer; local accounts first, OAuth (Google) as a Phase 4+
  addition using the same user table.
- **Pure opaque tokens with Redis lookup**: adds a Redis round-trip per
  request; JWT avoids it while refresh+denylist covers revocation.

## Consequences

- Clock skew and token expiry must be handled client-side (refresh loop).
- Secret management: JWT secret from env (`.env`, never committed);
  production uses RS256 key pair mounted as files.
