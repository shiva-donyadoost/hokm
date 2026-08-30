import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { api, ApiError } from '../api/client'
import { useAuth } from '../state/auth'
import type { Room } from '../protocol/messages'

export function Rooms() {
  const user = useAuth((s) => s.user)
  const navigate = useNavigate()
  const [rooms, setRooms] = useState<Room[]>([])
  const [name, setName] = useState('')
  const [visibility, setVisibility] = useState('public')
  const [roundCount, setRoundCount] = useState(1)
  const [gameSpeed, setGameSpeed] = useState('medium')
  const [chatEnabled, setChatEnabled] = useState(true)
  const [joinCode, setJoinCode] = useState('')
  const [error, setError] = useState<string | null>(null)
  const lastRoom = localStorage.getItem('hokm.lastRoom')

  useEffect(() => {
    let live = true
    const load = async () => {
      try {
        const res = await api.listRooms()
        if (live) setRooms(res.rooms)
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) navigate('/login')
      }
    }
    void load()
    const t = setInterval(load, 3000)
    return () => {
      live = false
      clearInterval(t)
    }
  }, [navigate])

  return (
    <div className="min-h-dvh bg-slate-950 p-4 max-w-lg mx-auto">
      <Header title="Rooms" />

      {/* Return-to-game banner: refresh persistence (impliment.md §25) */}
      {lastRoom ? (
        <div className="card mb-4 flex items-center justify-between !py-3">
          <span className="text-sm text-slate-300">You have an active table.</span>
          <button className="btn-primary" onClick={() => navigate(`/room/${lastRoom}`)}>
            Return to game
          </button>
        </div>
      ) : null}

      {/* Create room */}
      <section className="card mb-4">
        <h2 className="font-bold mb-2">Create a room</h2>
        <div className="flex flex-col gap-2">
          <input
            className="input"
            placeholder="Room name"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <div className="flex gap-2">
            <select
              className="input flex-1"
              value={visibility}
              onChange={(e) => setVisibility(e.target.value)}
              aria-label="visibility"
            >
              <option value="public">Public</option>
              <option value="private">Private (code only)</option>
            </select>
            <select
              className="input flex-1"
              value={roundCount}
              onChange={(e) => setRoundCount(Number(e.target.value))}
              aria-label="rounds to win"
            >
              {[1, 3, 5].map((n) => (
                <option key={n} value={n}>
                  {n} round{n > 1 ? 's' : ''} to win
                </option>
              ))}
            </select>
          </div>
          <div className="flex gap-2">
            <select
              className="input flex-1"
              value={gameSpeed}
              onChange={(e) => setGameSpeed(e.target.value)}
              aria-label="game speed"
            >
              <option value="fast">Fast (5s)</option>
              <option value="medium">Medium (10s)</option>
              <option value="slow">Slow (15s)</option>
            </select>
            <label className="input flex items-center gap-2 flex-1 text-sm text-slate-300">
              <input
                type="checkbox"
                checked={chatEnabled}
                onChange={(e) => setChatEnabled(e.target.checked)}
              />
              chat
            </label>
          </div>
          <button
            className="btn-primary"
            onClick={async () => {
              try {
                const res = await api.createRoom(name, visibility, roundCount, gameSpeed, chatEnabled)
                navigate(`/room/${res.room.id}`)
              } catch (e) {
                setError(e instanceof Error ? e.message : 'failed')
              }
            }}
          >
            Create
          </button>
        </div>
      </section>

      {/* Join by code */}
      <section className="card mb-4">
        <h2 className="font-bold mb-2">Join with code</h2>
        <div className="flex gap-2">
          <input
            className="input flex-1 uppercase tracking-widest"
            placeholder="ABC123"
            maxLength={6}
            value={joinCode}
            onChange={(e) => setJoinCode(e.target.value.toUpperCase())}
          />
          <button
            className="btn-primary"
            onClick={async () => {
              try {
                const res = await api.joinRoom(joinCode)
                navigate(`/room/${res.room.id}`)
              } catch (e) {
                setError(e instanceof Error ? e.message : 'failed')
              }
            }}
          >
            Join
          </button>
        </div>
      </section>

      {error ? <p className="text-rose-400 text-sm mb-3">{error}</p> : null}

      {/* Public rooms */}
      <section>
        <h2 className="font-bold mb-2 text-slate-300">Open rooms</h2>
        {rooms.length === 0 ? (
          <p className="text-slate-500 text-sm">No public rooms yet — create one!</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {rooms.map((r) => (
              <li key={r.id} className="card flex items-center justify-between">
                <div>
                  <p className="font-semibold">{r.name}</p>
                  <p className="text-xs text-slate-400">
                    {r.members.length}/4 players · code {r.code}
                  </p>
                </div>
                <button
                  className="btn-secondary"
                  onClick={async () => {
                    try {
                      await api.joinRoom(r.code)
                      navigate(`/room/${r.id}`)
                    } catch (e) {
                      setError(e instanceof Error ? e.message : 'failed')
                    }
                  }}
                >
                  Join
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <p className="text-center text-xs text-slate-600 mt-6">
        signed in as {user?.username}
      </p>
    </div>
  )
}

export function Header({ title }: { title: string }) {
  const user = useAuth((s) => s.user)
  const logout = useAuth((s) => s.logout)
  const navigate = useNavigate()
  return (
    <div className="flex items-center justify-between mb-4">
      <Link to="/rooms" className="text-xl font-black">
        HOKM<span className="text-teal-400">.</span>
      </Link>
      <div className="flex items-center gap-2 text-sm">
        <span className="text-slate-300">{title}</span>
        <Link to="/profile" className="text-teal-400 hover:underline">
          {user?.username}
        </Link>
        <button
          className="text-slate-500 hover:text-slate-300"
          onClick={() => {
            logout()
            navigate('/login')
          }}
        >
          logout
        </button>
      </div>
    </div>
  )
}
