// Structured client diagnostics for intermittent UI / WS bugs (ADR-0014).
// Ring buffer + console JSON. No tokens, passwords, or full chat bodies.
// In DevTools: window.__HOKM_DIAG__.dump() or .tail(50)

export type DiagLevel = 'debug' | 'info' | 'warn' | 'error'

export interface DiagEvent {
  t: number
  level: DiagLevel
  cat: string
  msg: string
  data?: Record<string, unknown>
}

const MAX = 200
const buffer: DiagEvent[] = []
let installed = false

function push(level: DiagLevel, cat: string, msg: string, data?: Record<string, unknown>): void {
  const ev: DiagEvent = { t: Date.now(), level, cat, msg, data }
  buffer.push(ev)
  if (buffer.length > MAX) buffer.shift()
  const line = JSON.stringify(ev)
  if (level === 'error') console.error('[hokm]', line)
  else if (level === 'warn') console.warn('[hokm]', line)
  else console.info('[hokm]', line)
}

export function diag(level: DiagLevel, cat: string, msg: string, data?: Record<string, unknown>): void {
  push(level, cat, msg, data)
}

export function diagInfo(cat: string, msg: string, data?: Record<string, unknown>): void {
  push('info', cat, msg, data)
}

export function diagWarn(cat: string, msg: string, data?: Record<string, unknown>): void {
  push('warn', cat, msg, data)
}

export function diagError(cat: string, msg: string, data?: Record<string, unknown>): void {
  push('error', cat, msg, data)
}

export function diagDump(): DiagEvent[] {
  return buffer.slice()
}

export function diagTail(n = 50): DiagEvent[] {
  return buffer.slice(Math.max(0, buffer.length - n))
}

export function diagClear(): void {
  buffer.length = 0
}

/** Play-card / trump attempt outcome for stuck-turn diagnosis. */
export function logPlayAttempt(data: {
  action: 'play_card' | 'select_trump'
  ok: boolean
  reason?: string
  card?: { suit: string; rank: number }
  suit?: string
  myTurn?: boolean
  phase?: string
  turn?: number
  you?: number
  wsState?: number | string
  legalKeys?: string[]
}): void {
  push(data.ok ? 'info' : 'warn', 'play', data.ok ? 'attempt_ok' : 'attempt_blocked', data)
}

/** Snapshot of turn / hand / lock flags (call on turn change or blocked play). */
export function logTurnSnapshot(data: {
  phase: string
  turn: number
  you: number
  myTurn: boolean
  trumpPending: boolean
  matchOver: boolean
  trickLen: number
  handKeys: string[]
  legalKeys: string[]
  selected: string | null
  collecting: boolean
  reveal: boolean
  connected: boolean
  wsState: number | string
  deadlineKind?: string
}): void {
  push('info', 'turn', 'snapshot', data)
}

export function installClientDiagnostics(): void {
  if (installed || typeof window === 'undefined') return
  installed = true
  window.addEventListener('error', (ev) => {
    push('error', 'window', 'uncaught', {
      message: String(ev.message ?? ''),
      source: String(ev.filename ?? ''),
      line: ev.lineno,
      col: ev.colno,
    })
  })
  window.addEventListener('unhandledrejection', (ev) => {
    const reason = ev.reason
    push('error', 'window', 'unhandledrejection', {
      message: reason instanceof Error ? reason.message : String(reason),
    })
  })
  const api = {
    dump: diagDump,
    tail: diagTail,
    clear: diagClear,
    log: diag,
  }
  ;(window as unknown as { __HOKM_DIAG__?: typeof api }).__HOKM_DIAG__ = api
  push('info', 'diag', 'installed')
}
