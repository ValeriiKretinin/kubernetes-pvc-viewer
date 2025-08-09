import { useEffect, useMemo, useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { FilePanel } from './components/FilePanel'
import { HeaderBar } from './components/HeaderBar'
import { Bars3Icon } from '@heroicons/react/24/outline'

export default function App() {
  const [namespaces, setNamespaces] = useState<string[]>(() => {
    try {
      const cached = localStorage.getItem('pvcviewer.namespaces')
      return cached ? JSON.parse(cached) : []
    } catch { return [] }
  })
  const [namespace, setNamespace] = useState<string>('')
  const [pvcs, setPvcs] = useState<string[]>([])
  const [pvc, setPvc] = useState<string>('')
  const [pvcsCache, setPvcsCache] = useState<Record<string, string[]>>({})
  const [pvcsLoading, setPvcsLoading] = useState<boolean>(false)
  const [nsLoading, setNsLoading] = useState<boolean>(false)
  const [theme, setTheme] = useState<'light'|'dark'>(() => {
    try {
      const saved = localStorage.getItem('pvcviewer.theme') as 'light'|'dark'|null
      if (saved === 'light' || saved === 'dark') return saved
    } catch {}
    // default to dark if nothing saved
    return 'dark'
  })
  const [query, setQuery] = useState<string>('')
  const [sidebarOpen, setSidebarOpen] = useState<boolean>(false)

  useEffect(() => {
    setNsLoading(true)
    fetch('/api/v1/namespaces')
      .then(r => r.json())
      .then(list => { setNamespaces(list); try { localStorage.setItem('pvcviewer.namespaces', JSON.stringify(list)) } catch {} })
      .catch(()=>{})
      .finally(()=>setNsLoading(false))
  }, [])

  useEffect(() => {
    // reset PVC and entries when namespace changes
    setPvc('')
    // show cached immediately if present
    if (namespace && pvcsCache[namespace]) {
      setPvcs(pvcsCache[namespace])
    } else {
      setPvcs([])
    }
    if (!namespace) return
    setPvcsLoading(true)
    fetch(`/api/v1/pvcs?namespace=${encodeURIComponent(namespace)}`)
      .then(r => r.json())
      .then(list => {
        setPvcs(list)
        setPvcsCache(prev => ({ ...prev, [namespace]: list }))
      })
      .catch(()=>{})
      .finally(() => setPvcsLoading(false))
  }, [namespace])

  // persist theme
  useEffect(() => { try { localStorage.setItem('pvcviewer.theme', theme) } catch {} }, [theme])

  return (
    <div className={"flex h-screen "+ (theme==='dark'?'dark':'')}>
      <div className="flex-1 flex flex-col">
        <HeaderBar namespaces={namespaces} namespace={namespace} onNamespace={setNamespace}
                   pvcs={pvcs} pvc={pvc} onPvc={setPvc}
                   onSearch={setQuery} theme={theme} setTheme={t=>setTheme(t)}
                   nsLoading={nsLoading} pvcsLoading={pvcsLoading}
                   onToggleSidebar={()=>setSidebarOpen(o=>!o)} sidebarOpen={sidebarOpen} />
        <div className="flex-1 flex overflow-hidden">
          <div className={`relative transition-all duration-200 ${sidebarOpen ? 'w-80' : 'w-0'} overflow-hidden surface-gradient bg-white dark:bg-gray-950 border-r border-gray-200 dark:border-gray-800`}> 
            <div className={`absolute inset-0 transition-transform duration-200 ${sidebarOpen ? 'translate-x-0' : '-translate-x-full'}`}>
              <Sidebar namespaces={namespaces} namespace={namespace} onNamespace={setNamespace}
                       pvcs={pvcs} pvc={pvc} onPvc={setPvc} pvcsLoading={pvcsLoading} nsLoading={nsLoading} />
            </div>
          </div>
          <div className="flex-1 overflow-hidden">
            <FilePanel namespace={namespace} pvc={pvc} query={query} />
          </div>
        </div>
      </div>
    </div>
  )
}



