const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

const ROOT = path.resolve('E:/project/hokm');
const utf8 = (p, s) => {
  fs.mkdirSync(path.dirname(p), { recursive: true });
  fs.writeFileSync(p, s, { encoding: 'utf8' });
  console.log('wrote', path.relative(ROOT, p), Buffer.byteLength(s), 'bytes');
};

const FFMPEG = 'E:/tools/ffmpeg/ffmpeg-9.0.1-essentials_build/bin/ffmpeg.exe';

function writeWav(file, samples, sampleRate = 44100) {
  const dataSize = samples.length * 2;
  const buf = Buffer.alloc(44 + dataSize);
  buf.write('RIFF', 0);
  buf.writeUInt32LE(36 + dataSize, 4);
  buf.write('WAVE', 8);
  buf.write('fmt ', 12);
  buf.writeUInt32LE(16, 16);
  buf.writeUInt16LE(1, 20);
  buf.writeUInt16LE(1, 22);
  buf.writeUInt32LE(sampleRate, 24);
  buf.writeUInt32LE(sampleRate * 2, 28);
  buf.writeUInt16LE(2, 32);
  buf.writeUInt16LE(16, 34);
  buf.write('data', 36);
  buf.writeUInt32LE(dataSize, 40);
  for (let i = 0; i < samples.length; i++) {
    let v = Math.max(-1, Math.min(1, samples[i]));
    buf.writeInt16LE((v * 32767) | 0, 44 + i * 2);
  }
  fs.writeFileSync(file, buf);
}

function tone(freq, dur, opts = {}) {
  const sr = opts.sr || 44100;
  const n = Math.floor(sr * dur);
  const samples = new Float64Array(n);
  const vol = opts.vol ?? 0.4;
  const decay = opts.decay ?? 8;
  const noise = opts.noise ?? 0;
  for (let i = 0; i < n; i++) {
    const t = i / sr;
    const env = Math.exp(-decay * t);
    let s = Math.sin(2 * Math.PI * freq * t) * env * vol;
    if (noise) s += (Math.random() * 2 - 1) * noise * env;
    if (opts.freq2) s += Math.sin(2 * Math.PI * opts.freq2 * t) * env * (opts.vol2 ?? vol * 0.5);
    samples[i] = s;
  }
  return samples;
}

function mix(...parts) {
  let len = 0;
  for (const p of parts) len = Math.max(len, p.length);
  const out = new Float64Array(len);
  for (const p of parts) {
    for (let i = 0; i < p.length; i++) out[i] += p[i];
  }
  let peak = 0;
  for (let i = 0; i < out.length; i++) peak = Math.max(peak, Math.abs(out[i]));
  if (peak > 1) for (let i = 0; i < out.length; i++) out[i] /= peak;
  return out;
}


function generateSfx() {
  const dir = path.join(ROOT, 'frontend/public/assets/audio');
  fs.mkdirSync(dir, { recursive: true });
  const files = {
    'card-deal-01.wav': mix(tone(420, 0.08, { decay: 30, noise: 0.25, vol: 0.35 }), tone(880, 0.05, { decay: 40, vol: 0.15 })),
    'card-deal-02.wav': mix(tone(390, 0.09, { decay: 28, noise: 0.28, vol: 0.35 }), tone(760, 0.05, { decay: 38, vol: 0.14 })),
    'card-deal-03.wav': mix(tone(450, 0.07, { decay: 32, noise: 0.22, vol: 0.33 }), tone(920, 0.045, { decay: 42, vol: 0.16 })),
    'card-play-01.wav': mix(tone(180, 0.12, { decay: 18, noise: 0.35, vol: 0.45 }), tone(120, 0.08, { decay: 22, vol: 0.2 })),
    'card-play-02.wav': mix(tone(200, 0.11, { decay: 17, noise: 0.32, vol: 0.42 }), tone(140, 0.07, { decay: 20, vol: 0.18 })),
    'card-play-03.wav': mix(tone(165, 0.13, { decay: 16, noise: 0.38, vol: 0.44 }), tone(110, 0.09, { decay: 19, vol: 0.2 })),
    'hakem-selected.wav': mix(tone(523.25, 0.25, { decay: 6, vol: 0.35 }), tone(659.25, 0.25, { decay: 6, vol: 0.28, freq2: 783.99, vol2: 0.22 })),
    'trump-selected.wav': mix(tone(392, 0.2, { decay: 5, vol: 0.3, freq2: 493.88, vol2: 0.25 }), tone(587.33, 0.28, { decay: 4, vol: 0.28 })),
    'trump-cut.wav': mix(tone(110, 0.18, { decay: 10, noise: 0.2, vol: 0.5 }), tone(880, 0.15, { decay: 12, vol: 0.35 }), tone(1320, 0.12, { decay: 14, vol: 0.2 })),
    'trick-won.wav': mix(tone(523.25, 0.18, { decay: 7, vol: 0.3 }), tone(659.25, 0.22, { decay: 6, vol: 0.28 }), tone(783.99, 0.28, { decay: 5, vol: 0.25 })),
    'card-collect.wav': mix(tone(300, 0.2, { decay: 9, noise: 0.15, vol: 0.3 }), tone(240, 0.18, { decay: 10, vol: 0.22 })),
  };
  for (const [name, samples] of Object.entries(files)) {
    const wav = path.join(dir, name);
    writeWav(wav, samples);
    const ogg = wav.replace(/\.wav$/, '.ogg');
    if (fs.existsSync(FFMPEG)) {
      const r = spawnSync(FFMPEG, ['-y', '-i', wav, '-c:a', 'libvorbis', '-q:a', '4', ogg], { encoding: 'utf8' });
      if (r.status !== 0) console.warn('ffmpeg ogg failed', name, r.stderr?.slice(-200));
      else console.log('ogg', name);
    }
  }
  return Object.keys(files);
}


const AUDIO_CONFIG = `import { ANIM } from '../config'

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
      src: [\`\${A}/card-deal-01.ogg\`, \`\${A}/card-deal-01.wav\`],
      variants: [
        [\`\${A}/card-deal-01.ogg\`, \`\${A}/card-deal-01.wav\`],
        [\`\${A}/card-deal-02.ogg\`, \`\${A}/card-deal-02.wav\`],
        [\`\${A}/card-deal-03.ogg\`, \`\${A}/card-deal-03.wav\`],
      ],
    },
    cardPlay: {
      volume: 0.55,
      src: [\`\${A}/card-play-01.ogg\`, \`\${A}/card-play-01.wav\`],
      variants: [
        [\`\${A}/card-play-01.ogg\`, \`\${A}/card-play-01.wav\`],
        [\`\${A}/card-play-02.ogg\`, \`\${A}/card-play-02.wav\`],
        [\`\${A}/card-play-03.ogg\`, \`\${A}/card-play-03.wav\`],
      ],
    },
    hakemSelected: {
      volume: 0.5,
      src: [\`\${A}/hakem-selected.ogg\`, \`\${A}/hakem-selected.wav\`],
    },
    trumpSelected: {
      volume: 0.5,
      src: [\`\${A}/trump-selected.ogg\`, \`\${A}/trump-selected.wav\`],
    },
    trumpCut: {
      volume: 0.65,
      src: [\`\${A}/trump-cut.ogg\`, \`\${A}/trump-cut.wav\`],
    },
    trickWon: {
      volume: 0.55,
      src: [\`\${A}/trick-won.ogg\`, \`\${A}/trick-won.wav\`],
    },
    cardCollect: {
      volume: 0.45,
      src: [\`\${A}/card-collect.ogg\`, \`\${A}/card-collect.wav\`],
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
`;


const AUDIO_MANAGER = `import { Howl, Howler } from 'howler'
import { AUDIO, type SoundId, type SoundDef } from './audioConfig'
import { diagWarn, diagInfo } from '../diagnostics/clientLog'

// AudioManager is the ONLY Howler entry point (ADR-0019). Presentation code
// calls play/stop/setMuted; it never constructs Howl itself.

type HowlBag = { howl: Howl; variants?: Howl[] }

class AudioManagerImpl {
  private sounds = new Map<SoundId, HowlBag>()
  private unlocked = false
  private muted = false
  private master = AUDIO.masterVolume
  private ready = false
  private unlockBound = false

  /** Idempotent preload; safe to call before any gesture. */
  preload(): void {
    if (this.ready) return
    try {
      this.muted = this.readMute()
      Howler.mute(this.muted)
      Howler.volume(this.master)
      for (const id of Object.keys(AUDIO.sounds) as SoundId[]) {
        this.ensure(id)
      }
      this.ready = true
      this.bindUnlock()
      diagInfo('audio', 'preloaded', { muted: this.muted })
    } catch (err) {
      diagWarn('audio', 'preload_failed', { err: String(err) })
    }
  }

  private ensure(id: SoundId): HowlBag | null {
    const existing = this.sounds.get(id)
    if (existing) return existing
    const def = AUDIO.sounds[id] as SoundDef
    try {
      const howl = this.makeHowl(def.src, def.volume)
      let variants: Howl[] | undefined
      if (def.variants && def.variants.length > 0) {
        variants = def.variants.map((src) => this.makeHowl(src, def.volume))
      }
      const bag = { howl, variants }
      this.sounds.set(id, bag)
      return bag
    } catch (err) {
      diagWarn('audio', 'howl_create_failed', { id, err: String(err) })
      return null
    }
  }

  private makeHowl(src: string[], volume: number): Howl {
    return new Howl({
      src,
      volume: volume * this.master,
      preload: true,
      html5: false,
      onloaderror: (_id, err) => {
        diagWarn('audio', 'load_error', { src: src[0], err: String(err) })
      },
      onplayerror: (_id, err) => {
        diagWarn('audio', 'play_error', { src: src[0], err: String(err) })
        // Mobile autoplay: try unlock then replay once.
        this.unlock()
      },
    })
  }

  play(id: SoundId): void {
    try {
      if (!this.ready) this.preload()
      if (this.muted) return
      const bag = this.ensure(id)
      if (!bag) return
      const pool = bag.variants && bag.variants.length > 0 ? bag.variants : [bag.howl]
      const pick = pool[Math.floor(Math.random() * pool.length)] ?? bag.howl
      pick.volume((AUDIO.sounds[id].volume) * this.master)
      pick.play()
    } catch (err) {
      diagWarn('audio', 'play_failed', { id, err: String(err) })
    }
  }

  stop(id?: SoundId): void {
    try {
      if (id) {
        const bag = this.sounds.get(id)
        bag?.howl.stop()
        bag?.variants?.forEach((h) => h.stop())
        return
      }
      for (const bag of this.sounds.values()) {
        bag.howl.stop()
        bag.variants?.forEach((h) => h.stop())
      }
    } catch (err) {
      diagWarn('audio', 'stop_failed', { id, err: String(err) })
    }
  }

  setMasterVolume(v: number): void {
    this.master = Math.max(0, Math.min(1, v))
    try {
      Howler.volume(this.master)
    } catch {
      /* ignore */
    }
  }

  getMasterVolume(): number {
    return this.master
  }

  setMuted(muted: boolean): void {
    this.muted = muted
    try {
      Howler.mute(muted)
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem(AUDIO.muteStorageKey, muted ? '1' : '0')
      }
    } catch (err) {
      diagWarn('audio', 'mute_failed', { err: String(err) })
    }
  }

  isMuted(): boolean {
    return this.muted
  }

  toggleMute(): boolean {
    this.setMuted(!this.muted)
    return this.muted
  }

  /** Resume AudioContext after a user gesture (iOS/Android autoplay). */
  unlock(): void {
    if (this.unlocked) return
    try {
      Howler.ctx?.resume?.()
      // Silent unlock buffer via Howler internal path.
      const silent = new Howl({ src: ['data:audio/wav;base64,UklGRiQAAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQAAAAA='], volume: 0 })
      silent.play()
      silent.once('play', () => {
        silent.stop()
        silent.unload()
      })
      this.unlocked = true
      diagInfo('audio', 'unlocked')
    } catch (err) {
      diagWarn('audio', 'unlock_failed', { err: String(err) })
    }
  }

  private bindUnlock(): void {
    if (this.unlockBound || typeof window === 'undefined') return
    this.unlockBound = true
    const once = () => {
      this.unlock()
      window.removeEventListener('pointerdown', once)
      window.removeEventListener('keydown', once)
      window.removeEventListener('touchstart', once)
    }
    window.addEventListener('pointerdown', once, { passive: true })
    window.addEventListener('keydown', once)
    window.addEventListener('touchstart', once, { passive: true })
  }

  private readMute(): boolean {
    try {
      if (typeof localStorage === 'undefined') return false
      return localStorage.getItem(AUDIO.muteStorageKey) === '1'
    } catch {
      return false
    }
  }
}

export const AudioManager = new AudioManagerImpl()
`;


const MAP_EVENTS = `import type { CompletedTrick, Suit } from '../protocol/messages'

// Pure mapping from authoritative engine/WS data to presentation audio
// events (ADR-0019). No Howler imports here - keep this unit-testable.

export type PresentationAudioEvent =
  | { type: 'CARD_DEALT'; index: number; total: number }
  | { type: 'CARD_PLAYED'; seat: number }
  | { type: 'HAKEM_SELECTED'; seat: number }
  | { type: 'TRUMP_SELECTED'; suit: string; automatic?: boolean }
  | { type: 'TRUMP_CUT'; seat: number }
  | { type: 'TRICK_WON'; seat: number }
  | { type: 'CARD_COLLECT'; seat: number }

export interface EngineEventLite {
  name: string
  payload: unknown
}

/** True when a non-trump lead was won by a trump card (real cut-win). */
export function isTrumpCutWin(trick: CompletedTrick, trump?: Suit | null): boolean {
  if (!trump) return false
  if (trick.lead_suit === trump) return false
  const winnerPlay = trick.cards.find((c) => c.seat === trick.winner)
  if (!winnerPlay) return false
  return winnerPlay.card.suit === trump
}

/**
 * Map one public engine event (WS EVENTS name + payload) to zero or more
 * presentation events. trick_completed yields TRUMP_CUT or TRICK_WON;
 * CARD_COLLECT is scheduled by the hook using ANIM timing, not here.
 */
export function mapEngineEvent(ev: EngineEventLite): PresentationAudioEvent[] {
  const name = ev.name
  const p = (ev.payload ?? {}) as Record<string, unknown>
  switch (name) {
    case 'hakem_selected':
      return [{ type: 'HAKEM_SELECTED', seat: Number(p.seat ?? 0) }]
    case 'next_round_started':
      return [{ type: 'HAKEM_SELECTED', seat: Number(p.hakem ?? 0) }]
    case 'trump_selected':
      return [{
        type: 'TRUMP_SELECTED',
        suit: String(p.suit ?? ''),
        automatic: Boolean(p.automatic),
      }]
    case 'card_played':
      return [{ type: 'CARD_PLAYED', seat: Number(p.seat ?? 0) }]
    case 'trick_completed': {
      const trick = (p.trick ?? p) as CompletedTrick
      if (!trick || typeof trick.winner !== 'number' || !trick.cards) return []
      // trump may be attached by the hook; payload alone has lead_suit + cards.
      return [{ type: 'TRICK_WON', seat: trick.winner }]
    }
    default:
      return []
  }
}

/** Resolve trick_completed into TRUMP_CUT or TRICK_WON using view.trump. */
export function mapTrickCompleted(
  trick: CompletedTrick,
  trump?: Suit | null,
): PresentationAudioEvent {
  if (isTrumpCutWin(trick, trump)) {
    return { type: 'TRUMP_CUT', seat: trick.winner }
  }
  return { type: 'TRICK_WON', seat: trick.winner }
}

/** Hand-length jumps that match the existing deal animation triggers. */
export function dealCountsForHandJump(prevLen: number, nextLen: number): number {
  if (prevLen === 0 && nextLen === 5) return 5
  if (prevLen === 5 && nextLen === 13) return 8
  return 0
}
`;

const PRESENTATION_BUS = `import type { PresentationAudioEvent } from './mapEngineEvents'

type Listener = (ev: PresentationAudioEvent) => void

// Thin in-process bus so GameTable/animations can emit without importing Howler.
const listeners = new Set<Listener>()

export function subscribePresentationAudio(fn: Listener): () => void {
  listeners.add(fn)
  return () => { listeners.delete(fn) }
}

export function emitPresentationAudio(ev: PresentationAudioEvent): void {
  for (const fn of listeners) {
    try {
      fn(ev)
    } catch {
      // never break game loop
    }
  }
}
`;


const USE_GAME_AUDIO = `import { useEffect, useRef } from 'react'
import { AudioManager } from './AudioManager'
import { AUDIO } from './audioConfig'
import {
  dealCountsForHandJump,
  mapEngineEvent,
  mapTrickCompleted,
  type PresentationAudioEvent,
} from './mapEngineEvents'
import { emitPresentationAudio, subscribePresentationAudio } from './presentationEvents'
import type { CompletedTrick, SeatView } from '../protocol/messages'
import { animDuration } from '../config'

export interface EngineEventState {
  name: string
  payload: unknown
  seq: number
}

/**
 * Wires authoritative SeatView + WS EVENTS into AudioManager.
 * Mount once under GameTable. Audio failures never throw into React.
 */
export function useGameAudio(view: SeatView | null, engineEvent: EngineEventState | null): void {
  const prevHand = useRef(0)
  const lastSeq = useRef(0)
  const timers = useRef<ReturnType<typeof setTimeout>[]>([])

  const clearTimers = () => {
    for (const t of timers.current) clearTimeout(t)
    timers.current = []
  }

  const schedule = (ms: number, fn: () => void) => {
    const t = setTimeout(fn, ms)
    timers.current.push(t)
  }

  useEffect(() => {
    AudioManager.preload()
    const unsub = subscribePresentationAudio((ev) => {
      playForEvent(ev, schedule)
    })
    return () => {
      unsub()
      clearTimers()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Hand-length deal inference (private deal events are not on the public bus).
  useEffect(() => {
    if (!view) return
    const len = view.your_hand?.length ?? 0
    const n = dealCountsForHandJump(prevHand.current, len)
    if (n > 0) {
      for (let i = 0; i < n; i++) {
        const idx = i
        schedule(i * AUDIO.timing.dealStaggerMs, () => {
          emitPresentationAudio({ type: 'CARD_DEALT', index: idx, total: n })
        })
      }
    }
    prevHand.current = len
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view?.your_hand?.length])

  // WS EVENTS mapping.
  useEffect(() => {
    if (!engineEvent || engineEvent.seq === lastSeq.current) return
    lastSeq.current = engineEvent.seq
    const name = engineEvent.name
    const payload = engineEvent.payload as Record<string, unknown>

    if (name === 'trick_completed') {
      const trick = (payload.trick ?? payload) as CompletedTrick
      if (trick && typeof trick.winner === 'number') {
        const mapped = mapTrickCompleted(trick, view?.trump)
        schedule(AUDIO.timing.trickWonDelayMs, () => emitPresentationAudio(mapped))
        schedule(AUDIO.timing.cardCollectDelayMs, () => {
          emitPresentationAudio({ type: 'CARD_COLLECT', seat: trick.winner })
        })
      }
      return
    }

    for (const ev of mapEngineEvent({ name, payload })) {
      if (ev.type === 'CARD_PLAYED') {
        schedule(animDuration(AUDIO.timing.cardPlayImpactDelayMs), () => emitPresentationAudio(ev))
      } else {
        emitPresentationAudio(ev)
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [engineEvent?.seq])
}

function playForEvent(
  ev: PresentationAudioEvent,
  _schedule: (ms: number, fn: () => void) => void,
): void {
  switch (ev.type) {
    case 'CARD_DEALT':
      AudioManager.play('cardDeal')
      break
    case 'CARD_PLAYED':
      AudioManager.play('cardPlay')
      break
    case 'HAKEM_SELECTED':
      AudioManager.play('hakemSelected')
      break
    case 'TRUMP_SELECTED':
      AudioManager.play('trumpSelected')
      break
    case 'TRUMP_CUT':
      AudioManager.play('trumpCut')
      break
    case 'TRICK_WON':
      AudioManager.play('trickWon')
      break
    case 'CARD_COLLECT':
      AudioManager.play('cardCollect')
      break
    default:
      break
  }
}
`;

const MUTE_BUTTON = `import { useEffect, useState } from 'react'
import { AudioManager } from '../audio/AudioManager'

/** Small mute toggle; mute is localStorage-only (never sent to server). */
export function AudioMuteButton({ className = '' }: { className?: string }) {
  const [muted, setMuted] = useState(false)
  useEffect(() => {
    AudioManager.preload()
    setMuted(AudioManager.isMuted())
  }, [])
  return (
    <button
      type="button"
      className={\`px-2 py-1 rounded-full border text-xs font-semibold shadow \${
        muted
          ? 'bg-slate-800 border-slate-600 text-slate-400'
          : 'bg-slate-900/90 border-slate-500 text-slate-100'
      } \${className}\`}
      aria-pressed={muted}
      aria-label={muted ? 'unmute sound' : 'mute sound'}
      onClick={() => {
        AudioManager.unlock()
        setMuted(AudioManager.toggleMute())
      }}
    >
      {muted ? 'Sound off' : 'Sound on'}
    </button>
  )
}
`;



function main() {
  generateSfx()
  const path = require("path")
  const audioDir = path.join(ROOT, "frontend/src/audio")
  fs.mkdirSync(audioDir, { recursive: true })
  utf8(path.join(audioDir, "audioConfig.ts"), AUDIO_CONFIG)
  utf8(path.join(audioDir, "AudioManager.ts"), AUDIO_MANAGER)
  utf8(path.join(audioDir, "mapEngineEvents.ts"), MAP_EVENTS)
  utf8(path.join(audioDir, "presentationEvents.ts"), PRESENTATION_BUS)
  utf8(path.join(audioDir, "useGameAudio.ts"), USE_GAME_AUDIO)
  utf8(path.join(ROOT, "frontend/src/components/AudioMuteButton.tsx"), MUTE_BUTTON)
  console.log("core sources written")
}
main()

