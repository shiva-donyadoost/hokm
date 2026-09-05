import { useEffect, useRef } from 'react'
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
