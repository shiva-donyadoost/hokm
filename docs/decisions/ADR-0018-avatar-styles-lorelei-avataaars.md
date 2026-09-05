# ADR-0018

Status: Accepted
Date: 2026-09-05

Decision: Persist DiceBear avatar_style alongside avatar_seed; host both
lorelei and avataaars via the same DiceBear 9.x HTTP CDN (extends ADR-0017).

## Context

ADR-0017 added a curated seed gallery but locked the visual style to lorelei.
Players want more face variety. DiceBear already serves the avataaars style at
https://api.dicebear.com/9.x/avataaars/svg?seed=..., so a second CDN or
npm package is unnecessary.

## Decision

1. Add nullable users.avatar_style (TEXT). Allowed values: lorelei,
   avataaars. Empty/null with a non-empty seed means legacy ADR-0017 rows
   and defaults to lorelei. Empty seed keeps ADR-0015 user_id/username
   fallback (style defaults to lorelei for the URL).
2. Keep the same 18 curated seeds for both styles (URLs differ by style path).
   Server whitelist validates (style, seed) together; untrusted clients cannot
   invent styles or seeds.
3. Register and PATCH /api/me accept avatar_style + avatar_seed. UI
   requires a pick; omitting style while sending a seed defaults to lorelei
   for backward-compatible clients.
4. Room Member, WS ROOM snapshots, and leaderboard Entry carry both fields so
   peers render the correct face. UpdateMemberAvatar refreshes style+seed.
5. Frontend AvatarPicker shows two labeled groups (Lorelei / Avataaars).
   PlayerAvatar / dicebearUrl use the stored style.

## Alternatives rejected

- Separate Avataaars npm/CDN: extra dependency and operational surface; DiceBear
  already hosts the artwork.
- Free-form style strings: same abuse surface as free-form seeds (H3).
- Encode style:seed in a single column only: workable, but separate columns
  keep SQL/leaderboard scans and legacy seed rows simpler.
- Drop lorelei: breaks ADR-0015/0017 visual continuity for existing users.

## Consequences

- Migration 0006 adds users.avatar_style.
- Legacy rows (seed only) keep looking like lorelei until the user picks again.
- Frontend and backend seed lists stay in sync; styles are a shared enum.
- Supersedes the lorelei-only constraint of ADR-0017 while keeping its
  whitelist and persistence model.
