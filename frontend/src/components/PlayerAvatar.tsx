import { useMemo, useState } from 'react'

/** ADR-0015: DiceBear HTTP API - lorelei, seed = user_id (else username). */
const DICEBEAR_STYLE = 'lorelei'
const DICEBEAR_VERSION = '9.x'

export function avatarSeed(userId?: string | null, username?: string | null): string {
  const id = (userId ?? '').trim()
  if (id) return id
  return (username ?? '').trim()
}

export function dicebearUrl(seed: string, size = 64): string {
  const q = encodeURIComponent(seed || 'player')
  return `https://api.dicebear.com/${DICEBEAR_VERSION}/${DICEBEAR_STYLE}/svg?seed=${q}&size=${size}`
}

type AvatarSize = 'sm' | 'md' | 'lg'

const SIZE_CLASS: Record<AvatarSize, string> = {
  sm: 'w-7 h-7 text-[10px]',
  md: 'w-10 h-10 text-sm',
  lg: 'w-16 h-16 text-2xl',
}

function initialsOf(name: string, seed: string): string {
  const base = (name || seed || '?').trim()
  if (!base) return '?'
  return base.slice(0, 2).toUpperCase()
}

export function PlayerAvatar({
  seed,
  name = '',
  size = 'md',
  className = '',
}: {
  /** Stable DiceBear seed (prefer user_id). */
  seed: string
  name?: string
  size?: AvatarSize
  className?: string
}) {
  const [failed, setFailed] = useState(false)
  const trimmed = seed.trim()
  const initials = useMemo(() => initialsOf(name, trimmed), [name, trimmed])
  const showImg = Boolean(trimmed) && !failed

  if (!showImg) {
    return (
      <div
        className={`${SIZE_CLASS[size]} rounded-full bg-teal-700 flex items-center justify-center font-black text-white shrink-0 ${className}`}
        aria-hidden
        title={name || undefined}
      >
        {initials}
      </div>
    )
  }

  return (
    <img
      src={dicebearUrl(trimmed)}
      alt=""
      title={name || undefined}
      className={`${SIZE_CLASS[size]} rounded-full bg-slate-700 object-cover shrink-0 ${className}`}
      loading="lazy"
      decoding="async"
      referrerPolicy="no-referrer"
      onError={() => setFailed(true)}
    />
  )
}
