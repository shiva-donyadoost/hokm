import { useEffect, useRef, useState } from 'react'
import type { PointerEvent as ReactPointerEvent } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../state/auth'
import { useGame } from '../state/game'
import { GameTable } from '../components/GameTable'
import { ChatPanel } from '../components/ChatPanel'
import { CardBack } from '../components/Card'
import { Header } from './Rooms'
import type { RoomMember } from '../protocol/messages'

// Room is the lobby + live game surface. It subscribes once on mount and
// renders whichever stage the server reports (ADR-0004: server authority).
export function Room() {
  const { id = '' } = useParams()
  const user = useAuth((s) => s.user)
  const navigate = useNavigate()
  const connect = useGame((s) => s.connect)
  const disconnect = useGame((s) => s.disconnect)
  const startGame = useGame((s) => s.startGame)
  const connected = useGame((s) => s.connected)
  const room = useGame((s) => s.room)
  const view = useGame((s) => s.view)

  useEffect(() => {
    if (!id || !user) return
    void connect(id)
    return () => disconnect()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, user?.id])

  // Unexpected WS close is retried inside useGame.connect (ADR-0014).

  useEffect(() => {
    if (room?.status === 'closed') {
      disconnect()
      navigate('/rooms')
    }
  }, [room?.status, disconnect, navigate])

  if (!user) {
    return (
      <div className="min-h-dvh flex items-center justify-center bg-slate-950">
        <p className="text-slate-400 animate-pulse">restoring session...</p>
      </div>
    )
  }
  if (!room) {
    return (
      <div className="min-h-dvh flex items-center justify-center bg-slate-950">
        <p className="text-slate-400 animate-pulse">
          {connected ? 'joining room...' : 'connecting...'}
        </p>
      </div>
    )
  }

  if (room.status === 'in_game' && view) {
    return <GameTable room={room} view={view} />
  }

  const me = room.members.find((m) => m.user_id === user.id) ?? null
  const isHost = me?.is_host ?? false
  const allReady = room.members.length === 4 && room.members.every((m) => m.ready)
  const humans = room.members.filter((m) => !m.is_ai).length

  return (
    <div className="min-h-dvh bg-slate-950 p-4 max-w-2xl mx-auto">
      <Header title={room.name} />

      <section className="card mb-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="font-bold">Lobby</h2>
          <span className="text-xs text-slate-400">
            code <span className="font-mono text-teal-300 text-sm">{room.code}</span>
          </span>
        </div>

        <LobbyTeams roomId={room.id} members={room.members} isHost={isHost} userId={user.id} />

        <div className="flex flex-wrap gap-2 mt-3">
          {me && !me.ready ? (
            <button
              className="btn-primary"
              onClick={async () => {
                await api.setReady(room.id, true)
              }}
            >
              Ready up
            </button>
          ) : null}
          {me?.ready ? (
            <button
              className="btn-secondary"
              onClick={async () => {
                await api.setReady(room.id, false)
              }}
            >
              Unready
            </button>
          ) : null}
          {isHost && room.members.length < 4 ? (
            <>
              <button
                className="btn-secondary"
                onClick={async () => {
                  await api.fillAI(room.id)
                }}
              >
                Fill empty with AI
              </button>
              <select
                className="input w-auto text-sm"
                defaultValue=""
                onChange={async (e) => {
                  const d = e.target.value
                  if (d) {
                    await api.addAI(room.id, d)
                    e.target.value = ''
                  }
                }}
              >
                <option value="" disabled>
                  + Add AI
                </option>
                {['easy', 'medium', 'hard', 'expert', 'pro'].map((d) => (
                  <option key={d} value={d}>
                    AI ({d})
                  </option>
                ))}
              </select>
            </>
          ) : null}
          {isHost ? (
            <button
              className="btn-danger"
              onClick={async () => {
                if (!window.confirm('Delete this room? Everyone will be removed.')) return
                await api.deleteRoom(room.id)
                navigate('/rooms')
              }}
            >
              Delete room
            </button>
          ) : null}
          <button
            className="btn-danger"
            onClick={async () => {
              await api.leaveRoom(room.id)
              navigate('/rooms')
            }}
          >
            Leave
          </button>
        </div>
      </section>

      {isHost ? (
        <button
          className="btn-primary w-full text-base py-3 disabled:opacity-40"
          disabled={!allReady}
          onClick={() => startGame()}
          title={
            room.members.length < 4
              ? 'need 4 players (fill with AI)'
              : !allReady
                ? 'waiting for everyone to ready up'
                : ''
          }
        >
          {room.members.length < 4
            ? `waiting for players (${room.members.length}/4)`
            : allReady
              ? 'Start game'
              : 'waiting for ready'}
        </button>
      ) : (
        <p className="text-center text-sm text-slate-400">
          {humans < 4 ? 'waiting for the host to start...' : 'waiting for ready...'}
        </p>
      )}

      <ChatPanel />

      <p className="text-center text-xs text-slate-600 mt-4">
        {connected ? ' connected' : ' disconnected'} - share code{' '}
        <span className="font-mono text-teal-400">{room.code}</span> to invite
      </p>
    </div>
  )
}

function LobbyTeams({
  roomId,
  members,
  isHost,
  userId,
}: {
  roomId: string
  members: RoomMember[]
  isHost: boolean
  userId: string
}) {
  const [pick, setPick] = useState<number | null>(null)
  const drag = useRef<{ from: number; x: number; y: number; moved: boolean } | null>(null)

  const move = async (from: number, to: number) => {
    if (from === to) return
    await api.moveSeats(roomId, from, to)
    setPick(null)
  }

  const onPointerDown = (e: ReactPointerEvent<HTMLLIElement>, seat: number, occupied: boolean) => {
    if (!isHost || !occupied) return
    e.currentTarget.setPointerCapture(e.pointerId)
    drag.current = { from: seat, x: e.clientX, y: e.clientY, moved: false }
  }

  const onPointerMove = (e: ReactPointerEvent<HTMLLIElement>) => {
    const d = drag.current
    if (!d) return
    if (Math.hypot(e.clientX - d.x, e.clientY - d.y) > 8) d.moved = true
  }

  const onPointerUp = (e: ReactPointerEvent<HTMLLIElement>, seat: number, occupied: boolean) => {
    const d = drag.current
    drag.current = null
    if (!isHost) return
    if (d?.moved) {
      const el = document.elementFromPoint(e.clientX, e.clientY)
      const target = el?.closest('[data-seat]')
      const to = target ? Number(target.getAttribute('data-seat')) : NaN
      if (!Number.isNaN(to)) void move(d.from, to)
      return
    }
    if (pick === null) {
      if (occupied) setPick(seat)
      return
    }
    void move(pick, seat)
  }

  const renderSeat = (seat: number) => {
    const m = members.find((x) => x.seat === seat) ?? null
    const selected = pick === seat
    return (
      <li
        key={seat}
        data-seat={seat}
        className={`flex items-center justify-between bg-slate-800/60 rounded-lg px-3 py-2 touch-none ${
          selected ? 'ring-2 ring-teal-400' : ''
        } ${isHost ? 'cursor-grab' : ''}`}
        onPointerDown={(e) => {
          if ((e.target as HTMLElement).closest('button')) return
          onPointerDown(e, seat, Boolean(m))
        }}
        onPointerMove={onPointerMove}
        onPointerUp={(e) => onPointerUp(e, seat, Boolean(m))}
        onPointerCancel={() => {
          drag.current = null
        }}
      >
        <div className="flex items-center gap-2 min-w-0">
          <CardBack size="sm" />
          <span className="text-sm truncate">
            {m ? (
              <>
                {m.is_host ? '- ' : ''}
                {m.username}
                {m.is_ai ? ' -' : ''}
                {m.is_ai && m.ai_difficulty ? (
                  <span className="text-slate-500"> ({m.ai_difficulty})</span>
                ) : null}
              </>
            ) : (
              <span className="text-slate-500">empty seat</span>
            )}
          </span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          {m && !m.is_ai ? (
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${
                m.ready ? 'bg-emerald-500/20 text-emerald-300' : 'bg-slate-700 text-slate-400'
              }`}
            >
              {m.ready ? 'ready' : 'not ready'}
            </span>
          ) : null}
          {isHost && m && m.is_ai ? (
            <button
              className="text-xs text-rose-400 hover:text-rose-300"
              onClick={async (ev) => {
                ev.stopPropagation()
                await api.removeAI(roomId, m.user_id)
              }}
            >
              remove
            </button>
          ) : null}
          {isHost && m && !m.is_ai && m.user_id !== userId ? (
            <button
              className="text-xs text-rose-400 hover:text-rose-300"
              onClick={async (ev) => {
                ev.stopPropagation()
                await api.kick(roomId, m.user_id)
              }}
            >
              kick
            </button>
          ) : null}
        </div>
      </li>
    )
  }

  return (
    <>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <h3 className="text-xs uppercase tracking-wide text-teal-300 mb-2">Team A</h3>
          <ul className="flex flex-col gap-2">{[0, 2].map(renderSeat)}</ul>
        </div>
        <div>
          <h3 className="text-xs uppercase tracking-wide text-rose-300 mb-2">Team B</h3>
          <ul className="flex flex-col gap-2">{[1, 3].map(renderSeat)}</ul>
        </div>
      </div>
      {isHost ? (
        <p className="text-xs text-slate-500 mt-2">
          Drag a player onto another seat, or tap one seat then another, to rearrange.
        </p>
      ) : null}
    </>
  )
}
