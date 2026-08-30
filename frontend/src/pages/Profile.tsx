import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../state/auth'
import { Header } from './Rooms'

export function Profile() {
  const user = useAuth((s) => s.user)
  const loadProfile = useAuth((s) => s.loadProfile)

  useEffect(() => {
    void loadProfile()
  }, [loadProfile])

  if (!user) {
    return (
      <div className="min-h-dvh bg-slate-950 p-4 max-w-lg mx-auto">
        <Header title="Profile" />
        <p className="text-slate-400">Not signed in.</p>
      </div>
    )
  }

  return (
    <div className="min-h-dvh bg-slate-950 p-4 max-w-lg mx-auto">
      <Header title="Profile" />
      <section className="card">
        <div className="flex items-center gap-4 mb-4">
          <div className="w-16 h-16 rounded-full bg-teal-700 flex items-center justify-center text-2xl font-black">
            {user.username.slice(0, 2).toUpperCase()}
          </div>
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
          <dd>coming in Phase 13</dd>
        </dl>
      </section>
      <p className="text-center mt-4">
        <Link className="text-sm text-slate-500 hover:text-slate-300" to="/rooms">
          - back to rooms
        </Link>
      </p>
    </div>
  )
}
