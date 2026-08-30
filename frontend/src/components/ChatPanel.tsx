import { useEffect, useRef, useState } from 'react'
import { useGame } from '../state/game'
import { GAME } from '../config'

// ChatPanel renders room chat: history + live messages + input with an
// inline emoji picker (section 32). The picker is game-scoped and lightweight so
// it stays inside the viewport on mobile (section 50).
export function ChatPanel({ compact = false }: { compact?: boolean }) {
  const chat = useGame((s) => s.chat)
  const sendChat = useGame((s) => s.sendChat)
  const [body, setBody] = useState('')
  const [pickerOpen, setPickerOpen] = useState(false)
  const listRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = listRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [chat.length])

  return (
    <div className="flex flex-col border border-slate-800 rounded-xl bg-slate-900/80 overflow-hidden">
      <div
        ref={listRef}
        className={`flex flex-col gap-0.5 overflow-y-auto px-3 py-2 ${
          compact ? 'h-28' : 'h-56'
        }`}
      >
        {chat.length === 0 ? (
          <p className="text-xs text-slate-600">no messages yet - say salaam</p>
        ) : (
          chat.map((m) => (
            <p key={m.id} className={`text-xs leading-snug ${m.is_system ? 'text-amber-300/80 italic' : ''}`}>
              {m.is_system ? (
                m.body
              ) : (
                <>
                  <span className="font-semibold text-teal-300">{m.username}:</span> {m.body}
                </>
              )}
            </p>
          ))
        )}
      </div>
      {pickerOpen ? (
        <div
          className="flex flex-wrap gap-1 p-2 border-t border-slate-800 bg-slate-800/60"
          role="listbox"
          aria-label="emoji picker"
        >
          {GAME.chatEmojis.map((e) => (
            <button
              key={e}
              type="button"
              role="option"
              aria-selected={false}
              className="text-xl p-1 rounded hover:bg-slate-700 active:scale-90"
              onClick={() => {
                setBody((b) => b + e)
                setPickerOpen(false)
              }}
            >
              {e}
            </button>
          ))}
        </div>
      ) : null}
      <form
        className="flex gap-1 p-2 border-t border-slate-800"
        onSubmit={(e) => {
          e.preventDefault()
          const b = body.trim()
          if (!b) return
          sendChat(b)
          setBody('')
          setPickerOpen(false)
        }}
      >
        <button
          type="button"
          className="px-2 rounded-lg bg-slate-800 hover:bg-slate-700 text-lg"
          aria-label="open emoji picker"
          onClick={() => setPickerOpen((v) => !v)}
        >
          -
        </button>
        <input
          className="input !py-1 text-xs"
          placeholder="message..."
          maxLength={500}
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />
        <button className="btn-primary !px-3 !py-1 text-xs" type="submit">
          send
        </button>
      </form>
    </div>
  )
}
