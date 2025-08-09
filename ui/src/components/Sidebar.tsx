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
    <div className="w-80 border-r border-gray-200 dark:border-gray-700 p-3 flex flex-col gap-3 bg-white dark:bg-gray-900 overflow-auto">
      <div className="text-xs uppercase opacity-70">Folders</div>
      <div className="text-xs opacity-60 mb-1">{namespace && pvc ? `${namespace} / ${pvc}` : 'Select namespace and PVC in the header'}</div>
      <div className="mt-2">
        <TreeView namespace={namespace} pvc={pvc} />
      </div>
    </div>
  )
}



