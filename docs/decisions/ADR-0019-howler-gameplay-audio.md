 ADR-0019

Status: Accepted
Date: 2026-09-05

Decision: Centralize gameplay SFX behind Howler AudioManager;
map authoritative engine/WS transitions to presentation-only audio.
Audio never affects legality or server state (H3 / ADR-0004).

## Context

Table animates deal/play/reveal/collect from SeatView + ANIM.
Need SFX timed to those moments without scattering Howler in UI.
Engine events exist; WS EVENTS carries public kinds. Private deal
events are stripped by publicEvents. No engine trump_cut; infer cut
when lead is non-trump and winner card is trump.

## Decision

1. Howler only via AudioManager (frontend/src/audio/AudioManager.ts).
2. Config paths/volumes/delays in audioConfig.ts (reuse ANIM).
3. Assets at frontend/public/assets/audio/ (Vite + WEB_DIR).
4. useGameAudio maps store + presentation bus to AudioManager.
5. game.ts handles Msg.Events as lastEngineEvent (seq).
6. CARD_DEALT from your_hand jumps 0->5 and 5->13.
7. TRUMP_CUT inferred from authoritative trick + trump.
8. Failures logged; mute is localStorage-only.
9. Synthetic wav+ogg SFX; keep filenames on replace.
10. scripts/check-audio-mapping.mjs regression tests.

## Alternatives rejected

- Scatter Howl in components.
- Engine EventTrumpCut.
- Public stripped deal events.
- HTML5-only audio.
- Repo-embedded reference video.

## Consequences

- Adds howler dependency.
- Asset validation includes SFX filenames.
- Play SFX delayed to card land timing.

## Supersession

Asset delivery amended by ADR-0020 (Vite-inlined data: URLs; no Howler
network fetches of /assets/audio/*). Mapping/AudioManager rules above still apply.
