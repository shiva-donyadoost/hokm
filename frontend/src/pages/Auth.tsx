import { useState } from 'react'
import type { ReactNode } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../state/auth'

export function Login() {
  const login = useAuth((s) => s.login)
  const loading = useAuth((s) => s.loading)
  const error = useAuth((s) => s.error)
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')

  return (
    <AuthShell title="Welcome back" subtitle="Sign in to play HOKM">
      <form
        className="flex flex-col gap-3"
        onSubmit={async (e) => {
          e.preventDefault()
          try {
            await login(username, password)
            navigate('/rooms')
          } catch {
            /* error shown by store */
          }
        }}
      >
        <input
          className="input"
          placeholder="Username"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="username"
          required
        />
        <input
          className="input"
          type="password"
          placeholder="Password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="current-password"
          required
        />
        {error ? <p className="text-rose-400 text-sm">{error}</p> : null}
        <button className="btn-primary" disabled={loading}>
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
      <p className="text-sm text-slate-400 mt-4">
        No account?{' '}
        <Link className="text-teal-400 hover:underline" to="/register">
          Create one
        </Link>
      </p>
    </AuthShell>
  )
}

export function Register() {
  const register = useAuth((s) => s.register)
  const loading = useAuth((s) => s.loading)
  const error = useAuth((s) => s.error)
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  return (
    <AuthShell title="Create account" subtitle="Join the table">
      <form
        className="flex flex-col gap-3"
        onSubmit={async (e) => {
          e.preventDefault()
          try {
            await register(username, email, password)
            navigate('/rooms')
          } catch {
            /* error shown by store */
          }
        }}
      >
        <input
          className="input"
          placeholder="Username (3-24 chars)"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          autoComplete="username"
          required
        />
        <input
          className="input"
          type="email"
          placeholder="Email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          autoComplete="email"
          required
        />
        <input
          className="input"
          type="password"
          placeholder="Password (8+ chars)"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="new-password"
          required
        />
        {error ? <p className="text-rose-400 text-sm">{error}</p> : null}
        <button className="btn-primary" disabled={loading}>
          {loading ? 'Creating…' : 'Create account'}
        </button>
      </form>
      <p className="text-sm text-slate-400 mt-4">
        Already registered?{' '}
        <Link className="text-teal-400 hover:underline" to="/login">
          Sign in
        </Link>
      </p>
    </AuthShell>
  )
}

export function AuthShell({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle: string
  children: ReactNode
}) {
  return (
    <div className="min-h-dvh flex items-center justify-center bg-gradient-to-b from-table-900 to-slate-950 p-4">
      <div className="w-full max-w-sm bg-slate-900/80 border border-slate-800 rounded-2xl p-6">
        <h1 className="text-3xl font-black tracking-tight mb-1">
          HOKM<span className="text-teal-400">.</span>
        </h1>
        <p className="text-slate-400 text-sm mb-6">{title} — {subtitle}</p>
        {children}
      </div>
    </div>
  )
}
