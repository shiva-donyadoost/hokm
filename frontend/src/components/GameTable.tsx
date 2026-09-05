import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import type { PointerEvent as ReactPointerEvent } from 'react'
import { Card, CardBack } from './Card'
import { PlayerAvatar, avatarSeed } from './PlayerAvatar'
import { ChatPanel } from './ChatPanel'
import { TrickArea } from './TrickArea'
import { useGame } from '../state/game'
import { useAuth } from '../state/auth'
import { ANIM, animDuration } from '../config'
import { diagInfo, logPlayAttempt, logTurnSnapshot } from '../diagnostics/clientLog'
import {
  cardKey,
  isCardLegal,
  isMyTurn,
  legalCards,
  requiredLeadSuit,
} from '../game/legality'
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
  const connected = useGame((s) => s.connected)
  const ws = useGame((s) => s.ws)
  const myId = useAuth0()
  const narrow = useNarrow()

  const [selected, setSelected] = useState<string | null>(null)
  const [chatOpen, setChatOpen] = useState(false)
  const [unread, setUnread] = useState(0)
  const [dealKey, setDealKey] = useState(0)
  const [reveal, setReveal] = useState<{ number: number; winner: number } | null>(null)
  const [collecting, setCollecting] = useState(false)
  const lastTrickSeen = useRef(0)
  const drag = useRef<{
    card: string | null
    startX: number
    startY: number
    dx: number
    dy: number
    moved: boolean
  } | null>(null)

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
  useLayoutEffect(() => {
    const lt = view.last_trick
    if (!lt || lt.number === lastTrickSeen.current) return
    if ((view.current_trick?.length ?? 0) > 0) return
    lastTrickSeen.current = lt.number
    setReveal({ number: lt.number, winner: lt.winner })
    setCollecting(false)
    const revealMs = animDuration(ANIM.trickWinnerMs)
    const collectMs = animDuration(ANIM.cardCollectionMs)
    const t1 = setTimeout(() => setCollecting(true), revealMs)
    const t2 = setTimeout(() => {
      setReveal(null)
      setCollecting(false)
    }, revealMs + collectMs)
    return () => { clearTimeout(t1); clearTimeout(t2) }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view.last_trick?.number, view.current_trick?.length])

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

  const leadSuit = useMemo(
    () => requiredLeadSuit(view.your_hand, view.current_trick),
    [view.your_hand, view.current_trick],
  )

  const myTurn = isMyTurn({
    phase: view.phase,
    turn: view.turn,
    you: view.you,
    trickLen: view.current_trick?.length ?? 0,
    matchOver: view.match_over,
  })

  const legalOpts = useMemo(() => ({ myTurn, leadSuit }), [myTurn, leadSuit])
  const isLegal = (c: { suit: Suit }) => isCardLegal(c, legalOpts)
  const followsSuit = (c: { suit: Suit }) => leadSuit === null || c.suit === leadSuit
  const keyOf = cardKey

  const trumpPending =
    view.phase === 'trump_selection' && view.hakem === view.you && !view.trump

  const wsState = ws?.readyState ?? 'none'
  const handKeys = useMemo(
    () => (view.your_hand ?? []).map(cardKey),
    [view.your_hand],
  )
  const legalKeys = useMemo(
    () => legalCards(view.your_hand ?? [], legalOpts).map(cardKey),
    [view.your_hand, legalOpts],
  )

  // Snapshot when turn / phase / connection changes (stuck-turn diagnosis).
  useEffect(() => {
    logTurnSnapshot({
      phase: view.phase,
      turn: view.turn,
      you: view.you,
      myTurn,
      trumpPending,
      matchOver: view.match_over,
      trickLen: view.current_trick?.length ?? 0,
      handKeys,
      legalKeys,
      selected,
      collecting,
      reveal: reveal !== null,
      connected,
      wsState,
      deadlineKind: view.deadline_kind,
    })
  }, [
    view.phase, view.turn, view.you, myTurn, trumpPending, view.match_over,
    view.current_trick?.length, handKeys, legalKeys, selected, collecting, reveal,
    connected, wsState, view.deadline_kind,
  ])

  const tryPlay = (c: { suit: Suit; rank: number }) => {
    if (!isLegal(c)) {
      logPlayAttempt({
        action: 'play_card', ok: false, reason: 'not_legal_client',
        card: c, myTurn, phase: view.phase, turn: view.turn, you: view.you,
        wsState, legalKeys,
      })
      return
    }
    if (!connected || ws?.readyState !== WebSocket.OPEN) {
      logPlayAttempt({
        action: 'play_card', ok: false, reason: 'ws_not_open',
        card: c, myTurn, phase: view.phase, turn: view.turn, you: view.you,
        wsState, legalKeys,
      })
    } else {
      logPlayAttempt({
        action: 'play_card', ok: true, reason: 'sent',
        card: c, myTurn, phase: view.phase, turn: view.turn, you: view.you,
        wsState, legalKeys,
      })
    }
    playCard(c)
    setSelected(null)
  }

  const onCardTap = (c: { suit: Suit; rank: number }) => {
    if (trumpPending) {
      diagInfo('play', 'select_trump_tap', { suit: c.suit, wsState })
      selectTrump(c.suit)
      return
    }
    if (!isLegal(c)) {
      logPlayAttempt({
        action: 'play_card', ok: false, reason: 'tap_not_legal',
        card: c, myTurn, phase: view.phase, turn: view.turn, you: view.you,
        wsState, legalKeys,
      })
      return
    }
    const k = keyOf(c)
    if (selected === k) {
      tryPlay(c)
    } else {
      setSelected(k)
    }
  }

  // Drag on the INNER transform node so CSS deal animation (outer) cannot
  // own the same transform used for fan + drag (ADR-0014).
  const onDragStart = (e: ReactPointerEvent, c: { suit: Suit; rank: number }) => {
    if (!trumpPending && !isLegal(c)) return
    e.currentTarget.setPointerCapture(e.pointerId)
    drag.current = { card: keyOf(c), startX: e.clientX, startY: e.clientY, dx: 0, dy: 0, moved: false }
  }
  const onDragMove = (e: ReactPointerEvent) => {
    if (!drag.current) return
    drag.current.dx = e.clientX - drag.current.startX
    drag.current.dy = e.clientY - drag.current.startY
    if (Math.hypot(drag.current.dx, drag.current.dy) > 12) drag.current.moved = true
    if (!drag.current.moved) return
    const played = view.your_hand?.find((c) => keyOf(c) === drag.current?.card)
    if (!played || !isLegal(played)) return
    const el = document.getElementById('card-' + drag.current.card)
    if (el) el.style.transform = `translate(${drag.current.dx}px, ${drag.current.dy - 24}px) scale(1.08)`
  }
  const resetDragTransform = (cardId: string | null) => {
    if (!cardId) return
    const el = document.getElementById('card-' + cardId)
    if (!el) return
    el.style.transition = `transform ${animDuration(ANIM.cardReturnMs)}ms ease`
    el.style.transform = ''
    setTimeout(() => { if (el) el.style.transition = '' }, animDuration(ANIM.cardReturnMs))
  }
  const onDragEnd = (c: { suit: Suit; rank: number }) => {
    const d = drag.current
    if (!d) return
    drag.current = null
    resetDragTransform(d.card)
    if (trumpPending) {
      onCardTap(c)
      return
    }
    const played = view.your_hand?.find((x) => keyOf(x) === d.card)
    const liftThresh = narrow ? 36 : 64
    if (played && d.moved && d.dy < -liftThresh && isLegal(played)) {
      tryPlay(played)
      return
    }
    if (!d.moved) onCardTap(c)
  }
  const onDragCancel = () => {
    const d = drag.current
    drag.current = null
    if (d) resetDragTransform(d.card)
  }

  const actingSeat =
    view.deadline_kind === 'trump' ? view.hakem :
    view.deadline_kind === 'card' ? view.turn : -1
  const actingDeadline = actingSeat >= 0 ? (view.deadline_unix_ms ?? 0) : 0

  const rel = (offset: number) => (view.you + offset) % 4
  const hand = sortHand(view.your_hand ?? [], view.trump)
  const self = memberBySeat(room, view.you)
  const teamTricks = (seat: number) => view.tricks_this_round[seat % 2] ?? 0
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
            tricks={teamTricks(rel(2))}
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
            tricks={teamTricks(rel(1))}
            hideHand={narrow}
          />
          <div className="flex justify-center relative">
            <TrickArea
              trick={view.last_trick && showReveal ? view.last_trick.cards : (view.current_trick ?? [])}
              you={view.you}
              trump={view.trump}
              collecting={collecting}
              winnerSeat={showReveal ? revealWinnerSeat : -1}
              skipEnter={showReveal}
            />
          </div>
          <SeatPlate
            member={memberBySeat(room, rel(3))}
            cardCount={view.hand_counts[rel(3)] ?? 0}
            isTurn={actingSeat === rel(3)}
            isHakem={view.hakem === rel(3)}
            deadline={actingSeat === rel(3) ? actingDeadline : 0}
            tricks={teamTricks(rel(3))}
            hideHand={narrow}
          />
        </div>

        {/* Overlapping hand: fan on desktop, flat row on mobile (ADR-0013). */}
        <div className="flex flex-col items-center gap-1 pb-2">
          <div className="flex items-center gap-2 max-w-[90vw]">
            <PlayerAvatar
              seed={avatarSeed(self?.user_id ?? myId, self?.username, self?.avatar_seed)}
              style={self?.avatar_style}
              name={self?.username ?? "you"}
              size="sm"
            />
            <div
              className={`px-2 py-1 rounded-full text-xs font-semibold truncate
              ${actingSeat === view.you ? 'bg-amber-400 text-slate-900' : 'bg-slate-800 text-slate-200'}
              ${view.hakem === view.you ? 'ring-2 ring-amber-300' : ''}`}
            >
              {view.hakem === view.you ? '[H] ' : ''}
              {self ? self.username : 'you'}
              <span className="ml-1 text-[10px] font-bold opacity-80">{teamTricks(view.you)}</span>
            </div>
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
              const fan = n <= 1 ? 0 : Math.min(8, 42 / n)
              const angle = narrow ? 0 : (i - mid) * fan
              const overlap = n <= 1 ? 0 : (narrow ? 34 : Math.min(42, 22 + n))
              const drop = narrow ? 0 : Math.abs(i - mid) * Math.abs(i - mid) * 1.4
              const lift = selected === k ? drop - (narrow ? 12 : 22) : drop
              const dealDelay = prevHand.current === 5 ? i * ANIM.dealStaggerMs : 0
              const playable = trumpPending || isLegal(c)
              return (
                <div
                  key={k}
                  data-deal={dealDelay}
                  className="deal-card"
                  style={{
                    marginLeft: i === 0 ? 0 : -overlap,
                    zIndex: selected === k ? 50 : i,
                    animationDelay: `${dealDelay}ms`,
                    // Illegal cards must not swallow hits on covered legal cards.
                    pointerEvents: playable ? 'auto' : 'none',
                  }}
                >
                  <div
                    id={'card-' + k}
                    className="touch-none"
                    style={{
                      transform: `rotate(${angle}deg) translateY(${lift}px)`,
                      transformOrigin: 'bottom center',
                      touchAction: 'none',
                    }}
                    onPointerDown={playable ? (e) => onDragStart(e, c) : undefined}
                    onPointerMove={playable ? onDragMove : undefined}
                    onPointerUp={playable ? () => onDragEnd(c) : undefined}
                    onPointerCancel={playable ? onDragCancel : undefined}
                  >
                    <Card
                      card={c}
                      size={narrow ? 'md' : 'lg'}
                      disabled={!playable}
                      dimmed={view.phase === 'trick_play' && !followsSuit(c)}
                      selected={selected === k}
                      style={{ pointerEvents: 'none' }}
                    />
                  </div>
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

      {!connected ? (
        <div className="fixed top-12 inset-x-0 flex justify-center pointer-events-none z-50">
          <div className="px-3 py-1.5 rounded-lg bg-amber-500/95 text-slate-900 text-sm shadow-lg font-semibold">
            disconnected - reconnecting...
          </div>
        </div>
      ) : null}

      {lastError ? (
        <div className="fixed top-20 inset-x-0 flex justify-center pointer-events-none z-50">
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

function SeatPlate({ member, cardCount, isTurn, isHakem, deadline, tricks, hideHand }: {
  member: ReturnType<typeof memberBySeat>
  cardCount: number
  isTurn: boolean
  isHakem: boolean
  deadline: number
  tricks: number
  hideHand: boolean
}) {
  return (
    <div className="flex flex-col gap-1 items-center max-w-24 sm:max-w-36">
      <PlayerAvatar
        seed={avatarSeed(member?.user_id, member?.username, member?.avatar_seed)}
        style={member?.avatar_style}
        name={member?.username ?? ""}
        size="sm"
      />
      <div
        className={`px-2 py-1 rounded-full text-xs font-semibold truncate
          ${isTurn ? 'bg-amber-400 text-slate-900 animate-pulse' : 'bg-slate-800 text-slate-200'}
          ${isHakem ? 'ring-2 ring-amber-300' : ''}`}
        title={isHakem ? 'Hakem' : undefined}
      >
        {isHakem ? '[H] ' : ''}
        {member ? member.username + (member.is_ai ? ' [AI]' : '') : '...'}
        <span className="ml-1 text-[10px] font-bold opacity-80">{tricks}</span>
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
