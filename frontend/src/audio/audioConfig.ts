import { ANIM } from '../config'

// Presentation-only audio configuration (ADR-0019). Paths are Vite/public
// URLs that also work when the build is served from WEB_DIR in Docker.

export type SoundId =
  | 'cardDeal'
  | 'cardPlay'
  | 'hakemSelected'
  | 'trumpSelected'
  | 'trumpCut'
  | 'trickWon'
  | 'cardCollect'

export interface SoundDef {
  /** Howler src list; ogg first, wav fallback. */
  src: string[]
  volume: number
  /** When true, AudioManager picks a random Howl from the variant list. */
  variants?: string[][]
}

const A = '/assets/audio'

export const AUDIO = {
  masterVolume: 0.75,
  muteStorageKey: 'hokm.audio.muted',
  sounds: {
    cardDeal: {
      volume: 0.4,
      src: [`${A}/card-deal-01.ogg`, `${A}/card-deal-01.wav`],
      variants: [
        [`${A}/card-deal-01.ogg`, `${A}/card-deal-01.wav`],
        [`${A}/card-deal-02.ogg`, `${A}/card-deal-02.wav`],
        [`${A}/card-deal-03.ogg`, `${A}/card-deal-03.wav`],
      ],
    },
    cardPlay: {
      volume: 0.55,
      src: [`${A}/card-play-01.ogg`, `${A}/card-play-01.wav`],
      variants: [
        [`${A}/card-play-01.ogg`, `${A}/card-play-01.wav`],
        [`${A}/card-play-02.ogg`, `${A}/card-play-02.wav`],
        [`${A}/card-play-03.ogg`, `${A}/card-play-03.wav`],
      ],
    },
    hakemSelected: {
      volume: 0.5,
      src: [`${A}/hakem-selected.ogg`, `${A}/hakem-selected.wav`],
    },
    trumpSelected: {
      volume: 0.5,
      src: [`${A}/trump-selected.ogg`, `${A}/trump-selected.wav`],
    },
    trumpCut: {
      volume: 0.65,
      src: [`${A}/trump-cut.ogg`, `${A}/trump-cut.wav`],
    },
    trickWon: {
      volume: 0.55,
      src: [`${A}/trick-won.ogg`, `${A}/trick-won.wav`],
    },
    cardCollect: {
      volume: 0.45,
      src: [`${A}/card-collect.ogg`, `${A}/card-collect.wav`],
    },
  } satisfies Record<SoundId, SoundDef>,

  /** Delays relative to authoritative presentation moments. */
  timing: {
    /** Play card SFX when the flying card reaches the table (matches ANIM.cardPlayMs). */
    cardPlayImpactDelayMs: ANIM.cardPlayMs,
    /** Stagger between deal one-shots (matches deal animation). */
    dealStaggerMs: ANIM.dealStaggerMs,
    /** Trick-won / trump-cut fires at reveal start. */
    trickWonDelayMs: 0,
    /** Collect SFX when cards start flying to the winner. */
    cardCollectDelayMs: ANIM.trickWinnerMs,
  },
} as const

/** Required production filenames (replace anytime; keep these names). */
export const AUDIO_ASSET_FILES = [
  'card-deal-01.wav', 'card-deal-01.ogg',
  'card-deal-02.wav', 'card-deal-02.ogg',
  'card-deal-03.wav', 'card-deal-03.ogg',
  'card-play-01.wav', 'card-play-01.ogg',
  'card-play-02.wav', 'card-play-02.ogg',
  'card-play-03.wav', 'card-play-03.ogg',
  'hakem-selected.wav', 'hakem-selected.ogg',
  'trump-selected.wav', 'trump-selected.ogg',
  'trump-cut.wav', 'trump-cut.ogg',
  'trick-won.wav', 'trick-won.ogg',
  'card-collect.wav', 'card-collect.ogg',
] as const
