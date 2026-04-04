import { useState, useEffect, useCallback, useRef } from 'react'
import type { Model, View } from './types'
import TableBrowser from './components/TableBrowser'
import QueryBuilder from './components/QueryBuilder'
import RawSQL from './components/RawSQL'

type UiSize = 'compact' | 'default' | 'comfortable'

const NAV_ITEMS: Array<{ key: View; label: string; abbr: string }> = [
  { key: 'tables', label: 'Tables', abbr: 'TB' },
  { key: 'query', label: 'Query Builder', abbr: 'QB' },
  { key: 'sql', label: 'Raw SQL', abbr: 'SQ' },
]

const VIEW_LABELS: Record<View, string> = {
  tables: 'TABLES',
  query: 'QUERY BUILDER',
  sql: 'RAW SQL',
}

function App() {
  const [view, setView] = useState<View>('query')
  const [models, setModels] = useState<Model[]>([])
  const [selectedModel, setSelectedModel] = useState<string>('')
  const [modelsLoading, setModelsLoading] = useState(true)

  const [sidebarCollapsed, setSidebarCollapsed] = useState<boolean>(() => {
    return localStorage.getItem('prim-sidebar') === 'collapsed'
  })

  const [uiSize, setUiSize] = useState<UiSize>(() => {
    return (localStorage.getItem('prim-ui-size') as UiSize) || 'default'
  })

  const [settingsOpen, setSettingsOpen] = useState(false)
  const settingsRef = useRef<HTMLDivElement>(null)

  // Apply ui size on mount and change
  useEffect(() => {
    document.documentElement.setAttribute('data-ui-size', uiSize)
    localStorage.setItem('prim-ui-size', uiSize)
  }, [uiSize])

  // Persist sidebar state
  useEffect(() => {
    localStorage.setItem('prim-sidebar', sidebarCollapsed ? 'collapsed' : 'expanded')
  }, [sidebarCollapsed])

  // Fetch models
  useEffect(() => {
    fetch('/api/tables')
      .then(res => res.json())
      .then((data: Model[]) => {
        setModels(data)
        if (data.length > 0 && data[0]) {
          setSelectedModel(data[0].name)
        }
      })
      .catch(() => setModels([]))
      .finally(() => setModelsLoading(false))
  }, [])

  // Keyboard shortcut: Cmd+B / Ctrl+B to toggle sidebar
  const toggleSidebar = useCallback(() => {
    setSidebarCollapsed(prev => !prev)
  }, [])

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'b') {
        e.preventDefault()
        toggleSidebar()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [toggleSidebar])

  // Close settings panel on outside click
  useEffect(() => {
    if (!settingsOpen) return
    const handler = (e: MouseEvent) => {
      if (settingsRef.current && !settingsRef.current.contains(e.target as Node)) {
        setSettingsOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [settingsOpen])

  const sizeOptions: UiSize[] = ['compact', 'default', 'comfortable']

  return (
    <div className="flex flex-col h-screen overflow-hidden bg-[#0a0a0a] text-white">
      {/* Top Bar */}
      <header className="h-8 flex-shrink-0 border-b border-white/[0.06] bg-[#0a0a0a] flex items-center px-4 justify-between">
        <div className="flex items-center">
          <span className="tracking-[0.35em] text-[10px] font-bold text-white/60">PRIM</span>
          <span className="text-white/20 ml-1.5 tracking-[0.35em] text-[10px]">STUDIO</span>
        </div>
        <div className="relative" ref={settingsRef}>
          <button
            onClick={() => setSettingsOpen(prev => !prev)}
            className="text-white/30 hover:text-white/60 transition-colors duration-150 cursor-pointer text-[13px] leading-none"
            aria-label="Settings"
          >
            ⚙
          </button>
          {settingsOpen && (
            <div className="absolute right-2 top-8 bg-[#111] border border-white/10 rounded-md p-3 shadow-xl z-50">
              <p className="ui-text-label text-white/30 mb-2">FONT SIZE</p>
              <div className="flex gap-1">
                {sizeOptions.map(size => (
                  <button
                    key={size}
                    onClick={() => setUiSize(size)}
                    className={`px-2 py-1 rounded text-[9px] tracking-[0.15em] uppercase cursor-pointer transition-colors duration-150 ${
                      uiSize === size
                        ? 'bg-[rgba(5,223,114,0.12)] text-[#05df72]'
                        : 'text-white/30 hover:text-white/60 bg-white/[0.03]'
                    }`}
                  >
                    {size}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </header>

      {/* Middle: Sidebar + Content */}
      <div className="flex flex-1 min-h-0">
        {/* Sidebar */}
        <aside
          className={`flex-shrink-0 bg-[#0a0a0a] border-r border-white/[0.06] flex flex-col transition-all duration-200 ease-out ${
            sidebarCollapsed ? 'w-12' : 'w-52'
          }`}
        >
          {/* Sidebar toggle */}
          <div className="px-3 pt-2 pb-1 flex items-center">
            <button
              onClick={toggleSidebar}
              className="text-white/20 hover:text-white/50 transition-colors duration-150 cursor-pointer text-[11px] leading-none"
              aria-label="Toggle sidebar"
              title="Toggle sidebar (⌘B)"
            >
              {sidebarCollapsed ? '▶' : '◀'}
            </button>
          </div>

          {/* Navigation */}
          <nav className="px-1.5 py-1 space-y-0.5">
            {NAV_ITEMS.map(item => (
              <button
                key={item.key}
                onClick={() => setView(item.key)}
                className={`w-full flex items-center gap-2 rounded-sm transition-all duration-150 cursor-pointer ${
                  sidebarCollapsed ? 'justify-center px-0 py-1.5' : 'px-2.5 py-1.5'
                } ${
                  view === item.key
                    ? 'bg-[rgba(5,223,114,0.08)] text-[#05df72]'
                    : 'text-white/40 hover:text-white/70'
                }`}
                title={sidebarCollapsed ? item.label : undefined}
              >
                {view === item.key && !sidebarCollapsed && (
                  <span className="text-[6px]">●</span>
                )}
                {sidebarCollapsed ? (
                  <span className="font-mono text-[10px] tracking-tight">{item.abbr}</span>
                ) : (
                  <span className="ui-text-sm tracking-[0.12em] uppercase">{item.label}</span>
                )}
              </button>
            ))}
          </nav>

          {/* Models List */}
          <div className="flex-1 overflow-y-auto px-1.5 py-2 border-t border-white/[0.04] mt-1">
            {!sidebarCollapsed && (
              <p className="px-2.5 ui-text-label text-white/20 mb-1.5">MODELS</p>
            )}
            {modelsLoading ? (
              <div className={`py-2 ui-text-sm text-white/30 ${sidebarCollapsed ? 'text-center' : 'px-2.5'}`}>
                {sidebarCollapsed ? '...' : 'Loading...'}
              </div>
            ) : models.length === 0 ? (
              <div className={`py-2 ui-text-sm text-white/30 ${sidebarCollapsed ? 'text-center' : 'px-2.5'}`}>
                {sidebarCollapsed ? '—' : 'No models found'}
              </div>
            ) : (
              <div className="space-y-0.5">
                {models.map(m => (
                  <button
                    key={m.name}
                    onClick={() => {
                      setSelectedModel(m.name)
                      setView('tables')
                    }}
                    className={`w-full text-left rounded-sm transition-all duration-150 cursor-pointer ${
                      sidebarCollapsed ? 'px-0 py-1 text-center' : 'px-2.5 py-1'
                    } ${
                      selectedModel === m.name
                        ? 'text-[#05df72]'
                        : 'text-white/40 hover:text-white/70'
                    }`}
                    title={sidebarCollapsed ? m.name : undefined}
                  >
                    {sidebarCollapsed ? (
                      <span className="font-mono text-[10px]">{m.name.slice(0, 2).toUpperCase()}</span>
                    ) : (
                      <span className="font-mono ui-text-base">{m.name}</span>
                    )}
                  </button>
                ))}
              </div>
            )}
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 overflow-y-auto bg-[#0a0a0a]">
          {view === 'tables' && (
            <TableBrowser
              models={models}
              selectedModel={selectedModel}
              onSelectModel={setSelectedModel}
            />
          )}
          {view === 'query' && <QueryBuilder models={models} />}
          {view === 'sql' && <RawSQL />}
        </main>
      </div>

      {/* Status Bar */}
      <footer className="h-6 flex-shrink-0 border-t border-white/[0.06] bg-[#0a0a0a] flex items-center px-4 text-[9px] tracking-[0.15em] text-white/20">
        <div className="flex-1 flex items-center gap-1.5">
          {models.length > 0 ? (
            <>
              <span className="text-[#05df72]">■</span>
              <span>SCHEMA LOADED</span>
            </>
          ) : (
            <>
              <span className="text-red-500">■</span>
              <span>NO SCHEMA</span>
            </>
          )}
        </div>
        <div className="flex-1 text-center">
          {models.length} MODEL{models.length !== 1 ? 'S' : ''}
        </div>
        <div className="flex-1 text-right">
          {VIEW_LABELS[view]}
        </div>
      </footer>
    </div>
  )
}

export default App
