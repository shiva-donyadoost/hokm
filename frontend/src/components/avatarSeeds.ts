// Curated DiceBear seeds (ADR-0017). Keep in sync with backend AllowedAvatarSeeds.
export const AVATAR_SEEDS = [
  'fox', 'owl', 'panda', 'tiger', 'wolf', 'hawk',
  'seal', 'koala', 'otter', 'raven', 'lynx', 'bison',
  'coral', 'maple', 'comet', 'dune', 'ember', 'jade',
] as const

export type AvatarSeedOption = (typeof AVATAR_SEEDS)[number]

export function isAllowedAvatarSeed(seed: string): boolean {
  return (AVATAR_SEEDS as readonly string[]).includes(seed)
}
