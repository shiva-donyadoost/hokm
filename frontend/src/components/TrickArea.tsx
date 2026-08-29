import { Card } from './Card'
import type { PlayedCard } from '../protocol/messages'

interface TrickAreaProps {
  trick: PlayedCard[]
  you: number
  trump?: string
}

// Relative positions on the table: bottom = you, left = next, top = partner,
// right = previous. Matches Hokm turn order (counter-clockwise rendering).
function positionOf(seat: number, you: number): string {
  const rel = (seat - you + 4) % 4
  switch (rel) {
    case 0:
      return 'translate-y-10'
    case 1:
      return '-translate-x-16'
    case 2:
      return '-translate-y-10'
    case 3:
      return 'translate-x-16'
    default:
      return ''
  }
}

// TrickArea renders cards currently in the trick, positioned around center.
export function TrickArea({ trick, you, trump }: TrickAreaProps) {
  return (
    <div className="relative flex items-center justify-center min-h-44 min-w-44">
      <div className="absolute inset-0 rounded-full bg-table-700/40 border border-table-600/60" />
      {trump ? (
        <div className="absolute top-1 right-1 text-[10px] uppercase tracking-widest text-amber-300/80">
          trump
        </div>
      ) : null}
      {trick.map((pc) => (
        <div
          key={pc.seat}
          className={`absolute transition-transform duration-300 ${positionOf(pc.seat, you)}`}
        >
          <Card card={pc.card} size="md" />
        </div>
      ))}
    </div>
  )
}
