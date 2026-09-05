import { AVATAR_SEEDS } from './avatarSeeds'
import { PlayerAvatar } from './PlayerAvatar'

/** Grid of curated DiceBear seeds for register/profile (ADR-0017). */
export function AvatarPicker({
  value,
  onChange,
  label = 'Choose avatar',
}: {
  value: string
  onChange: (seed: string) => void
  label?: string
}) {
  return (
    <div>
      <p className="text-sm text-slate-400 mb-2">{label}</p>
      <div className="grid grid-cols-6 gap-2" role="listbox" aria-label={label}>
        {AVATAR_SEEDS.map((seed) => {
          const selected = value === seed
          return (
            <button
              key={seed}
              type="button"
              role="option"
              aria-selected={selected}
              title={seed}
              onClick={() => onChange(seed)}
              className={
                'rounded-xl p-1 border transition focus:outline-none focus:ring-2 focus:ring-teal-400 ' +
                (selected
                  ? 'border-teal-400 bg-teal-500/20 ring-2 ring-teal-400'
                  : 'border-slate-700 bg-slate-800/60 hover:border-slate-500')
              }
            >
              <PlayerAvatar seed={seed} name={seed} size="md" className="mx-auto" />
            </button>
          )
        })}
      </div>
    </div>
  )
}
