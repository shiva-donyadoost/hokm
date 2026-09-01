import type { CSSProperties } from 'react'
import { cardFaceAsset, CARD_BACK_ASSET } from '../assets/cardAssets'
import { rankLabel, type Card as CardT } from '../protocol/messages'

interface CardProps {
  card: CardT
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  dimmed?: boolean
  selected?: boolean
  onClick?: () => void
  style?: CSSProperties
  className?: string
}

const SIZES = {
  sm: { w: 36, h: 50 },
  md: { w: 52, h: 73 },
  lg: { w: 66, h: 92 },
}

// Card renders the SVG asset for a face-up card. Face data (rank/suit text)
// is kept in aria-labels for accessibility (section 48) - the SVG is the artwork.
export function Card({ card, size = 'md', disabled, dimmed, selected, onClick, style, className }: CardProps) {
  const dim = SIZES[size]
  return (
    <button
      type="button"
      onClick={disabled ? undefined : onClick}
      aria-disabled={disabled}
      style={{ width: dim.w, height: dim.h, pointerEvents: disabled ? 'none' : 'auto', ...style }}
      className={`select-none p-0 bg-transparent border-0 transition-opacity
        ${disabled ? 'cursor-default' : 'cursor-pointer'}
        ${dimmed ? 'opacity-40' : 'opacity-100'}
        ${selected ? 'ring-2 ring-amber-300 rounded-lg' : ''}
        ${className ?? ''}`}
      aria-label={rankLabel(card.rank) + ' of ' + card.suit}
    >
      <img
        src={cardFaceAsset(card)}
        alt=""
        draggable={false}
        width={dim.w}
        height={dim.h}
        style={{ display: 'block', borderRadius: 6 }}
      />
    </button>
  )
}

// CardBack renders the hidden-card asset (section 7, section 35).
export function CardBack({ size = 'sm', style }: { size?: 'sm' | 'md' | 'lg'; style?: CSSProperties }) {
  const dim = SIZES[size]
  return (
    <img
      src={CARD_BACK_ASSET}
      alt="hidden card"
      draggable={false}
      width={dim.w}
      height={dim.h}
      style={{ display: 'block', borderRadius: 6, ...style }}
    />
  )
}
