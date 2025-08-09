import { useEffect, useState } from 'react'

type Props = {
  namespaces: string[]
  namespace: string
  onNamespace: (ns: string)=>void
  pvcs: string[]
  pvc: string
  onPvc: (p: string)=>void
  onSearch: (q: string)=>void
  theme: 'light'|'dark'
  setTheme: (t: 'light'|'dark')=>void
  nsLoading?: boolean
  pvcsLoading?: boolean
  onToggleSidebar?: ()=>void
  sidebarOpen?: boolean
}

export function HeaderBar({ namespaces, namespace, onNamespace, pvcs, pvc, onPvc, onSearch, theme, setTheme, nsLoading, pvcsLoading, onToggleSidebar, sidebarOpen }: Props) {
  const [q, setQ] = useState('')
  useEffect(()=>{ onSearch(q) }, [q])
  return (
    <div className="header-gradient sticky top-0 z-40">
      <div className="max-w-screen-2xl mx-auto px-4 py-3 flex items-center gap-3">
        <div className="flex items-center gap-2">
          <button className="px-2 py-1.5 rounded-md border border-gray-300 dark:border-gray-700 bg-white/70 dark:bg-gray-800/70" onClick={onToggleSidebar} aria-label="Toggle sidebar">☰</button>
          <div className="text-xl font-semibold tracking-tight text-strong">PVC Viewer</div>
        </div>
        <div className="ml-4 flex items-center gap-2">
          <select className="px-2 py-1 rounded bg-white/70 dark:bg-gray-800/70 border border-gray-300 dark:border-gray-600 backdrop-blur"
                  value={namespace} onChange={e=>onNamespace(e.target.value)}>
            <option value="">{nsLoading ? 'Loading namespaces…' : 'Namespace'}</option>
            {namespaces.map(ns => <option key={ns} value={ns}>{ns}</option>)}
          </select>
          <select className="px-2 py-1 rounded bg-white/70 dark:bg-gray-800/70 border border-gray-300 dark:border-gray-600 backdrop-blur"
                  value={pvc} onChange={e=>onPvc(e.target.value)}>
            <option value="">{pvcsLoading ? 'Loading PVCs…' : 'PVC'}</option>
            {pvcs.map(p => <option key={p} value={p}>{p}</option>)}
          </select>
        </div>
        <div className="ml-4 flex-1">
          <input className="w-full px-3 py-2 rounded-lg bg-white/80 dark:bg-gray-800/70 border border-gray-300 dark:border-gray-600 backdrop-blur focus:outline-none focus:ring-2 focus:ring-blue-500/40 text-gray-900 dark:text-gray-100 placeholder:text-gray-500 dark:placeholder:text-gray-400"
                 placeholder="Search files by name…"
                 spellCheck={false} autoCorrect="off" autoComplete="off"
                 value={q} onChange={e=>setQ(e.target.value)} />
        </div>
        <button className="btn" onClick={()=>setTheme(theme==='dark'?'light':'dark')}>
          {theme==='dark'?'🌙 Dark':'☀️ Light'}
        </button>
      </div>
    </div>
  )
}


