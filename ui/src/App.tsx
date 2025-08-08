import { useEffect, useMemo, useState } from 'react'
import { Sidebar } from './components/Sidebar'
import { FilePanel } from './components/FilePanel'

export default function App() {
  const [namespaces, setNamespaces] = useState<string[]>([])
  const [namespace, setNamespace] = useState<string>('')
  const [pvcs, setPvcs] = useState<string[]>([])
  const [pvc, setPvc] = useState<string>('')

  useEffect(() => {
    fetch('/api/v1/namespaces').then(r => r.json()).then(setNamespaces).catch(()=>{})
  }, [])

  useEffect(() => {
    if (!namespace) return
    fetch(`/api/v1/pvcs?namespace=${encodeURIComponent(namespace)}`)
      .then(r => r.json()).then(setPvcs).catch(()=>{})
  }, [namespace])

  return (
    <div className="flex h-screen">
      <Sidebar namespaces={namespaces} namespace={namespace} onNamespace={setNamespace}
               pvcs={pvcs} pvc={pvc} onPvc={setPvc} />
      <div className="flex-1 overflow-hidden">
        <FilePanel namespace={namespace} pvc={pvc} />
      </div>
    </div>
  )
}



