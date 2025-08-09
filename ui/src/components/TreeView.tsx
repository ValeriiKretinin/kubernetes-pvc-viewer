import { useCallback, useEffect, useState } from 'react'

type Node = { path: string; name: string; expanded: boolean; loaded: boolean; children: Node[] }

type Props = { namespace: string; pvc: string }

export function TreeView({ namespace, pvc }: Props) {
  const [root, setRoot] = useState<Node>({ path: '/', name: '/', expanded: true, loaded: false, children: [] })

  useEffect(() => {
    // reset on selection change
    setRoot({ path: '/', name: '/', expanded: true, loaded: false, children: [] })
  }, [namespace, pvc])

  const loadDir = useCallback(async (node: Node) => {
    if (!namespace || !pvc) return
    try {
      const q = new URLSearchParams({ ns: namespace, pvc, path: node.path, limit: '1000', offset: '0' })
      const r = await fetch(`/api/v1/tree?${q.toString()}`)
      const items = await r.json()
      const dirs = items.filter((it: any) => it.isDir)
      node.children = dirs.map((d: any) => ({ path: d.path, name: d.name, expanded: false, loaded: false, children: [] }))
      node.loaded = true
      setRoot(r => ({ ...r }))
    } catch {}
  }, [namespace, pvc])

  const toggle = async (node: Node) => {
    node.expanded = !node.expanded
    if (node.expanded && !node.loaded) {
      await loadDir(node)
    } else {
      setRoot(r => ({ ...r }))
    }
  }

  const navigate = (path: string) => {
    window.dispatchEvent(new CustomEvent('pvcviewer:navigate', { detail: { path } }))
  }

  const render = (node: Node, depth = 0) => (
    <div key={node.path}>
      <div className="flex items-center gap-1 cursor-pointer hover:underline" style={{ paddingLeft: depth * 12 }}>
        <button className="text-xs" onClick={() => toggle(node)}>{node.expanded ? '▾' : '▸'}</button>
        <span onClick={() => navigate(node.path)}>{node.name}</span>
      </div>
      {node.expanded && node.children.map(child => render(child, depth + 1))}
    </div>
  )

  // lazy load root
  useEffect(() => {
    if (root.expanded && !root.loaded) {
      loadDir(root)
    }
  }, [root, loadDir])

  if (!namespace || !pvc) return null
  return (
    <div className="text-sm">
      {render(root)}
    </div>
  )
}


