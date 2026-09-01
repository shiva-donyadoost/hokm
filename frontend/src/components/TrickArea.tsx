import { useEffect, useState } from 'react'
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

function relOf(seat: number, you: number): number {
  return (seat - you + 4) % 4
}

function seatVector(rel: number, dist: number): string {
  switch (rel) {
    case 0:
      return `translate(0px, ${dist}px)`
    case 1:
      return `translate(-${dist}px, 0px)`
    case 2:
      return `translate(0px, -${dist}px)`
    case 3:
      return `translate(${dist}px, 0px)`
    default:
      return 'translate(0px, 0px)'
  }
}

function restTransform(seat: number, you: number): string {
  return seatVector(relOf(seat, you), 48)
}

function originTransform(seat: number, you: number): string {
  return `${seatVector(relOf(seat, you), 170)} scale(0.85)`
}

function collectTransform(you: number, winner: number, idx: number): string {
  const spread = (idx % 4) * 8 - 12
  const rel = relOf(winner, you)
  const along =
    rel === 0 || rel === 2
      ? `translate(${spread}px, 0px)`
      : `translate(0px, ${spread}px)`
  return `${seatVector(rel, 168)} ${along} scale(0.55)`
}

function FlyingCard({
  pc,
  you,
  collecting,
  winnerSeat,
  idx,
}: {
  pc: PlayedCard
  you: number
  collecting: boolean
  winnerSeat: number
  idx: number
}) {
  const rest = restTransform(pc.seat, you)
  const origin = originTransform(pc.seat, you)
  const collect = collectTransform(you, winnerSeat, idx)
  const [transform, setTransform] = useState(origin)

  useEffect(() => {
    const id = requestAnimationFrame(() => setTransform(rest))
    return () => cancelAnimationFrame(id)
  }, [rest])

  useEffect(() => {
    if (collecting && winnerSeat >= 0) {
      setTransform(collect)
    }
  }, [collecting, winnerSeat, collect])

  const ms = collecting ? animDuration(ANIM.cardCollectionMs) : animDuration(ANIM.cardPlayMs)
  return (
    <div
      className="absolute"
      style={{
        transform,
        transition: `transform ${ms}ms ease-out`,
        zIndex: collecting ? idx : pc.seat,
      }}
    >
      <Card card={pc.card} size="md" />
    </div>
  )
}

export function TrickArea({ trick, you, collecting = false, winnerSeat = -1 }: TrickAreaProps) {
  return (
    <div className="relative flex items-center justify-center min-h-44 min-w-44">
      <div className="absolute inset-0 rounded-full bg-table-700/40 border border-table-600/60" />
      {trick.map((pc, idx) => (
        <FlyingCard
          key={pc.seat + '-' + pc.card.suit + '-' + pc.card.rank}
          pc={pc}
          you={you}
          collecting={collecting}
          winnerSeat={winnerSeat}
          idx={idx}
        />
      ))}
    </div>
  )
}
