import { isRed, rankLabel, SUIT_SYMBOL, type Card as CardT } from '../protocol/messages'

interface CardProps {
  card: CardT
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  onClick?: () => void
}

const SIZES = {
  sm: 'w-9 h-13 text-xs rounded-md',
  md: 'w-12 h-18 text-base rounded-lg',
  lg: 'w-14 h-20 text-lg rounded-lg',
}

// Card renders a single playing card with CSS only (no images).
export function Card({ card, size = 'md', disabled, onClick }: CardProps) {
  const red = isRed(card.suit)
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={`select-none ${SIZES[size]} bg-white shadow-md border border-slate-300
        flex flex-col items-center justify-center font-bold
        ${red ? 'text-rose-600' : 'text-slate-900'}
        ${disabled ? 'opacity-40' : 'active:scale-95 transition-transform cursor-pointer'}`}
      aria-label={rankLabel(card.rank) + ' of ' + card.suit}
    >
      <span className="leading-none">{rankLabel(card.rank)}</span>
      <span className="leading-none text-lg">{SUIT_SYMBOL[card.suit]}</span>
    </button>
  )
}

// CardBack renders the face-down card.
export function CardBack({ size = 'sm' }: { size?: 'sm' | 'md' | 'lg' }) {
  return (
    <div
      className={`${SIZES[size]} bg-teal-800 border border-teal-600 rounded-lg shadow`}
      aria-hidden
    />
  )
}
