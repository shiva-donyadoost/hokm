// Pure client-side playability helpers (mirror server legalCardsFor).
// Kept free of React so they can be regression-tested (ADR-0014).

import type { Card, Suit } from '../protocol/messages'

export function cardKey(c: { suit: Suit; rank: number }): string {
  return c.suit + c.rank
}

/** Lead suit the local player must follow, or null if any card is legal. */
export function requiredLeadSuit(
  hand: Card[] | undefined,
  trick: { card: Card }[] | undefined,
): Suit | null {
  if (!trick || trick.length === 0 || !hand || hand.length === 0) return null
  const first = trick[0]
  if (!first) return null
  const lead = first.card.suit
  const hasLead = hand.some((c) => c.suit === lead)
  return hasLead ? lead : null
}

export function isMyTurn(opts: {
  phase: string
  turn: number
  you: number
  trickLen: number
  matchOver: boolean
}): boolean {
  return (
    opts.phase === 'trick_play' &&
    opts.turn === opts.you &&
    opts.trickLen < 4 &&
    !opts.matchOver
  )
}

export function isCardLegal(
  card: { suit: Suit },
  opts: { myTurn: boolean; leadSuit: Suit | null },
): boolean {
  if (!opts.myTurn) return false
  return opts.leadSuit === null || card.suit === opts.leadSuit
}

export function legalCards(hand: Card[], opts: { myTurn: boolean; leadSuit: Suit | null }): Card[] {
  if (!opts.myTurn) return []
  return hand.filter((c) => isCardLegal(c, opts))
}
