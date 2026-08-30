import { Card } from './Card'
import { ANIM, animDuration } from '../config'
import type { PlayedCard } from '../protocol/messages'

interface TrickAreaProps {
  trick: PlayedCard[]
  you: number
  trump?: string
  collecting?: boolean
  winnerSeat?: number
}

// Relative positions on the table: bottom = you, left = next, top = partner,
// right = previous. Matches Hokm turn order rendering.
function positionOf(seat: number, you: number): string {
  const rel = (seat - you + 4) % 4
  switch (rel) {
    case 0:
      return 'translate(0, 40px)'
    case 1:
      return 'translate(-64px, 0)'
    case 2:
      return 'translate(0, -40px)'
    case 3:
      return 'translate(64px, 0)'
    default:
      return ''
  }
}

// Winner-collect target offset: cards drift toward the winner's seat and
// stack (-section 15). Presentation only - the server already decided the winner.
function collectTransform(seat: number, you: number, winner: number): string {
  const base = positionOf(seat, you)
  const rel = (winner - you + 4) % 4
  const target =
    rel === 0 ? 'translate(0, 90px) scale(0.55)' :
    rel === 1 ? 'translate(-120px, 0) scale(0.55)' :
    rel === 2 ? 'translate(0, -90px) scale(0.55)' :
    'translate(120px, 0) scale(0.55)'
  return `${base} ${target}`
}

// TrickArea renders cards currently in the trick, positioned around center.
// A played card animates in over CardPlayDuration (-section 13).
export function TrickArea({ trick, you, trump, collecting = false, winnerSeat = -1 }: TrickAreaProps) {
  return (
    <div className="relative flex items-center justify-center min-h-44 min-w-44">
      <div className="absolute inset-0 rounded-full bg-table-700/40 border border-table-600/60" />
      {trump ? (
        <div className="absolute top-1 right-1 text-[10px] uppercase tracking-widest text-amber-300/80">
          trump
        </div>
      ) : null}
      {trick.map((pc) => {
        const isWinner = winnerSeat === pc.seat
        const transform = collecting && winnerSeat >= 0
          ? collectTransform(pc.seat, you, winnerSeat)
          : positionOf(pc.seat, you)
        return (
          <div
            key={pc.seat}
            className={`play-card absolute ${isWinner && !collecting ? 'ring-2 ring-amber-300 rounded-lg z-10' : ''}`}
            style={{
              transform,
              transition: `transform ${animDuration(ANIM.cardPlayMs)}ms ease`,
            }}
          >
            <Card card={pc.card} size="md" />
          </div>
        )
      })}
    </div>
  )
}
