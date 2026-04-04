import { useState, useEffect, useRef } from 'react'
import type { PrimQuery } from '../types'
import CodePreview from './CodePreview'
import GoSyntaxHighlighter from './GoSyntaxHighlighter'

interface LiveCodePreviewProps {
  query: PrimQuery | null;
}

function LiveCodePreview({ query }: LiveCodePreviewProps) {
  const [code, setCode] = useState('')
  const [loading, setLoading] = useState(false)
  const [fullscreen, setFullscreen] = useState(false)
  const [copied, setCopied] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const abortRef = useRef<AbortController | null>(null)

  useEffect(() => {
    if (!fullscreen) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setFullscreen(false)
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [fullscreen])

  const handleCopy = () => {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  useEffect(() => {
    if (timerRef.current) clearTimeout(timerRef.current)

    if (!query || !query.model || !query.name) {
      setCode('')
      setLoading(false)
      return
    }

    setLoading(true)

    timerRef.current = setTimeout(() => {
      if (abortRef.current) abortRef.current.abort()
      const controller = new AbortController()
      abortRef.current = controller

      fetch('/api/query/build', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(query),
        signal: controller.signal,
      })
        .then(r => r.json())
        .then((data: { code: string; structs: string }) => {
          const full = (data.structs ? data.structs + '\n' : '') + data.code
          setCode(full)
          setLoading(false)
        })
        .catch(err => {
          if (err.name !== 'AbortError') {
            setLoading(false)
          }
        })
    }, 500)

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [query])

  if (!query || !query.model || !query.name) {
    return (
      <div className="h-full flex items-center justify-center p-8">
        <p className="text-[11px] text-white/25 tracking-wider uppercase text-center">
          Select a model and enter a method name<br />to see generated Go code
        </p>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col">
      <div className="flex items-center justify-between px-4 py-3">
        <span className="text-[10px] tracking-[0.2em] uppercase text-white/30 font-medium">
          GENERATED OUTPUT
        </span>
        <div className="flex items-center gap-3">
          {code && (
            <button
              onClick={() => setFullscreen(true)}
              className="ui-text-label text-white/20 hover:text-white/50 cursor-pointer transition-colors"
              title="Fullscreen (F11)"
            >
              EXPAND ↗
            </button>
          )}
          {loading && (
            <span className="w-1.5 h-1.5 rounded-full bg-[#05df72] animate-pulse" />
          )}
        </div>
      </div>
      <div className="flex-1 overflow-y-auto">
        {code ? (
          <CodePreview code={code} />
        ) : (
          <div className="flex items-center justify-center h-full p-8">
            <p className="text-[11px] text-white/20 tracking-wider">
              {loading ? '' : 'Code preview will appear here.'}
            </p>
          </div>
        )}
      </div>

      {fullscreen && code && (
        <div className="fixed inset-0 z-50 bg-[#0a0a0a]/95 backdrop-blur-sm flex flex-col animate-in">
          <div className="flex items-center justify-between px-6 py-3 border-b border-white/[0.06]">
            <span className="ui-text-label text-white/30">GENERATED CODE</span>
            <div className="flex items-center gap-3">
              <button
                onClick={handleCopy}
                className="ui-text-label cursor-pointer transition-colors"
                style={{ color: copied ? '#05df72' : 'rgba(255,255,255,0.3)' }}
              >
                {copied ? 'COPIED' : 'COPY'}
              </button>
              <button
                onClick={() => setFullscreen(false)}
                className="text-white/30 hover:text-white/70 cursor-pointer transition-colors ui-text-label"
              >
                ESC ✕
              </button>
            </div>
          </div>
          <div className="flex-1 overflow-y-auto p-2">
            <GoSyntaxHighlighter code={code} />
          </div>
        </div>
      )}
    </div>
  )
}

export default LiveCodePreview
