import { useEffect } from 'react'
import type { ReactNode } from 'react'
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './state/auth'
import { Login, Register } from './pages/Auth'
import { Rooms } from './pages/Rooms'
import { Room } from './pages/Room'
import { Profile } from './pages/Profile'
import { Leaderboard } from './pages/Leaderboard'

function RequireAuth({ children }: { children: ReactNode }) {
  const user = useAuth((s) => s.user)
  const loadProfile = useAuth((s) => s.loadProfile)
  const loading = useAuth((s) => s.loading)

  useEffect(() => {
    // Restore session from a persisted access token on first load.
    if (!user) void loadProfile()
  }, [user, loadProfile])

  if (!user && loading) return null
  // Not authoritative: loadProfile clears user when the token is dead.
  if (!user && !loading) {
    // Give the restored-session effect a moment by checking token presence.
    if (!localStorage.getItem('hokm.access')) return <Navigate to="/login" replace />
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
