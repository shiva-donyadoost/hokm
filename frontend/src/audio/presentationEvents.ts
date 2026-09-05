import type { PresentationAudioEvent } from './mapEngineEvents'

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
