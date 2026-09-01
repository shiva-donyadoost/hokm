import { create } from 'zustand'
import { ApiError, api, clearTokens, ensureFreshAccess, hasSessionTokens, setTokens, type User } from '../api/client'

interface AuthState {
  user: User | null
  loading: boolean
  booting: boolean
  error: string | null
  login: (username: string, password: string) => Promise<void>
  register: (username: string, email: string, password: string) => Promise<void>
  logout: () => void
  boot: () => Promise<void>
  loadProfile: () => Promise<void>
}

export const useAuth = create<AuthState>((set) => ({
  user: null,
  loading: false,
  booting: hasSessionTokens(),
  error: null,

  login: async (username, password) => {
    set({ loading: true, error: null })
    try {
      const res = await api.login(username, password)
      setTokens(res.tokens.access_token, res.tokens.refresh_token)
      set({ user: res.user, loading: false, booting: false })
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
      set({ user: res.user, loading: false, booting: false })
    } catch (e) {
      set({ loading: false, error: e instanceof Error ? e.message : 'registration failed' })
      throw e
    }
  },

  logout: () => {
    clearTokens()
    set({ user: null, booting: false })
  },

  boot: async () => {
    if (!hasSessionTokens()) {
      set({ booting: false })
      return
    }
    set({ booting: true, error: null })
    try {
      await ensureFreshAccess()
      const res = await api.me()
      set({ user: res.user, booting: false, error: null })
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        clearTokens()
        set({ user: null, booting: false })
        return
      }
      set({
        booting: false,
        error: e instanceof Error ? e.message : 'could not restore session',
      })
    }
  },

  loadProfile: async () => {
    try {
      await ensureFreshAccess()
      const res = await api.me()
      set({ user: res.user })
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        clearTokens()
        set({ user: null })
      }
    }
  },
}))
