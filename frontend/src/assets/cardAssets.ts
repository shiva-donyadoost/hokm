// Card asset resolver + build-time validation (impliment.md Â§34â€“Â§36).
// One mapping layer: Card â†’ SVG asset. No file paths in components.

const RANK_FILE: Record<number, string> = {
  2: '2', 3: '3', 4: '4', 5: '5', 6: '6', 7: '7', 8: '8', 9: '9',
  10: '10', 11: 'jack', 12: 'queen', 13: 'king', 14: 'ace',
}

const SUIT_FILE: Record<string, string> = {
  spades: 'spades',
  hearts: 'hearts',
  diamonds: 'diamonds',
  clubs: 'clubs',
}

export const CARD_BACK_ASSET = '/cards/card-back.svg'
export const CARD_ASSET_BASE = '/cards'

// cardFaceAsset resolves a card to its SVG URL.
export function cardFaceAsset(card: { suit: string; rank: number }): string {
  const r = RANK_FILE[card.rank]
  const s = SUIT_FILE[card.suit]
  if (!r || !s) return CARD_BACK_ASSET // unknown card â†’ never broken image
  return `${CARD_ASSET_BASE}/${r}_of_${s}.svg`
}

// REQUIRED_ASSETS is the canonical list used by validation.
export function requiredAssets(): string[] {
  const out = ['card-back.svg']
  for (const s of Object.values(SUIT_FILE)) {
    for (const r of Object.values(RANK_FILE)) {
      out.push(`${r}_of_${s}.svg`)
    }
  }
  return out
}
