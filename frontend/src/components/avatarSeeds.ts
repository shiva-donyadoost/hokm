// Curated DiceBear seeds/styles (ADR-0017/0018). Keep in sync with backend.
export const AVATAR_STYLES = ['lorelei', 'avataaars'] as const

export type AvatarStyleOption = (typeof AVATAR_STYLES)[number]

export const DEFAULT_AVATAR_STYLE: AvatarStyleOption = 'lorelei'

export const AVATAR_SEEDS = [
  'fox', 'owl', 'panda', 'tiger', 'wolf', 'hawk',
  'seal', 'koala', 'otter', 'raven', 'lynx', 'bison',
  'coral', 'maple', 'comet', 'dune', 'ember', 'jade',
] as const

export type AvatarSeedOption = (typeof AVATAR_SEEDS)[number]

export function isAllowedAvatarSeed(seed: string): boolean {
  return (AVATAR_SEEDS as readonly string[]).includes(seed)
}

export function isAllowedAvatarStyle(style: string): boolean {
  return (AVATAR_STYLES as readonly string[]).includes(style)
}

export function normalizeAvatarStyle(style?: string | null): AvatarStyleOption {
  const s = (style ?? '').trim().toLowerCase()
  if (isAllowedAvatarStyle(s)) return s as AvatarStyleOption
  return DEFAULT_AVATAR_STYLE
}
