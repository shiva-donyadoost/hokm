import { useEffect, useMemo, useRef, useState } from 'react'
import type { PointerEvent as ReactPointerEvent } from 'react'
import { Card, CardBack } from './Card'
import { ChatPanel } from './ChatPanel'
import { TrickArea } from './TrickArea'
import { useGame } from '../state/game'
import { useAuth } from '../state/auth'
import { ANIM, animDuration } from '../config'
import {
  SUIT_GLYPH,
  isRed,
  sortHand,
  type RoundResult,
  type Room,
  type SeatView,
  type Suit,
} from '../protocol/messages'

interface GameTableProps {
  room: Room
  view: SeatView
}

function memberBySeat(room: Room, seat: number) {
  return room.members.find((m) => m.seat === seat) ?? null
}

// Countdown bar (impliment.md section 9): renders from the authoritative
// server deadline; recalculated locally - no per-second server traffic.
function CountdownBar({ deadlineMs, label }: { deadlineMs: number; label: string }) {
  const totalRef = useRef(Math.max(1, deadlineMs - Date.now()))
  const [remaining, setRemaining] = useState(() => Math.max(0, deadlineMs - Date.now()))
  useEffect(() => {
    totalRef.current = Math.max(1, deadlineMs - Date.now())
    setRemaining(Math.max(0, deadlineMs - Date.now()))
    const id = setInterval(() => setRemaining(Math.max(0, deadlineMs - Date.now())), 100)
    return () => clearInterval(id)
  }, [deadlineMs])
  const pct = Math.min(100, Math.max(0, ((totalRef.current - remaining) / totalRef.current) * 100))
  const urgent = pct > 70
  return (
    <div
      className="w-full h-2 bg-slate-700 rounded-full overflow-hidden mt-0.5"
      role="timer"
      aria-label={`${label}: ${Math.ceil(remaining / 1000)} seconds remaining`}
    >
      <div
        className={`h-full transition-[width] duration-100 ease-linear ${urgent ? 'bg-rose-400' : 'bg-amber-400'}`}
        style={{ width: `${pct}%` }}
      />
    </div>
  )
}

function useNarrow(): boolean {
  const [narrow, setNarrow] = useState(() =>
    typeof window !== 'undefined' && window.matchMedia('(max-width: 639px)').matches,
  )
  useEffect(() => {
    const mq = window.matchMedia('(max-width: 639px)')
    const onChange = () => setNarrow(mq.matches)
    onChange()
    mq.addEventListener('change', onChange)
    return () => mq.removeEventListener('change', onChange)
  }, [])
  return narrow
}

export function GameTable({ room, view }: GameTableProps) {
  const playCard = useGame((s) => s.playCard)
  const selectTrump = useGame((s) => s.selectTrump)
  const replayGame = useGame((s) => s.replayGame)
  const lastError = useGame((s) => s.lastError)
  const chat = useGame((s) => s.chat)
  const myId = useAuth0()
  const narrow = useNarrow()

  const [selected, setSelected] = useState<string | null>(null)
  const [chatOpen, setChatOpen] = useState(false)
  const [unread, setUnread] = useState(0)
  const [dealKey, setDealKey] = useState(0)
  const [reveal, setReveal] = useState<{ number: number; winner: number } | null>(null)
  const [collecting, setCollecting] = useState(false)
  const lastTrickSeen = useRef(0)
  const drag = useRef<{ card: string | null; startX: number; startY: number; dx: number; dy: number } | null>(null)

  // Unread badge: count others' messages while chat is closed.
  const lastSeenChat = useRef(chat.length)
  useEffect(() => {
    if (chatOpen) {
      setUnread(0)
      lastSeenChat.current = chat.length
      return
    }
    const newOnes = chat.slice(lastSeenChat.current).filter((m) => m.user_id !== myId && !m.is_system)
    if (newOnes.length > 0) setUnread((u) => u + newOnes.length)
    lastSeenChat.current = chat.length
  }, [chat, chatOpen, myId])

  // Winner reveal -> collect animation, driven by server trick events.
  useEffect(() => {
    const lt = view.last_trick
    if (!lt || lt.number === lastTrickSeen.current) return
    lastTrickSeen.current = lt.number
    setReveal({ number: lt.number, winner: lt.winner })
    const revealMs = animDuration(ANIM.trickWinnerMs)
    const collectMs = animDuration(ANIM.cardCollectionMs)
    const t1 = setTimeout(() => setCollecting(true), revealMs)
    const t2 = setTimeout(() => {
      setReveal(null)
      setCollecting(false)
    }, revealMs + collectMs)
    return () => { clearTimeout(t1); clearTimeout(t2) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view.last_trick?.number])

  useEffect(() => {
    if (view.phase === 'trump_selection' && view.round_number === 1 && !view.match_over) {
      lastTrickSeen.current = 0
      setReveal(null)
      setCollecting(false)
    }
  }, [view.phase, view.round_number, view.match_over])

  // Dealing animation trigger: hand size jumps from 0 to the initial deal.
  const handLen = view.your_hand?.length ?? 0
  const prevHand = useRef(0)
  useEffect(() => {
    if (prevHand.current === 0 && handLen === 5) setDealKey((k) => k + 1)
    prevHand.current = handLen
  }, [handLen])

  const leadSuit = useMemo(() => {
    const trick = view.current_trick
    if (!trick || trick.length === 0 || !view.your_hand) return null
    const first = trick[0]
    if (!first) return null
    const lead = first.card.suit
    const hasLead = view.your_hand.some((c) => c.suit === lead)
    return hasLead ? lead : null
  }, [view])

  const myTurn =
    view.phase === 'trick_play' && view.turn === view.you &&
    (view.current_trick?.length ?? 0) < 4 && !view.match_over

  const followsSuit = (c: { suit: Suit }) => leadSuit === null || c.suit === leadSuit
  const isLegal = (c: { suit: Suit }) => myTurn && followsSuit(c)

  const keyOf = (c: { suit: Suit; rank: number }) => c.suit + c.rank

  const tryPlay = (c: { suit: Suit; rank: number }) => {
    if (!isLegal(c)) return
    playCard(c)
    setSelected(null)
  }

  const trumpPending =
    view.phase === 'trump_selection' && view.hakem === view.you && !view.trump

  const onCardTap = (c: { suit: Suit; rank: number }) => {
    if (trumpPending) {
      selectTrump(c.suit)
      return
    }
    if (!isLegal(c)) return
    const k = keyOf(c)
    if (selected === k) {
      tryPlay(c)
    } else {
      setSelected(k)
    }
  }

  // Drag and drop via pointer events.
  const onDragStart = (e: ReactPointerEvent, c: { suit: Suit; rank: number }) => {
    if (!isLegal(c)) return
    drag.current = { card: keyOf(c), startX: e.clientX, startY: e.clientY, dx: 0, dy: 0 }
    ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
  }
  const onDragMove = (e: ReactPointerEvent) => {
    if (!drag.current) return
    drag.current.dx = e.clientX - drag.current.startX
    drag.current.dy = e.clientY - drag.current.startY
    const el = document.getElementById('card-' + drag.current.card)
    if (el) el.style.transform = `translate(${drag.current.dx}px, ${drag.current.dy - 24}px) scale(1.08)`
  }
  const onDragEnd = () => {
    const d = drag.current
    if (!d) return
    const el = document.getElementById('card-' + d.card)
    drag.current = null
    if (el) {
      el.style.transition = `transform ${animDuration(ANIM.cardReturnMs)}ms ease`
      el.style.transform = ''
      setTimeout(() => { if (el) el.style.transition = '' }, animDuration(ANIM.cardReturnMs))
    }
    const played = view.your_hand?.find((c) => keyOf(c) === d.card)
    if (played && Math.abs(d.dy) > 80 && isLegal(played)) {
      tryPlay(played) // released toward the table
    }
    // otherwise: cancelled - card snaps back, no gameplay action
  }

  const actingSeat =
    view.deadline_kind === 'trump' ? view.hakem :
    view.deadline_kind === 'card' ? view.turn : -1
  const actingDeadline = actingSeat >= 0 ? (view.deadline_unix_ms ?? 0) : 0

  const rel = (offset: number) => (view.you + offset) % 4
  const hand = sortHand(view.your_hand ?? [], view.trump)
  const self = memberBySeat(room, view.you)
  const teamRounds = (seat: number) => view.rounds_won[seat % 2] ?? 0
  const seatsOccupied = room.members.length === 4

  // Winner reveal overlay for the last completed trick (presentation only).
  const showReveal = reveal !== null && view.last_trick !== null &&
    (view.current_trick?.length ?? 0) === 0
  const revealWinnerSeat = reveal?.winner ?? -1

  return (
    <div className="flex flex-col min-h-dvh bg-gradient-to-b from-table-900 via-table-800 to-table-900 overflow-x-hidden">
      {/* Score bar */}
      <div className="flex items-center justify-between px-3 py-2 text-xs sm:text-sm">
        <div className="flex gap-3">
          <span className="font-semibold text-teal-200">
            Team A {view.rounds_won[0]} - {view.tricks_this_round[0]} tricks
          </span>
          <span className="text-slate-400">vs</span>
          <span className="font-semibold text-rose-200">
            {view.tricks_this_round[1]} tricks - {view.rounds_won[1]} Team B
          </span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-slate-400">round {view.round_number}/{room.round_count}</span>
          {view.trump ? <TrumpMark suit={view.trump} /> : null}
        </div>
      </div>

      {/* Round history strip */}
      {room.round_count > 1 ? (
        <div className="flex gap-1.5 px-3 pb-1 text-[11px] overflow-x-auto" aria-label="round results">
          {Array.from({ length: room.round_count }).map((_, i) => {
            const num = i + 1
            const done = (view.round_history ?? []).find((r: RoundResult) => r.number === num)
            const label =
              done ? `R${num}: ${done.winner_team === 0 ? 'Team A' : 'Team B'} won` :
              num === view.round_number ? `R${num} playing` : `R${num} not started`
            return (
              <span
                key={num}
                className={`px-2 py-0.5 rounded-full whitespace-nowrap ${
                  done
                    ? 'bg-slate-800 text-slate-300'
                    : num === view.round_number
                      ? 'bg-teal-700/60 text-teal-200'
                      : 'bg-slate-900 text-slate-600'
                }`}
              >
                {label}
              </span>
            )
          })}
        </div>
      ) : null}

      {/* Table area */}
      <div className="flex-1 grid grid-rows-[auto_1fr_auto] p-2 gap-2">
        <div className="flex justify-center">
          <SeatPlate
            member={memberBySeat(room, rel(2))}
            cardCount={view.hand_counts[rel(2)] ?? 0}
            isTurn={actingSeat === rel(2)}
            isHakem={view.hakem === rel(2)}
            deadline={actingSeat === rel(2) ? actingDeadline : 0}
            roundsWon={teamRounds(rel(2))}
            hideHand={narrow}
          />
        </div>

        <div className="grid grid-cols-[auto_1fr_auto] items-center">
          <SeatPlate
            member={memberBySeat(room, rel(1))}
            cardCount={view.hand_counts[rel(1)] ?? 0}
            isTurn={actingSeat === rel(1)}
            isHakem={view.hakem === rel(1)}
            deadline={actingSeat === rel(1) ? actingDeadline : 0}
            roundsWon={teamRounds(rel(1))}
            hideHand={narrow}
          />
          <div className="flex justify-center relative">
            <TrickArea
              trick={view.last_trick && showReveal ? view.last_trick.cards : (view.current_trick ?? [])}
              you={view.you}
              trump={view.trump}
              collecting={collecting}
              winnerSeat={showReveal ? revealWinnerSeat : -1}
            />
          </div>
          <SeatPlate
            member={memberBySeat(room, rel(3))}
            cardCount={view.hand_counts[rel(3)] ?? 0}
            isTurn={actingSeat === rel(3)}
            isHakem={view.hakem === rel(3)}
            deadline={actingSeat === rel(3) ? actingDeadline : 0}
            roundsWon={teamRounds(rel(3))}
            hideHand={narrow}
          />
        </div>

        {/* Arc hand with tap/drag interaction */}
        <div className="flex flex-col items-center gap-1 pb-2">
          <div
            className={`px-2 py-1 rounded-full text-xs font-semibold max-w-[90vw] truncate
              ${actingSeat === view.you ? 'bg-amber-400 text-slate-900' : 'bg-slate-800 text-slate-200'}
              ${view.hakem === view.you ? 'ring-2 ring-amber-300' : ''}`}
          >
            {view.hakem === view.you ? '[H] ' : ''}
            {self ? self.username : 'you'}
            <span className="ml-1 text-[10px] font-bold opacity-80">{teamRounds(view.you)}</span>
          </div>
          <div
            className={`text-xs px-2 py-0.5 rounded-full ${
              trumpPending || myTurn ? 'bg-amber-400 text-slate-900 font-bold' : 'bg-slate-800 text-slate-400'
            }`}
          >
            {trumpPending
              ? 'tap a card to choose trump'
              : myTurn
                ? (selected ? 'tap again to play' : 'your turn')
                : view.turn === -1 ? 'resolving...' : 'waiting'}
          </div>
          {actingSeat === view.you && actingDeadline ? (
            <div className="w-48">
              <CountdownBar deadlineMs={actingDeadline} label="your time" />
            </div>
          ) : null}
          <div
            className={`relative flex items-end justify-center w-full max-w-full sm:max-w-md px-1 ${
              narrow ? 'h-28' : 'h-32 sm:h-36'
            }`}
            key={dealKey}
          >
            {hand.map((c, i) => {
              const k = keyOf(c)
              const n = hand.length
              const mid = (n - 1) / 2
              const fan = n <= 1 ? 0 : Math.min(narrow ? 5 : 8, (narrow ? 28 : 42) / n)
              const angle = (i - mid) * fan
              const overlap = n <= 1 ? 0 : Math.min(narrow ? 50 : 42, (narrow ? 30 : 22) + n)
              const drop = Math.abs(i - mid) * Math.abs(i - mid) * (narrow ? 1.0 : 1.4)
              const lift = selected === k ? drop - (narrow ? 14 : 22) : drop
              const dealDelay = prevHand.current === 5 ? i * ANIM.dealStaggerMs : 0
              const playable = trumpPending || isLegal(c)
              return (
                <div
                  key={k}
                  id={'card-' + k}
                  data-deal={dealDelay}
                  className="deal-card"
                  style={{
                    transform: `rotate(${angle}deg) translateY(${lift}px)`,
                    transformOrigin: 'bottom center',
                    marginLeft: i === 0 ? 0 : -overlap,
                    zIndex: selected === k ? 50 : i,
                    animationDelay: `${dealDelay}ms`,
                  }}
                  onPointerDown={playable ? (e) => onDragStart(e, c) : undefined}
                  onPointerMove={playable ? onDragMove : undefined}
                  onPointerUp={playable ? onDragEnd : undefined}
                >
                  <Card
                    card={c}
                    size={narrow ? 'md' : 'lg'}
                    disabled={!playable}
                    dimmed={view.phase === 'trick_play' && !followsSuit(c)}
                    selected={selected === k}
                    onClick={() => onCardTap(c)}
                  />
                </div>
              )
            })}
          </div>
        </div>
      </div>

      {/* Overlays */}
      {room.chat_enabled ? (
        <>
          <button
            className={`fixed bottom-2 left-2 z-40 px-3 py-2 rounded-full border text-sm font-semibold shadow-lg
              ${chatOpen ? 'bg-teal-600 border-teal-500 text-white' : 'bg-slate-900/90 border-slate-600 text-slate-100'}`}
            onClick={() => setChatOpen((v) => !v)}
            aria-label={chatOpen ? 'close chat' : `open chat${unread ? `, ${unread} unread` : ''}`}
          >
            Chat{!chatOpen && unread ? ` (${unread})` : ''}
          </button>
          {chatOpen ? (
            <div className="fixed bottom-14 left-2 z-40 w-72 shadow-2xl">
              <ChatPanel compact />
            </div>
          ) : null}
        </>
      ) : null}

      {view.match_over ? (
        <MatchOverOverlay
          view={view}
          isHost={room.host_id === myId}
          seatsOccupied={seatsOccupied}
          onReplay={replayGame}
        />
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

function useAuth0(): string {
  return useAuth((s) => s.user?.id ?? '')
}

function SeatPlate({ member, cardCount, isTurn, isHakem, deadline, roundsWon, hideHand }: {
  member: ReturnType<typeof memberBySeat>
  cardCount: number
  isTurn: boolean
  isHakem: boolean
  deadline: number
  roundsWon: number
  hideHand: boolean
}) {
  return (
    <div className="flex flex-col gap-1 items-center max-w-24 sm:max-w-36">
      <div
        className={`px-2 py-1 rounded-full text-xs font-semibold truncate
          ${isTurn ? 'bg-amber-400 text-slate-900 animate-pulse' : 'bg-slate-800 text-slate-200'}
          ${isHakem ? 'ring-2 ring-amber-300' : ''}`}
        title={isHakem ? 'Hakem' : undefined}
      >
        {isHakem ? '[H] ' : ''}
        {member ? member.username + (member.is_ai ? ' [AI]' : '') : '...'}
        <span className="ml-1 text-[10px] font-bold opacity-80">{roundsWon}</span>
      </div>
      {deadline ? <CountdownBar deadlineMs={deadline} label="time to act" /> : null}
      {hideHand ? null : (
        <div className="flex gap-0.5">
          {member ? Array.from({ length: Math.min(cardCount, 8) }).map((_, i) => <CardBack key={i} size="sm" />) : null}
        </div>
      )}
    </div>
  )
}

function TrumpMark({ suit }: { suit: Suit }) {
  return (
    <span
      className={`px-2 py-0.5 rounded-full bg-slate-800 text-xl leading-none ${
        isRed(suit) ? 'text-rose-400' : 'text-slate-100'
      }`}
      aria-label={'trump is ' + suit}
    >
      {SUIT_GLYPH[suit]}
    </span>
  )
}

function MatchOverOverlay({
  view,
  isHost,
  seatsOccupied,
  onReplay,
}: {
  view: SeatView
  isHost: boolean
  seatsOccupied: boolean
  onReplay: () => void
}) {
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
          Team A {view.rounds_won[0]} - {view.rounds_won[1]} Team B
        </p>
        <div className="flex flex-col gap-2">
          {isHost ? (
            <button className="btn-primary w-full" onClick={onReplay}>
              Replay
            </button>
          ) : (
            <p className="text-sm text-slate-400">waiting for host to replay...</p>
          )}
          {seatsOccupied ? (
            <a href="/rooms" className="inline-block px-4 py-2 rounded-lg bg-slate-700 hover:bg-slate-600 font-semibold">
              Back to rooms
            </a>
          ) : null}
        </div>
      </div>
    </div>
  )
}
