import { useEffect } from 'react'
import type { ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { hasSessionTokens } from './api/client'
import { useAuth } from './state/auth'
import { Login, Register } from './pages/Auth'
import { Rooms } from './pages/Rooms'
import { Room } from './pages/Room'
import { Profile } from './pages/Profile'
import { Leaderboard } from './pages/Leaderboard'

function RequireAuth({ children }: { children: ReactNode }) {
  const user = useAuth((s) => s.user)
  const boot = useAuth((s) => s.boot)
  const booting = useAuth((s) => s.booting)
  const error = useAuth((s) => s.error)

  useEffect(() => {
    void boot()
  }, [boot])

  if (booting) {
    return (
      <div className="min-h-dvh flex items-center justify-center bg-slate-950">
        <p className="text-slate-400 animate-pulse">restoring session...</p>
      </div>
    )
  }

  if (!user) {
    if (hasSessionTokens()) {
      return (
        <div className="min-h-dvh flex flex-col items-center justify-center gap-3 bg-slate-950 p-4">
          <p className="text-slate-300 text-center">{error ?? 'could not reach the server'}</p>
          <button className="btn-primary" type="button" onClick={() => void boot()}>
            Retry
          </button>
        </div>
      )
    }
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Navigate to="/rooms" replace />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route
          path="/rooms"
          element={
            <RequireAuth>
              <Rooms />
            </RequireAuth>
          }
        />
        <Route
          path="/room/:id"
          element={
            <RequireAuth>
              <Room />
            </RequireAuth>
          }
        />
        <Route
          path="/profile"
          element={
            <RequireAuth>
              <Profile />
            </RequireAuth>
          }
        />
        <Route
          path="/leaderboard"
          element={
            <RequireAuth>
              <Leaderboard />
            </RequireAuth>
          }
        />
        <Route path="*" element={<Navigate to="/rooms" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
