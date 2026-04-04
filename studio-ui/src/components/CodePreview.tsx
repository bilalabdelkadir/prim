import { useState } from 'react'
import GoSyntaxHighlighter from './GoSyntaxHighlighter'

function CodePreview({ code }: { code: string }) {
  const [copied, setCopied] = useState(false)

  const handleCopy = () => {
    navigator.clipboard.writeText(code).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <div className="relative group">
      <button
        onClick={handleCopy}
        className="absolute top-3 right-3 z-10 px-2.5 py-1 text-[9px] tracking-[0.15em] uppercase font-medium bg-white/5 hover:bg-white/10 border border-white/10 rounded-sm transition-all duration-150 cursor-pointer opacity-0 group-hover:opacity-100"
        style={{ color: copied ? '#05df72' : 'rgba(255,255,255,0.5)' }}
      >
        {copied ? 'COPIED' : 'COPY'}
      </button>
      <GoSyntaxHighlighter code={code} />
    </div>
  )
}

export default CodePreview
