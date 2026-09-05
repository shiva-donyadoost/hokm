import { Howl, Howler } from 'howler'
import { AUDIO, type SoundId, type SoundDef } from './audioConfig'
import { diagWarn, diagInfo } from '../diagnostics/clientLog'

// AudioManager is the ONLY Howler entry point (ADR-0019). Presentation code
// calls play/stop/setMuted; it never constructs Howl itself.

type HowlBag = { howl: Howl; variants?: Howl[] }

class AudioManagerImpl {
  private sounds = new Map<SoundId, HowlBag>()
  private unlocked = false
  private muted = false
  private master: number = AUDIO.masterVolume
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
