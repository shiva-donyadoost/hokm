# ADR-0015

Status: Accepted
Date: 2026-09-05

Decision: lorelei HTTP API avatars; seed user_id; PlayerAvatar initials fallback.

## Context

Players shown as username text only. Need stable faces without uploads.
## Decision
Style lorelei SVG via public HTTP API. Seed user_id else username.
PlayerAvatar initials onError. GameTable seats, Room lobby, Profile, Leaderboard.
Prefer HTTP API. 
## Alternatives rejected
alt rejected.
## Consequences
Outbound HTTPS or initials. No backend change.
