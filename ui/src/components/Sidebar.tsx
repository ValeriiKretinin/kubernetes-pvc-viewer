type Props = {
  namespaces: string[]
  namespace: string
  onNamespace: (ns: string)=>void
  pvcs: string[]
  pvc: string
  onPvc: (p: string)=>void
  pvcsLoading?: boolean
  nsLoading?: boolean
}

export function Sidebar({ namespaces, namespace, onNamespace, pvcs, pvc, onPvc, pvcsLoading, nsLoading }: Props) {
  return (
    <div className="w-80 border-r border-gray-200 dark:border-gray-700 p-3 flex flex-col gap-4 bg-white dark:bg-gray-900">
      <div>
        <div className="text-xs uppercase opacity-70 mb-1">Namespaces</div>
        <div className="relative">
          <span className="absolute left-2 top-2.5 text-gray-400">🔎</span>
          <select className="w-full pl-7 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border border-gray-300 dark:border-gray-600 rounded p-2"
                  value={namespace} onChange={e=>onNamespace(e.target.value)}>
            <option value="">{nsLoading ? 'Loading namespaces...' : 'Select namespace'}</option>
            {namespaces.map(ns => <option key={ns} value={ns}>{ns}</option>)}
          </select>
        </div>
      </div>
      <div>
        <div className="text-xs uppercase opacity-70 mb-1">PVCs</div>
        <div className="relative">
          <span className="absolute left-2 top-2.5 text-gray-400">🗂️</span>
          <select className="w-full pl-7 bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100 border border-gray-300 dark:border-gray-600 rounded p-2"
                  value={pvc} onChange={e=>onPvc(e.target.value)}>
            <option value="">Select PVC</option>
            {pvcsLoading && <option value="" disabled>Loading...</option>}
            {pvcs.map(p => <option key={p} value={p}>{p}</option>)}
          </select>
        </div>
      </div>
    </div>
  )
}



