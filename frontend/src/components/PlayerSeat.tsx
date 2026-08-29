import { CardBack } from './Card'
import type { RoomMember } from '../protocol/messages'

interface PlayerSeatProps {
  member: RoomMember | null
  position: 'top' | 'left' | 'right'
  cardCount: number
  isTurn: boolean
  isHakem: boolean
}

// PlayerSeat shows an opponent: name plate, card backs, turn indicator.
export function PlayerSeat({ member, position, cardCount, isTurn, isHakem }: PlayerSeatProps) {
  const align =
    position === 'top'
      ? 'items-center'
      : position === 'left'
        ? 'items-start'
        : 'items-end'
  return (
    <div className={`flex flex-col gap-1 ${align}`}>
      <div
        className={`px-2 py-1 rounded-full text-xs font-semibold max-w-28 sm:max-w-36 truncate
          ${isTurn ? 'bg-amber-400 text-slate-900 animate-pulse' : 'bg-slate-800 text-slate-200'}
          ${isHakem ? 'ring-2 ring-amber-300' : ''}`}
        title={isHakem ? 'Hakem' : undefined}
      >
        {isHakem ? '👑 ' : ''}
        {member ? member.username + (member.is_ai ? ' 🤖' : '') : '…'}
      </div>
      <div className="flex gap-0.5">
        {member
          ? Array.from({ length: Math.min(cardCount, 8) }).map((_, i) => (
              <CardBack key={i} size="sm" />
            ))
          : null}
      </div>
    </div>
  )
}
