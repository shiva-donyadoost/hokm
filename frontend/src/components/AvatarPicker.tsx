import { AVATAR_SEEDS, AVATAR_STYLES, type AvatarStyleOption } from './avatarSeeds'
import { PlayerAvatar } from './PlayerAvatar'

const STYLE_LABEL: Record<AvatarStyleOption, string> = {
  lorelei: 'Lorelei',
  avataaars: 'Avataaars',
}

/** Grid of curated DiceBear style+seed pairs for register/profile (ADR-0018). */
export function AvatarPicker({
  style,
  seed,
  onChange,
  label = 'Choose avatar',
}: {
  style: string
  seed: string
  onChange: (style: string, seed: string) => void
  label?: string
}) {
  return (
    <div>
      <p className="text-sm text-slate-400 mb-2">{label}</p>
      <div className="flex flex-col gap-4">
        {AVATAR_STYLES.map((st) => (
          <div key={st}>
            <p className="text-xs uppercase tracking-wide text-slate-500 mb-2">{STYLE_LABEL[st]}</p>
            <div className="grid grid-cols-6 gap-2" role="listbox" aria-label={`${label} - ${STYLE_LABEL[st]}`}>
              {AVATAR_SEEDS.map((s) => {
                const selected = style === st && seed === s
                return (
                  <button
                    key={`${st}-${s}`}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    title={`${STYLE_LABEL[st]} / ${s}`}
                    onClick={() => onChange(st, s)}
                    className={
                      'rounded-xl p-1 border transition focus:outline-none focus:ring-2 focus:ring-teal-400 ' +
                      (selected
                        ? 'border-teal-400 bg-teal-500/20 ring-2 ring-teal-400'
                        : 'border-slate-700 bg-slate-800/60 hover:border-slate-500')
                    }
                  >
                    <PlayerAvatar seed={s} name={s} style={st} size="md" className="mx-auto" />
                  </button>
                )
              })}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
