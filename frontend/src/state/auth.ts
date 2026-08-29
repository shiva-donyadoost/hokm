import { create } from 'zustand'
import { api, clearTokens, setTokens, type User } from '../api/client'

interface AuthState {
  user: User | null
  loading: boolean
  error: string | null
  login: (username: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  logout: () => void
  loadProfile: () => Promise<void>
}

export const useAuth = create<AuthState>((set) => ({
  user: null,
  loading: false,
  error: null,

  login: async (username, password) => {
    set({ loading: true, error: null })
    try {
      const res = await api.login(username, password)
      setTokens(res.tokens.access_token, res.tokens.refresh_token)
      set({ user: res.user, loading: false })
    } catch (e) {
      set({ loading: false, error: e instanceof Error ? e.message : 'login failed' })
      throw e
    }
  },

  register: async (username, email, password) => {
    set({ loading: true, error: null })
    try {
      const res = await api.register(username, email, password)
      setTokens(res.tokens.access_token, res.tokens.refresh_token)
      set({ user: res.user, loading: false })
    } catch (e) {
      set({ loading: false, error: e instanceof Error ? e.message : 'registration failed' })
      throw e
    }
  },

  logout: () => {
    clearTokens()
    set({ user: null })
  },

  loadProfile: async () => {
    try {
      const res = await api.me()
      set({ user: res.user })
    } catch {
      set({ user: null })
    }
  },
}))
