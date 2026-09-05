# ADR-0017

Status: Accepted
Date: 2026-09-05

Decision: Persist a curated DiceBear avatar_seed on the user; keep lorelei;
fallback to user_id/username for legacy rows (extends ADR-0015).

## Context

ADR-0015 made faces deterministic from user_id, so players could not choose
an avatar at registration or change it later. Product now requires a
selectable gallery at signup and on Profile, with the choice visible on
table seats, lobby, leaderboard, and profile.

## Decision

1. Store avatar_seed TEXT on users (nullable). Empty/null means legacy
   fallback: seed = user_id else username (ADR-0015 behavior).
2. Keep DiceBear style lorelei / 9.x HTTP API (do not change style).
3. Curate a fixed whitelist of fun seeds (18). Client and server share the
   same list; server rejects any seed not on the whitelist (untrusted client).
4. Register accepts optional avatar_seed; if omitted/invalid, leave empty
   (fallback). Prefer requiring selection in the UI.
5. PATCH /api/me { avatar_seed } updates the profile; optimistic UI OK.
6. Room Member and leaderboard Entry include avatar_seed so peers see the
   chosen face over WS ROOM snapshots and REST.
7. On profile change, Manager.UpdateMemberAvatar refreshes seated members
   and notifies subscribers.

## Alternatives rejected

- Free-form seed strings: XSS/abuse surface and inconsistent gallery.
- Upload/custom images: storage, moderation, CDN complexity.
- Change DiceBear style: breaks ADR-0015 visual continuity.
- Client-only seed in localStorage: other players would not see it;
  violates server authority (H3).

## Consequences

- Migration 0005 adds users.avatar_seed.
- Existing users keep current faces until they pick one in Profile.
- Frontend AvatarPicker reused on Register and Profile.
- AI seats keep empty avatar_seed and fall back to ai user_id seed.
