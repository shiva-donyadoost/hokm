import { useEffect, useState } from 'react'
import { AudioManager } from '../audio/AudioManager'

/** Small mute toggle; mute is localStorage-only (never sent to server). */
export function AudioMuteButton({ className = '' }: { className?: string }) {
  const [muted, setMuted] = useState(false)
  useEffect(() => {
    AudioManager.preload()
    setMuted(AudioManager.isMuted())
  }, [])
  return (
    <button
      type="button"
      className={`px-2 py-1 rounded-full border text-xs font-semibold shadow ${
        muted
          ? 'bg-slate-800 border-slate-600 text-slate-400'
          : 'bg-slate-900/90 border-slate-500 text-slate-100'
      } ${className}`}
      aria-pressed={muted}
      aria-label={muted ? 'unmute sound' : 'mute sound'}
      onClick={() => {
        AudioManager.unlock()
        setMuted(AudioManager.toggleMute())
      }}
    >
      {muted ? 'Sound off' : 'Sound on'}
    </button>
  )
}
