// Centralized presentation configuration (impliment.md §38–§39).
// Gameplay timeouts are server-authoritative (delivered via SeatView
// deadlines); these values are PRESENTATION-only and must never influence
// game legality. Defaults mirror the server's GameConfig presentation
// values; they live here so no timing literal scatters through components.

export const ANIM = {
  /** Card movement duration when a played card appears in the trick (§13). */
  cardPlayMs: 500,
  /** Winner highlight before collection starts (§14). Reveal + collect
   * together match the server's 1s TrickPause so the next trick begins
   * exactly as the collection lands. */
  trickWinnerMs: 400,
  /** Cards flying toward the winner after the highlight (§15). */
  cardCollectionMs: 600,
  /** Per-card delay during the dealing animation (§6). */
  dealStaggerMs: 60,
  /** Cancelled drag snap-back duration (§23). */
  cardReturnMs: 250,
} as const

export const TIMERS = {
  /** Countdown bar smoothness. */
  tickMs: 100,
} as const

export const GAME = {
  /** Emoji set shown in the chat picker (kept small and game-appropriate). */
  chatEmojis: ['😀', '😂', '😮', '😎', '🤔', '😢', '😡', '👏', '🙏', '💪', '🔥', '♠️', '♥️', '♦️', '♣️', '🏆', '🎉', '👍'],
} as const

/** respectsReducedMotion: honor the user's prefers-reduced-motion setting
 * by shortening decorative animations (§48) — never gameplay timing. */
export function prefersReducedMotion(): boolean {
  return typeof window !== 'undefined' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export function animDuration(ms: number): number {
  return prefersReducedMotion() ? Math.min(ms, 120) : ms
}
