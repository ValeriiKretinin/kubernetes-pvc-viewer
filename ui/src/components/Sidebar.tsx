type Props = {
  namespaces: string[]
  namespace: string
  onNamespace: (ns: string)=>void
  pvcs: string[]
  pvc: string
  onPvc: (p: string)=>void
}

export function Sidebar({ namespaces, namespace, onNamespace, pvcs, pvc, onPvc }: Props) {
  return (
    <div className="w-80 border-r border-gray-200 dark:border-gray-700 p-3 flex flex-col gap-3">
      <div>
        <div className="text-xs uppercase opacity-70 mb-1">Namespaces</div>
        <select className="w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded p-2"
                value={namespace} onChange={e=>onNamespace(e.target.value)}>
          <option value="">Select namespace</option>
          {namespaces.map(ns => <option key={ns} value={ns}>{ns}</option>)}
        </select>
      </div>
      <div>
        <div className="text-xs uppercase opacity-70 mb-1">PVCs</div>
        <select className="w-full bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded p-2"
                value={pvc} onChange={e=>onPvc(e.target.value)}>
          <option value="">Select PVC</option>
          {pvcs.map(p => <option key={p} value={p}>{p}</option>)}
        </select>
      </div>
    </div>
  )
}



