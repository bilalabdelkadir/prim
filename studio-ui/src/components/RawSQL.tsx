import { useState } from 'react'

interface QueryResult {
  columns: string[];
  rows: Record<string, unknown>[];
  error?: string;
}

function RawSQL() {
  const [sql, setSql] = useState('')
  const [result, setResult] = useState<QueryResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const runQuery = () => {
    if (!sql.trim()) return
    setLoading(true)
    setError('')
    setResult(null)

    fetch('/api/sql/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sql }),
    })
      .then(res => res.json())
      .then((data: QueryResult) => {
        if (data.error) {
          setError(data.error)
        } else {
          setResult(data)
        }
      })
      .catch(err => setError(err instanceof Error ? err.message : 'Request failed'))
      .finally(() => setLoading(false))
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
      e.preventDefault()
      runQuery()
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Textarea area */}
      <div className="flex-shrink-0 p-4 pb-0">
        <textarea
          value={sql}
          onChange={e => setSql(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="SELECT * FROM users LIMIT 10;"
          className="w-full h-48 bg-[#050505] font-mono ui-text-md text-white/80 border border-white/[0.06] focus:border-white/30 rounded-sm p-3 resize-y outline-none placeholder:text-white/15 transition-all duration-150"
          spellCheck={false}
        />
      </div>

      {/* Action bar */}
      <div className="flex items-center justify-between px-4 py-2 flex-shrink-0">
        <button
          onClick={runQuery}
          disabled={loading || !sql.trim()}
          className="bg-[#05df72] text-black ui-text-label font-semibold px-4 py-1.5 rounded-sm hover:bg-[#00e87a] cursor-pointer transition-all duration-150
            disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {loading ? 'RUNNING...' : 'RUN'}
        </button>
        <span className="ui-text-label text-white/15 font-sans">
          {navigator.userAgent.includes('Mac') ? 'CMD' : 'CTRL'} + ENTER
        </span>
      </div>

      {/* Divider */}
      <div className="border-t border-white/[0.06] mx-4" />

      {/* Error — inline below action bar */}
      {error && (
        <div className="mx-4 mt-2 border border-red-500/30 bg-red-500/[0.05] text-red-400 ui-text-base font-mono rounded-sm p-2.5">
          {error}
        </div>
      )}

      {/* Results area — fills remaining space */}
      <div className="flex-1 overflow-y-auto px-4 pt-2 pb-4">
        {result && result.rows ? (
          <>
            <div className="mb-2">
              <span className="ui-text-label text-white/30 font-sans">
                {result.rows.length} row{result.rows.length !== 1 ? 's' : ''} returned
              </span>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b border-dashed border-white/10">
                    {result.columns.map(col => (
                      <th
                        key={col}
                        className="text-left px-3 py-2 ui-text-label text-white/30 whitespace-nowrap sticky top-0 bg-[#0a0a0a] z-10 font-sans"
                      >
                        {col}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {result.rows.map((row, i) => (
                    <tr
                      key={i}
                      className="border-b border-white/[0.04] last:border-b-0 hover:bg-white/[0.02] transition-all duration-150"
                    >
                      {result.columns.map(col => (
                        <td key={col} className="px-3 py-1.5 font-mono ui-text-base text-white/70 whitespace-nowrap">
                          {row[col] === null ? (
                            <span className="text-white/20 italic">NULL</span>
                          ) : (
                            String(row[col])
                          )}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        ) : !error && (
          <div className="flex items-center justify-center h-full">
            <span className="text-white/15 ui-text-sm tracking-wider font-sans">Run a query to see results</span>
          </div>
        )}
      </div>
    </div>
  )
}

export default RawSQL
