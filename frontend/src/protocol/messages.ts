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
  match_over: boolean
}

export interface RoomMember {
  user_id: string
  username: string
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
  SelectTrump: 'SELECT_TRUMP',
  PlayCard: 'PLAY_CARD',
} as const

export const Msg = {
  State: 'STATE',
  Events: 'EVENTS',
  Room: 'ROOM',
  Error: 'ERROR',
  Pong: 'PONG',
} as const

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

export const SUIT_SYMBOL: Record<Suit, string> = {
  spades: '♠',
  hearts: '♥',
  diamonds: '♦',
  clubs: '♣',
}

export const Suits: Suit[] = ['spades', 'hearts', 'diamonds', 'clubs']

export function isRed(s: Suit): boolean {
  return s === 'hearts' || s === 'diamonds'
}

export function cardLabel(c: Card): string {
  return rankLabel(c.rank) + SUIT_SYMBOL[c.suit]
}
