import { useEffect, useMemo, useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { FilePanel } from './components/FilePanel'

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
  const [theme, setTheme] = useState<'light'|'dark'>(() => (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'))

  useEffect(() => {
    setNsLoading(true)
    fetch('/api/v1/namespaces')
      .then(r => r.json())
      .then(list => { setNamespaces(list); try { localStorage.setItem('pvcviewer.namespaces', JSON.stringify(list)) } catch {} })
      .catch(()=>{})
      .finally(()=>setNsLoading(false))
  }, [])

  useEffect(() => {
    // reset PVC on namespace change to avoid stale selection/404s
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

  return (
    <div className={"flex h-screen "+ (theme==='dark'?'dark':'')}>
      <div className="fixed top-3 right-3 z-50">
        <button className="btn" onClick={()=>setTheme(t=>t==='dark'?'light':'dark')}>
          {theme==='dark'?'🌙 Dark':'☀️ Light'}
        </button>
      </div>
      <Sidebar namespaces={namespaces} namespace={namespace} onNamespace={setNamespace}
               pvcs={pvcs} pvc={pvc} onPvc={setPvc} pvcsLoading={pvcsLoading} nsLoading={nsLoading} />
      <div className="flex-1 overflow-hidden">
        <FilePanel namespace={namespace} pvc={pvc} />
      </div>
    </div>
  )
}



