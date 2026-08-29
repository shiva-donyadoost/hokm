import { useEffect, useRef, useState } from 'react'
import { useGame } from '../state/game'

// ChatPanel renders room chat: history + live messages + input.
export function ChatPanel({ compact = false }: { compact?: boolean }) {
  const chat = useGame((s) => s.chat)
  const sendChat = useGame((s) => s.sendChat)
  const [body, setBody] = useState('')
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
          <p className="text-xs text-slate-600">no messages yet — say salaam</p>
        ) : (
          chat.map((m) => (
            <p key={m.id} className={`text-xs leading-snug ${m.is_system ? 'text-amber-300/80 italic' : ''}`}>
              {m.is_system ? (
                m.body
              ) : (
                <>
                  <span className="font-semibold text-teal-300">{m.username}:</span>{' '}
                  {m.body}
                </>
              )}
            </p>
          ))
        )}
      </div>
      <form
        className="flex gap-1 p-2 border-t border-slate-800"
        onSubmit={(e) => {
          e.preventDefault()
          const b = body.trim()
          if (!b) return
          sendChat(b)
          setBody('')
        }}
      >
        <input
          className="input !py-1 text-xs"
          placeholder="message…"
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
