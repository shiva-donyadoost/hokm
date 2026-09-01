import { create } from 'zustand'
import { ensureFreshAccess, getAccessToken } from '../api/client'
import {
  Cmd,
  Msg,
  type ChatMessage,
  type Envelope,
  type Room,
  type SeatView,
  type Suit,
} from '../protocol/messages'

// GameStore holds the live WS connection state for one room: the lobby
// snapshot, the per-seat game view, and connection status. The server is
// the sole authority; this store only renders what it receives (ADR-0004).

interface GameState {
  ws: WebSocket | null
  connected: boolean
  room: Room | null
  view: SeatView | null
  chat: ChatMessage[]
  lastError: string | null
  connect: (roomId: string) => Promise<void>
  disconnect: () => void
  startGame: () => void
  replayGame: () => void
  selectTrump: (suit: Suit) => void
  playCard: (card: { suit: Suit; rank: number }) => void
  sendChat: (body: string) => void
}

let msgId = 0
function nextId(): string {
  msgId++
  return `m${msgId}`
}

export const useGame = create<GameState>((set, get) => ({
  ws: null,
  connected: false,
  room: null,
  view: null,
  chat: [],
  lastError: null,

  connect: async (roomId) => {
    get().disconnect()
    await ensureFreshAccess()
    const token = getAccessToken()
    if (!token) return
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${proto}://${location.host}/api/ws?token=${token}`)

    ws.onopen = () => {
      set({ connected: true, room: null, view: null, chat: [], lastError: null })
      localStorage.setItem('hokm.lastRoom', roomId)
      ws.send(JSON.stringify({ type: Cmd.Subscribe, id: nextId(), payload: { room_id: roomId } } satisfies Envelope))
    }
    ws.onmessage = (e) => {
      let env: Envelope
      try {
        env = JSON.parse(e.data) as Envelope
      } catch {
        return
      }
      switch (env.type) {
        case Msg.Room:
          set({ room: env.payload as Room })
          break
        case Msg.State:
          set({ view: env.payload as SeatView })
          break
        case Msg.Chat:
          set((s) => ({ chat: [...s.chat, env.payload as ChatMessage] }))
          break
        case Msg.Error: {
          const p = env.payload as { code?: string; message?: string }
          const silent = p.code === 'must_follow_suit' || p.code === 'not_your_turn' ||
            p.code === 'card_not_owned' || p.code === 'trick_not_full' ||
            p.code === 'wrong_phase' || p.code === 'not_hakem' || p.code === 'invalid_trump'
          if (!silent) {
            set({ lastError: p.message ?? 'unknown error' })
          }
          break
        }
        default:
          break
      }
    }
    ws.onclose = () => set({ connected: false })
    ws.onerror = () => set({ connected: false })
    set({ ws })
  },

  disconnect: () => {
    const ws = get().ws
    if (ws) {
      ws.onclose = null
      ws.close()
    }
    set({ ws: null, connected: false, room: null, view: null, chat: [], lastError: null })
    localStorage.removeItem('hokm.lastRoom')
  },

  startGame: () => {
    const g = get()
    const roomId = g.room?.id
    if (g.ws && roomId && g.ws.readyState === WebSocket.OPEN) {
      g.ws.send(JSON.stringify({ type: Cmd.StartGame, id: nextId(), payload: { room_id: roomId } } satisfies Envelope))
    }
  },

  replayGame: () => {
    const g = get()
    const roomId = g.room?.id
    if (g.ws && roomId && g.ws.readyState === WebSocket.OPEN) {
      g.ws.send(JSON.stringify({ type: Cmd.ReplayGame, id: nextId(), payload: { room_id: roomId } } satisfies Envelope))
    }
  },

  selectTrump: (suit) => {
    const g = get()
    const roomId = g.room?.id
    if (g.ws && roomId && g.ws.readyState === WebSocket.OPEN) {
      g.ws.send(JSON.stringify({ type: Cmd.SelectTrump, id: nextId(), payload: { room_id: roomId, suit } } satisfies Envelope))
    }
  },

  playCard: (card) => {
    const g = get()
    const roomId = g.room?.id
    if (g.ws && roomId && g.ws.readyState === WebSocket.OPEN) {
      g.ws.send(JSON.stringify({ type: Cmd.PlayCard, id: nextId(), payload: { room_id: roomId, card } } satisfies Envelope))
    }
  },

  sendChat: (body) => {
    const g = get()
    const roomId = g.room?.id
    if (g.ws && roomId && g.ws.readyState === WebSocket.OPEN) {
      g.ws.send(JSON.stringify({ type: Cmd.Chat, id: nextId(), payload: { room_id: roomId, body } } satisfies Envelope))
    }
  },
}))
