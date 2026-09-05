import type { CompletedTrick, Suit } from '../protocol/messages'

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
