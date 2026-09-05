# ADR-0020

Status: Accepted
Date: 2026-09-05

Decision: Bundle gameplay SFX via Vite imports and inline them as
`data:` URLs (assetsInlineLimit). Howler must not fetch bare
`/assets/audio/*` network URLs during preload/play.

## Context

Correct MIME (`audio/ogg`, `audio/wav`) and OggS/RIFF bodies were already
served from WEB_DIR. Users still saw the download manager open instead of
hearing SFX. Mute control is a real button, not a download link. Root cause:
browser download managers (notably Internet Download Manager) intercept
XHR/fetch/Howler loads of public audio URLs and force a file download.

## Decision

1. Source of truth: `frontend/src/assets/audio/` (same filenames as before).
2. `audioConfig.ts` imports every `.ogg`/`.wav`; Vite inlines them as
   `data:audio/...;base64,...` when under `build.assetsInlineLimit` (65536).
3. `AudioManager.makeHowl` sets `format: ['ogg','wav']` (required for data:
   URLs with no extension).
4. Keep a filename-identical mirror under `frontend/public/assets/audio/`
   so assets remain replaceable by name; rebuild picks up `src/` imports.
5. Do not point Howl `src` at `/assets/audio/...` network paths.

## Alternatives rejected

- Fix only MIME/SPA fallback (already correct; does not stop IDM).
- Serve audio with Content-Disposition: inline (IDM still intercepts).
- html5: true + blob fetch (still a network request IDM can hook).
- Ask every user to exclude the site from IDM (helpful tip, not a fix).

## Consequences

- Slightly larger JS/CSS chunk (SFX are tiny: ~5–25KB each).
- Replacing SFX requires editing `src/assets/audio/` (and preferably the
  public mirror) then rebuilding.
- Public `/assets/audio/*` may still exist for manual checks but is unused
  by Howler.
