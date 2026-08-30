import { useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../state/auth'
import { useGame } from '../state/game'
import { GameTable } from '../components/GameTable'
import { ChatPanel } from '../components/ChatPanel'
import { CardBack } from '../components/Card'
import { Header } from './Rooms'

// Room is the lobby + live game surface. It subscribes once on mount and
// renders whichever stage the server reports (ADR-0004: server authority).
export function Room() {
  const { id = '' } = useParams()
  const user = useAuth((s) => s.user)
  const navigate = useNavigate()
  const connect = useGame((s) => s.connect)
  const disconnect = useGame((s) => s.disconnect)
  const startGame = useGame((s) => s.startGame)
  const ws = useGame((s) => s.ws)
  const connected = useGame((s) => s.connected)
  const room = useGame((s) => s.room)
  const view = useGame((s) => s.view)

  useEffect(() => {
    if (!user) {
      navigate('/login')
      return
    }
    connect(id)
    return () => disconnect()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, user])

  // Re-subscribe after reconnects.
  useEffect(() => {
    if (connected && ws && ws.readyState === WebSocket.OPEN) return
  }, [connected, ws])

  if (!user) {
    navigate('/login')
    return null
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
    <div className="min-h-dvh bg-slate-950 p-4 max-w-lg mx-auto">
      <Header title={room.name} />

      <section className="card mb-4">
        <div className="flex items-center justify-between mb-3">
          <h2 className="font-bold">Lobby</h2>
          <span className="text-xs text-slate-400">
            code <span className="font-mono text-teal-300 text-sm">{room.code}</span>
          </span>
        </div>

        <ul className="flex flex-col gap-2">
          {[0, 1, 2, 3].map((seat) => {
            const m = room.members.find((x) => x.seat === seat) ?? null
            return (
              <li
                key={seat}
                className="flex items-center justify-between bg-slate-800/60 rounded-lg px-3 py-2"
              >
                <div className="flex items-center gap-2">
                  <CardBack size="sm" />
                  <span className="text-sm">
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
                <div className="flex items-center gap-2">
                  {m && !m.is_ai ? (
                    <span
                      className={`text-xs px-2 py-0.5 rounded-full ${
                        m.ready
                          ? 'bg-emerald-500/20 text-emerald-300'
                          : 'bg-slate-700 text-slate-400'
                      }`}
                    >
                      {m.ready ? 'ready' : 'not ready'}
                    </span>
                  ) : null}
                  {isHost && m && m.is_ai ? (
                    <button
                      className="text-xs text-rose-400 hover:text-rose-300"
                      onClick={async () => {
                        await api.removeAI(room.id, m.user_id)
                      }}
                    >
                      remove
                    </button>
                  ) : null}
                  {isHost && m && !m.is_ai && m.user_id !== user.id ? (
                    <button
                      className="text-xs text-rose-400 hover:text-rose-300"
                      onClick={async () => {
                        await api.kick(room.id, m.user_id)
                      }}
                    >
                      kick
                    </button>
                  ) : null}
                </div>
              </li>
            )
          })}
        </ul>

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
