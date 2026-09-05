import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, type StatsEntry } from '../api/client'
import { useAuth } from '../state/auth'
import { PlayerAvatar, avatarSeed } from '../components/PlayerAvatar'
import { Header } from './Rooms'

export function Profile() {
  const user = useAuth((s) => s.user)
  const loadProfile = useAuth((s) => s.loadProfile)
  const [stats, setStats] = useState<StatsEntry | null>(null)

  useEffect(() => {
    void loadProfile()
  }, [loadProfile])

  useEffect(() => {
    let live = true
    void api.myStats().then((res) => {
      if (live) setStats(res.stats)
    }).catch(() => {
      if (live) setStats(null)
    })
    return () => { live = false }
  }, [])

  if (!user) {
    return (
      <div className="min-h-dvh bg-slate-950 p-4 max-w-lg mx-auto">
        <Header title="Profile" />
        <p className="text-slate-400">Not signed in.</p>
      </div>
    )
  }

  const played = stats?.games_played ?? 0
  const winRate = played > 0 ? Math.round(((stats?.wins ?? 0) / played) * 100) : 0

  return (
    <div className="min-h-dvh bg-slate-950 p-4 max-w-lg mx-auto">
      <Header title="Profile" />
      <section className="card">
        <div className="flex items-center gap-4 mb-4">
          <PlayerAvatar seed={avatarSeed(user.id, user.username)} name={user.username} size="lg" />
          <div>
            <p className="text-lg font-bold">{user.username}</p>
            <p className="text-sm text-slate-400">{user.email}</p>
          </div>
        </div>
        <dl className="text-sm grid grid-cols-2 gap-2">
          <dt className="text-slate-500">Member since</dt>
          <dd>{new Date(user.created_at).toLocaleDateString()}</dd>
          <dt className="text-slate-500">Account</dt>
          <dd>{user.is_guest ? 'guest' : 'registered'}</dd>
          <dt className="text-slate-500">Rating</dt>
          <dd>{stats?.rating ?? 1000}</dd>
          <dt className="text-slate-500">Games</dt>
          <dd>{played}</dd>
          <dt className="text-slate-500">Wins</dt>
          <dd>{stats?.wins ?? 0}</dd>
          <dt className="text-slate-500">Losses</dt>
          <dd>{stats?.losses ?? 0}</dd>
          <dt className="text-slate-500">Win rate</dt>
          <dd>{played ? winRate + '%' : '-'}</dd>
          <dt className="text-slate-500">Rounds won</dt>
          <dd>{stats?.rounds_won ?? 0}</dd>
          <dt className="text-slate-500">Rounds lost</dt>
          <dd>{stats?.rounds_lost ?? 0}</dd>
        </dl>
      </section>
      <p className="text-center mt-4 flex justify-center gap-4">
        <Link className="text-sm text-teal-400 hover:text-teal-200" to="/leaderboard">
          leaderboard
        </Link>
        <Link className="text-sm text-slate-500 hover:text-slate-300" to="/rooms">
          back to rooms
        </Link>
      </p>
    </div>
  )
}
