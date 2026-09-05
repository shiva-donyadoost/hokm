// Wire protocol mirroring backend/internal/ws and internal/game types.
// Kept in sync manually; verified by E2E tests (ADR-0003).

export type Suit = 'spades' | 'hearts' | 'diamonds' | 'clubs'

export interface Card {
  suit: Suit
  rank: number // 2..14 (14 = ace)
}

export interface PlayedCard {
  seat: number
  card: Card
}

export interface CompletedTrick {
  number: number
  lead_suit: Suit
  cards: PlayedCard[]
  winner: number
  winner_team: number
}

export interface RoundResult {
  number: number
  winner_team: number
}

export interface SeatView {
  phase: string
  round_number: number
  hakem: number
  trump?: Suit
  turn: number
  you: number
  your_hand?: Card[]
  hand_counts: [number, number, number, number]
  current_trick?: PlayedCard[]
  last_trick?: CompletedTrick | null
  tricks_this_round: [number, number]
  rounds_won: [number, number]
  round_history?: RoundResult[]
  deadline_unix_ms?: number
  deadline_kind?: string
  match_over: boolean
}

export interface RoomMember {
  user_id: string
  username: string
  avatar_seed?: string
  seat: number
  ready: boolean
  is_host: boolean
  is_ai: boolean
  ai_difficulty?: string
}

export interface Room {
  id: string
  code: string
  name: string
  visibility: 'public' | 'private'
  host_id: string
  members: RoomMember[]
  status: string
  created_at: string
  round_count: number
  game_speed: string
  chat_enabled: boolean
}

// --- envelopes ---

export interface Envelope {
  type: string
  id?: string
  name?: string
  payload?: unknown
}

export const Cmd = {
  Ping: 'PING',
  Subscribe: 'SUBSCRIBE',
  StartGame: 'START_GAME',
  ReplayGame: 'REPLAY_GAME',
  SelectTrump: 'SELECT_TRUMP',
  PlayCard: 'PLAY_CARD',
  Chat: 'CHAT',
} as const

export const Msg = {
  State: 'STATE',
  Events: 'EVENTS',
  Room: 'ROOM',
  Chat: 'CHAT',
  Error: 'ERROR',
  Pong: 'PONG',
} as const

export interface ChatMessage {
  id: number
  room_id: string
  user_id: string
  username: string
  body: string
  is_system: boolean
  at: string
}

// --- card helpers ---

export const RANK_LABEL: Record<number, string> = {
  11: 'J',
  12: 'Q',
  13: 'K',
  14: 'A',
}

export function rankLabel(r: number): string {
  return RANK_LABEL[r] ?? String(r)
}

export const Suits: Suit[] = ['spades', 'hearts', 'diamonds', 'clubs']

export const SUIT_GLYPH: Record<Suit, string> = {
  spades: '\u2660',
  hearts: '\u2665',
  diamonds: '\u2666',
  clubs: '\u2663',
}

export function isRed(s: Suit): boolean {
  return s === 'hearts' || s === 'diamonds'
}

// suitOrder is trump first, then opposite color, then the remaining
// trump-color suit, then the last opposite-color suit (ADR-0012).
export function suitOrder(trump?: Suit): [Suit, Suit, Suit, Suit] {
  if (!trump) {
    return ['hearts', 'spades', 'diamonds', 'clubs']
  }
  if (isRed(trump)) {
    const restSame: Suit = trump === 'hearts' ? 'diamonds' : 'hearts'
    return [trump, 'spades', restSame, 'clubs']
  }
  const restSame: Suit = trump === 'spades' ? 'clubs' : 'spades'
  return [trump, 'hearts', restSame, 'diamonds']
}

// sortHand groups by suitOrder and ranks Ace (14) down to 2 inside each suit.
export function sortHand(hand: Card[], trump?: Suit): Card[] {
  const order = suitOrder(trump)
  const idx: Record<string, number> = {}
  order.forEach((s, i) => { idx[s] = i })
  return hand.slice().sort((a, b) => {
    const su = (idx[a.suit] ?? 99) - (idx[b.suit] ?? 99)
    if (su !== 0) return su
    return b.rank - a.rank
  })
}

