// REST client with token storage and transparent access-token refresh.
// Tokens live in localStorage (single-device game client; see ADR-0008).

const BASE = '/api'

import type { Room } from '../protocol/messages'

let accessToken: string | null = localStorage.getItem('hokm.access')
let refreshToken: string | null = localStorage.getItem('hokm.refresh')

export function setTokens(access: string, refresh: string): void {
  accessToken = access
  refreshToken = refresh
  localStorage.setItem('hokm.access', access)
  localStorage.setItem('hokm.refresh', refresh)
}

export function clearTokens(): void {
  accessToken = null
  refreshToken = null
  localStorage.removeItem('hokm.access')
  localStorage.removeItem('hokm.refresh')
}

export function getAccessToken(): string | null {
  return accessToken
}

export interface User {
  id: string
  username: string
  email: string
  created_at: string
  is_guest: boolean
}

interface TokenPair {
  access_token: string
  refresh_token: string
  expires_in: number
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message)
  }
}

async function refreshTokens(): Promise<boolean> {
  if (!refreshToken) return false
  try {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    })
    if (!res.ok) return false
    const body = (await res.json()) as { tokens: TokenPair }
    setTokens(body.tokens.access_token, body.tokens.refresh_token)
    return true
  } catch {
    return false
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown,
  retry = true,
): Promise<T> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`

  const res = await fetch(BASE + path, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
  })

  if (res.status === 401 && retry && refreshToken) {
    if (await refreshTokens()) {
      return request<T>(method, path, body, false)
    }
    clearTokens()
  }

  if (!res.ok) {
    let code = 'error'
    let message = `request failed (${res.status})`
    try {
      const err = (await res.json()) as { code?: string; message?: string }
      if (err.code) code = err.code
      if (err.message) message = err.message
    } catch {
      // non-JSON error body
    }
    throw new ApiError(res.status, code, message)
  }
  return (await res.json()) as T
}

export const api = {
  register: (username: string, email: string, password: string) =>
    request<{ user: User; tokens: TokenPair }>('POST', '/auth/register', {
      username,
      email,
      password,
    }),
  login: (username: string, password: string) =>
    request<{ user: User; tokens: TokenPair }>('POST', '/auth/login', {
      username,
      password,
    }),
  me: () => request<{ user: User }>('GET', '/me'),
  listRooms: () => request<{ rooms: Room[] }>('GET', '/rooms'),
  createRoom: (name: string, visibility: string, roundCount: number, gameSpeed: string, chatEnabled: boolean) =>
    request<{ room: Room }>('POST', '/rooms', {
      name,
      visibility,
      round_count: roundCount,
      game_speed: gameSpeed,
      chat_enabled: chatEnabled,
    }),
  joinRoom: (code: string) =>
    request<{ room: Room }>('POST', '/rooms/join', { code }),
  getRoom: (id: string) => request<{ room: Room }>('GET', `/rooms/${id}`),
  leaveRoom: (id: string) => request<{ status: string }>('POST', `/rooms/${id}/leave`, {}),
  setReady: (id: string, ready: boolean) =>
    request<{ room: Room }>('POST', `/rooms/${id}/ready`, { ready }),
  kick: (id: string, user_id: string) =>
    request<{ room: Room }>('POST', `/rooms/${id}/kick`, { user_id }),
  addAI: (id: string, difficulty: string) =>
    request<{ room: Room }>('POST', `/rooms/${id}/ai`, { difficulty }),
  removeAI: (id: string, user_id: string) =>
    request<{ room: Room }>('POST', `/rooms/${id}/ai/remove`, { user_id }),
}
