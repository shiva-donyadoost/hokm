import { ANIM } from '../config'

// Presentation-only audio configuration (ADR-0019 / ADR-0020).
// SFX are Vite-imported from src/assets/audio and inlined as data: URLs
// at build time so Howler never performs network fetches that download
// managers (IDM, etc.) can intercept.

import cardDeal01Ogg from '../assets/audio/card-deal-01.ogg'
import cardDeal01Wav from '../assets/audio/card-deal-01.wav'
import cardDeal02Ogg from '../assets/audio/card-deal-02.ogg'
import cardDeal02Wav from '../assets/audio/card-deal-02.wav'
import cardDeal03Ogg from '../assets/audio/card-deal-03.ogg'
import cardDeal03Wav from '../assets/audio/card-deal-03.wav'
import cardPlay01Ogg from '../assets/audio/card-play-01.ogg'
import cardPlay01Wav from '../assets/audio/card-play-01.wav'
import cardPlay02Ogg from '../assets/audio/card-play-02.ogg'
import cardPlay02Wav from '../assets/audio/card-play-02.wav'
import cardPlay03Ogg from '../assets/audio/card-play-03.ogg'
import cardPlay03Wav from '../assets/audio/card-play-03.wav'
import hakemSelectedOgg from '../assets/audio/hakem-selected.ogg'
import hakemSelectedWav from '../assets/audio/hakem-selected.wav'
import trumpSelectedOgg from '../assets/audio/trump-selected.ogg'
import trumpSelectedWav from '../assets/audio/trump-selected.wav'
import trumpCutOgg from '../assets/audio/trump-cut.ogg'
import trumpCutWav from '../assets/audio/trump-cut.wav'
import trickWonOgg from '../assets/audio/trick-won.ogg'
import trickWonWav from '../assets/audio/trick-won.wav'
import cardCollectOgg from '../assets/audio/card-collect.ogg'
import cardCollectWav from '../assets/audio/card-collect.wav'

export type SoundId =
  | 'cardDeal'
  | 'cardPlay'
  | 'hakemSelected'
  | 'trumpSelected'
  | 'trumpCut'
  | 'trickWon'
  | 'cardCollect'

export interface SoundDef {
  /** Howler src list; ogg first, wav fallback (data: or blob: only). */
  src: string[]
  volume: number
  /** When true, AudioManager picks a random Howl from the variant list. */
  variants?: string[][]
}

const deal01 = [cardDeal01Ogg, cardDeal01Wav]
const deal02 = [cardDeal02Ogg, cardDeal02Wav]
const deal03 = [cardDeal03Ogg, cardDeal03Wav]
const play01 = [cardPlay01Ogg, cardPlay01Wav]
const play02 = [cardPlay02Ogg, cardPlay02Wav]
const play03 = [cardPlay03Ogg, cardPlay03Wav]

export const AUDIO = {
  masterVolume: 0.75,
  muteStorageKey: 'hokm.audio.muted',
  sounds: {
    cardDeal: {
      volume: 0.4,
      src: deal01,
      variants: [deal01, deal02, deal03],
    },
    cardPlay: {
      volume: 0.55,
      src: play01,
      variants: [play01, play02, play03],
    },
    hakemSelected: {
      volume: 0.5,
      src: [hakemSelectedOgg, hakemSelectedWav],
    },
    trumpSelected: {
      volume: 0.5,
      src: [trumpSelectedOgg, trumpSelectedWav],
    },
    trumpCut: {
      volume: 0.65,
      src: [trumpCutOgg, trumpCutWav],
    },
    trickWon: {
      volume: 0.55,
      src: [trickWonOgg, trickWonWav],
    },
    cardCollect: {
      volume: 0.45,
      src: [cardCollectOgg, cardCollectWav],
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

/**
 * Required filenames under frontend/src/assets/audio/ (and mirrored under
 * frontend/public/assets/audio/ for replace-in-place workflows).
 * Replace anytime; keep these names; rebuild to pick up changes.
 */
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
