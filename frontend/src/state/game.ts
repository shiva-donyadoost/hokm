import { create } from 'zustand'
import { ensureFreshAccess, getAccessToken } from '../api/client'
import {
  diagError,
  diagInfo,
  diagWarn,
  logPlayAttempt,
} from '../diagnostics/clientLog'
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
// Unexpected close triggers bounded reconnect (ADR-0014).

interface GameState {
  ws: WebSocket | null
  connected: boolean
  room: Room | null
  view: SeatView | null
  chat: ChatMessage[]
  lastError: string | null
  /** Latest public WS EVENTS envelope for presentation (ADR-0019). */
  lastEngineEvent: { name: string; payload: unknown; seq: number } | null
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

/** Intentional disconnect suppresses auto-reconnect. */
let intentionalClose = false
let wantRoomId: string | null = null
let reconnectAttempt = 0
let reconnectTimer: ReturnType<typeof setTimeout> | null = null

function clearReconnectTimer(): void {
  if (reconnectTimer !== null) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }
}

function wsLabel(ws: WebSocket | null): string {
  if (!ws) return 'none'
  return String(ws.readyState)
}

export const useGame = create<GameState>((set, get) => ({
  ws: null,
  connected: false,
  room: null,
  view: null,
  chat: [],
  lastError: null,
  lastEngineEvent: null,

  connect: async (roomId) => {
    intentionalClose = false
    wantRoomId = roomId
    clearReconnectTimer()
    const prev = get().ws
    if (prev) {
      prev.onclose = null
      prev.onerror = null
      prev.onmessage = null
      prev.close()
    }
    await ensureFreshAccess()
    const token = getAccessToken()
    if (!token) {
      diagWarn('ws', 'connect_no_token', { roomId })
      return
    }
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const ws = new WebSocket(`${proto}://${location.host}/api/ws?token=${token}`)
    diagInfo('ws', 'connecting', { roomId, attempt: reconnectAttempt })

    ws.onopen = () => {
      reconnectAttempt = 0
      set({ connected: true, lastError: null })
      // Keep room/view across reconnect so the table does not flash lobby;
      // server will refresh both on SUBSCRIBE.
      localStorage.setItem('hokm.lastRoom', roomId)
      ws.send(JSON.stringify({ type: Cmd.Subscribe, id: nextId(), payload: { room_id: roomId } } satisfies Envelope))
      diagInfo('ws', 'open_subscribed', { roomId })
    }
    ws.onmessage = (e) => {
      let env: Envelope
      try {
        env = JSON.parse(e.data) as Envelope
      } catch {
        diagWarn('ws', 'bad_json')
        return
      }
      switch (env.type) {
        case Msg.Room:
          set({ room: env.payload as Room })
          break
        case Msg.State: {
          const view = env.payload as SeatView
          set({ view })
          diagInfo('ws', 'state', {
            phase: view.phase,
            turn: view.turn,
            you: view.you,
            trickLen: view.current_trick?.length ?? 0,
            handLen: view.your_hand?.length ?? 0,
            deadlineKind: view.deadline_kind,
          })
          break
        }
        case Msg.Chat:
          set((s) => ({ chat: [...s.chat, env.payload as ChatMessage] }))
          break
        case Msg.Events: {
          const name = env.name ?? ''
          let payload: unknown = env.payload
          // Backend may send payload as already-parsed object or JSON string.
          if (typeof payload === 'string') {
            try { payload = JSON.parse(payload) } catch { /* keep string */ }
          }
          set((st) => ({
            lastEngineEvent: {
              name,
              payload,
              seq: (st.lastEngineEvent?.seq ?? 0) + 1,
            },
          }))
          break
        }
        case Msg.Error: {
          const p = env.payload as { code?: string; message?: string }
          const silent = p.code === 'must_follow_suit' || p.code === 'not_your_turn' ||
            p.code === 'card_not_owned' || p.code === 'trick_not_full' ||
            p.code === 'wrong_phase' || p.code === 'not_hakem' || p.code === 'invalid_trump'
          diagWarn('ws', 'error', { code: p.code, message: p.message, silent })
          if (!silent) {
            set({ lastError: p.message ?? 'unknown error' })
          }
          break
        }
        default:
          break
      }
    }
    ws.onclose = () => {
      set({ connected: false })
      diagWarn('ws', 'close', { intentional: intentionalClose, wantRoomId })
      if (intentionalClose || !wantRoomId) return
      reconnectAttempt += 1
      if (reconnectAttempt > 8) {
        diagError('ws', 'reconnect_gave_up', { attempts: reconnectAttempt })
        set({ lastError: 'connection lost - refresh the page' })
        return
      }
      const delay = Math.min(8000, 500 * 2 ** Math.min(reconnectAttempt - 1, 4))
      diagInfo('ws', 'reconnect_scheduled', { delay, attempt: reconnectAttempt })
      clearReconnectTimer()
      reconnectTimer = setTimeout(() => {
        if (!wantRoomId || intentionalClose) return
        void get().connect(wantRoomId)
      }, delay)
    }
    ws.onerror = () => {
      diagWarn('ws', 'error_event')
      set({ connected: false })
    }
    set({ ws })
  },

  disconnect: () => {
    intentionalClose = true
    wantRoomId = null
    clearReconnectTimer()
    reconnectAttempt = 0
    const ws = get().ws
    if (ws) {
      ws.onclose = null
      ws.onerror = null
      ws.onmessage = null
      ws.close()
    }
    set({ ws: null, connected: false, room: null, view: null, chat: [], lastError: null, lastEngineEvent: null })
    localStorage.removeItem('hokm.lastRoom')
    diagInfo('ws', 'disconnect_intentional')
  },

  startGame: () => {
    const g = get()
    const roomId = g.room?.id
    if (g.ws && roomId && g.ws.readyState === WebSocket.OPEN) {
      g.ws.send(JSON.stringify({ type: Cmd.StartGame, id: nextId(), payload: { room_id: roomId } } satisfies Envelope))
      diagInfo('cmd', 'start_game', { roomId })
    } else {
      diagWarn('cmd', 'start_game_blocked', { ws: wsLabel(g.ws) })
    }
  },

  replayGame: () => {
    const g = get()
    const roomId = g.room?.id
    if (g.ws && roomId && g.ws.readyState === WebSocket.OPEN) {
      g.ws.send(JSON.stringify({ type: Cmd.ReplayGame, id: nextId(), payload: { room_id: roomId } } satisfies Envelope))
      diagInfo('cmd', 'replay_game', { roomId })
    } else {
      diagWarn('cmd', 'replay_game_blocked', { ws: wsLabel(g.ws) })
    }
  },

  selectTrump: (suit) => {
    const g = get()
    const roomId = g.room?.id
    if (g.ws && roomId && g.ws.readyState === WebSocket.OPEN) {
      g.ws.send(JSON.stringify({ type: Cmd.SelectTrump, id: nextId(), payload: { room_id: roomId, suit } } satisfies Envelope))
      logPlayAttempt({ action: 'select_trump', ok: true, reason: 'sent', suit, wsState: wsLabel(g.ws) })
    } else {
      logPlayAttempt({ action: 'select_trump', ok: false, reason: 'ws_not_open', suit, wsState: wsLabel(g.ws) })
      set({ lastError: 'not connected - cannot select trump' })
    }
  },

  playCard: (card) => {
    const g = get()
    const roomId = g.room?.id
    if (g.ws && roomId && g.ws.readyState === WebSocket.OPEN) {
      g.ws.send(JSON.stringify({ type: Cmd.PlayCard, id: nextId(), payload: { room_id: roomId, card } } satisfies Envelope))
      // GameTable also logs; this catches store-level sends.
      diagInfo('cmd', 'play_card_sent', { card, roomId })
    } else {
      logPlayAttempt({
        action: 'play_card', ok: false, reason: 'ws_not_open', card, wsState: wsLabel(g.ws),
      })
      set({ lastError: 'not connected - cannot play card' })
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
