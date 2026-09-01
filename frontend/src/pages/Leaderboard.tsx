import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api, type StatsEntry } from '../api/client'
import { Header } from './Rooms'

export function Leaderboard() {
  const [entries, setEntries] = useState<StatsEntry[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let live = true
    void api.leaderboard().then((res) => {
      if (live) setEntries(res.entries ?? [])
    }).catch((e) => {
      if (live) setError(e instanceof Error ? e.message : 'failed to load')
    })
    return () => { live = false }
  }, [])

  return (
    <div className="min-h-dvh bg-slate-950 p-4 max-w-lg mx-auto">
      <Header title="Leaderboard" />
      <section className="card">
        <h2 className="font-bold mb-3">Top players by wins</h2>
        {error ? <p className="text-rose-400 text-sm">{error}</p> : null}
        {entries.length === 0 && !error ? (
          <p className="text-slate-500 text-sm">No matches recorded yet.</p>
        ) : (
          <ol className="flex flex-col gap-2">
            {entries.map((e, i) => (
              <li key={e.user_id} className="flex items-center justify-between bg-slate-800/60 rounded-lg px-3 py-2">
                <div className="flex items-center gap-3">
                  <span className="text-slate-500 w-6 text-right font-mono">{i + 1}</span>
                  <div>
                    <p className="font-semibold">{e.username || e.user_id}</p>
                    <p className="text-xs text-slate-400">
                      {e.wins}W / {e.losses}L - {e.games_played} games
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <span className="text-teal-300 font-bold">{e.wins}W</span>
                  <p className="text-xs text-slate-500">{e.rating}</p>
                </div>
              </li>
            ))}
          </ol>
        )}
      </section>
      <p className="text-center mt-4">
        <Link className="text-sm text-slate-500 hover:text-slate-300" to="/rooms">
          back to rooms
        </Link>
      </p>
    </div>
  )
}
