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

import { TreeView } from './TreeView'

export function Sidebar({ namespaces, namespace, onNamespace, pvcs, pvc, onPvc, pvcsLoading, nsLoading }: Props) {
  return (
    <div className="w-80 border-r border-gray-200 dark:border-gray-800 p-4 flex flex-col gap-3 bg-white dark:bg-gray-950 overflow-auto">
      <div>
        <div className="text-[11px] uppercase tracking-wide text-muted">Folders</div>
        <div className="text-xs text-muted-weak mt-1">
          {namespace && pvc ? `${namespace} / ${pvc}` : 'Select namespace and PVC in the header'}
        </div>
      </div>
      <div className="mt-2 animate-fade-in">
        <TreeView namespace={namespace} pvc={pvc} />
      </div>
    </div>
  )
}



