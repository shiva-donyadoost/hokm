import { useMemo, useState } from 'react'
import { Card } from './Card'
import { PlayerSeat } from './PlayerSeat'
import { TrickArea } from './TrickArea'
import { ChatPanel } from './ChatPanel'
import { useGame } from '../state/game'
import {
  Suits,
  type Room,
  type SeatView,
  type Suit,
} from '../protocol/messages'

interface GameTableProps {
  room: Room
  view: SeatView
}

// memberBySeat finds the lobby member occupying a seat.
function memberBySeat(room: Room, seat: number) {
  return room.members.find((m) => m.seat === seat) ?? null
}

export function GameTable({ room, view }: GameTableProps) {
  const playCard = useGame((s) => s.playCard)
  const selectTrump = useGame((s) => s.selectTrump)
  const lastError = useGame((s) => s.lastError)
  const [chatOpen, setChatOpen] = useState(false)

  const legalSuits = useMemo(() => {
    const trick = view.current_trick
    if (!trick || trick.length === 0 || !view.your_hand) {
      return null // leading: everything legal
    }
    const first = trick[0]
    if (!first) return null
    const lead = first.card.suit
    const hasLead = view.your_hand.some((c) => c.suit === lead)
    return hasLead ? lead : null
  }, [view])

  const myTurn =
    view.phase === 'trick_play' && view.turn === view.you &&
    (view.current_trick?.length ?? 0) < 4 && !view.match_over

  const trumpPending =
    view.phase === 'trump_selection' && view.hakem === view.you && !view.trump

  // Relative seat layout: bottom=you, left=next, top=partner, right=prev.
  const rel = (offset: number) => (view.you + offset) % 4

  return (
    <div className="flex flex-col min-h-dvh bg-gradient-to-b from-table-900 via-table-800 to-table-900">
      {/* Score bar */}
      <div className="flex items-center justify-between px-3 py-2 text-xs sm:text-sm">
        <div className="flex gap-3">
          <span className="font-semibold text-teal-200">
            A {view.rounds_won[0]} · {view.tricks_this_round[0]} tricks
          </span>
          <span className="text-slate-400">vs</span>
          <span className="font-semibold text-rose-200">
            {view.tricks_this_round[1]} tricks · {view.rounds_won[1]} B
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-slate-400">round {view.round_number}</span>
          {view.trump ? (
            <span className="px-2 py-0.5 rounded-full bg-amber-400/20 text-amber-300 uppercase">
              trump: {view.trump}
            </span>
          ) : null}
        </div>
      </div>

      {/* Table area */}
      <div className="flex-1 grid grid-rows-[auto_1fr_auto] p-2 gap-2">
        <div className="flex justify-center">
          <PlayerSeat
            member={memberBySeat(room, rel(2))}
            position="top"
            cardCount={view.hand_counts[rel(2)] ?? 0}
            isTurn={view.turn === rel(2)}
            isHakem={view.hakem === rel(2)}
          />
        </div>

        <div className="grid grid-cols-[auto_1fr_auto] items-center">
          <PlayerSeat
            member={memberBySeat(room, rel(1))}
            position="left"
            cardCount={view.hand_counts[rel(1)] ?? 0}
            isTurn={view.turn === rel(1)}
            isHakem={view.hakem === rel(1)}
          />
          <div className="flex justify-center">
            <TrickArea trick={view.current_trick ?? []} you={view.you} trump={view.trump} />
          </div>
          <PlayerSeat
            member={memberBySeat(room, rel(3))}
            position="right"
            cardCount={view.hand_counts[rel(3)] ?? 0}
            isTurn={view.turn === rel(3)}
            isHakem={view.hakem === rel(3)}
          />
        </div>

        {/* Your hand */}
        <div className="flex flex-col items-center gap-1 pb-2">
          <div
            className={`text-xs px-2 py-0.5 rounded-full ${
              myTurn ? 'bg-amber-400 text-slate-900 font-bold' : 'bg-slate-800 text-slate-400'
            }`}
          >
            {myTurn ? 'your turn' : view.turn === -1 ? 'resolving…' : 'waiting'}
          </div>
          <div className="flex flex-wrap justify-center gap-1 sm:gap-1.5">
            {(view.your_hand ?? []).map((c) => {
              const legal =
                myTurn && (legalSuits === null || c.suit === legalSuits)
              return (
                <Card
                  key={c.suit + c.rank}
                  card={c}
                  size="lg"
                  disabled={!legal}
                  onClick={legal ? () => playCard(c) : undefined}
                />
              )
            })}
          </div>
        </div>
      </div>

      {/* Overlays */}
      <button
        className={`fixed bottom-2 left-2 z-40 px-3 py-2 rounded-full border text-sm font-semibold shadow-lg
          ${chatOpen
            ? 'bg-teal-600 border-teal-500 text-white'
            : 'bg-slate-900/90 border-slate-600 text-slate-100'}`}
        onClick={() => setChatOpen((v) => !v)}
        aria-label="toggle chat"
      >
        💬 Chat
      </button>
      {chatOpen ? (
        <div className="fixed bottom-14 left-2 z-40 w-72 shadow-2xl">
          <ChatPanel compact />
        </div>
      ) : null}

      {trumpPending ? (
        <div className="fixed inset-x-0 bottom-0 p-4 bg-slate-900/95 border-t border-amber-400/40">
          <p className="text-center text-sm text-amber-300 mb-2">
            You are the Hakem — choose trump
          </p>
          <div className="flex justify-center gap-2">
            {Suits.map((s: Suit) => (
              <button
                key={s}
                onClick={() => selectTrump(s)}
                className="px-4 py-2 rounded-lg bg-slate-800 hover:bg-slate-700 active:scale-95 capitalize"
              >
                {s}
              </button>
            ))}
          </div>
        </div>
      ) : null}

      {view.match_over ? (
        <MatchOverOverlay view={view} />
      ) : null}

      {lastError ? (
        <div className="fixed top-12 inset-x-0 flex justify-center pointer-events-none">
          <div className="px-3 py-1.5 rounded-lg bg-rose-600/90 text-white text-sm shadow-lg">
            {lastError}
          </div>
        </div>
      ) : null}
    </div>
  )
}

function MatchOverOverlay({ view }: { view: SeatView }) {
  const myTeam = view.you % 2
  const winner = view.rounds_won[0] > view.rounds_won[1] ? 0 : 1
  const won = winner === myTeam
  return (
    <div className="fixed inset-0 bg-slate-950/80 flex items-center justify-center p-4">
      <div className="bg-slate-900 rounded-2xl p-6 max-w-sm w-full text-center border border-slate-700">
        <h2 className={`text-2xl font-black mb-2 ${won ? 'text-emerald-400' : 'text-rose-400'}`}>
          {won ? 'Victory!' : 'Defeat'}
        </h2>
        <p className="text-slate-300 mb-4">
          Team A {view.rounds_won[0]} — {view.rounds_won[1]} Team B
        </p>
        <a
          href="/rooms"
          className="inline-block px-4 py-2 rounded-lg bg-teal-600 hover:bg-teal-500 font-semibold"
        >
          Back to rooms
        </a>
      </div>
    </div>
  )
}
