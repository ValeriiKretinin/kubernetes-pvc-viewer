import { useEffect, useMemo, useRef, useState } from 'react'
import { ContextMenu } from './ContextMenu'
import { FolderIcon, DocumentIcon, Squares2X2Icon, Bars3BottomLeftIcon } from '@heroicons/react/24/outline'
import { PreviewPane } from './PreviewPane'

type Entry = { name: string; path: string; isDir: boolean; size: number; mod: string; uid?: number; gid?: number; mode?: number }

type Props = { namespace: string; pvc: string; query?: string }

export function FilePanel({ namespace, pvc, query }: Props) {
  const [path, setPath] = useState<string>('/')
  const [entries, setEntries] = useState<Entry[]>([])
  const [total, setTotal] = useState<number>(0)
  const [offset, setOffset] = useState<number>(0)
  const [preview, setPreview] = useState<Entry|null>(null)
  const [error, setError] = useState<string>('')
  const [progress, setProgress] = useState<number>(0)
  const [view, setView] = useState<'table'|'grid'>('table')
  const limit = 200

  useEffect(() => { setPath('/'); setError('') }, [namespace, pvc])

  useEffect(() => {
    if (!namespace || !pvc) return
    const ac = new AbortController()
    const q = new URLSearchParams({ ns: namespace, pvc, path, limit: String(limit), offset: String(offset) })
    fetch(`/api/v1/tree?${q.toString()}`, { signal: ac.signal })
      .then(async r => {
        if (!r.ok) throw new Error(`API ${r.status}`)
        setTotal(Number(r.headers.get('X-Total-Count')||'0'))
        try { return await r.json() } catch { return [] }
      })
      .then(setEntries)
      .catch(e=>{ if ((e as any).name !== 'AbortError') setError(String(e)) })
    return () => ac.abort()
  }, [namespace, pvc, path, offset])

  const breadcrumbs = useMemo(() => {
    const segs = path.split('/').filter(Boolean)
    const acc: { name: string; path: string }[] = [{ name: 'root', path: '/' }]
    let cur = ''
    for (const s of segs) { cur += '/' + s; acc.push({ name: s, path: cur }) }
    return acc
  }, [path])

  const canPrev = offset > 0
  const canNext = offset + limit < total
  const filtered = useMemo(() => entries.filter(e => !query || e.name.toLowerCase().includes(query.toLowerCase())), [entries, query])

  // listen to tree navigation
  useEffect(() => {
    const onNav = (e: any) => {
      if (e?.detail?.path) { setPath(e.detail.path); setOffset(0) }
    }
    window.addEventListener('pvcviewer:navigate', onNav as any)
    return () => window.removeEventListener('pvcviewer:navigate', onNav as any)
  }, [])

  return (
    <div className="h-full flex flex-col bg-white dark:bg-gray-950">
      <div className="border-b border-gray-200 dark:border-gray-800 px-4 py-3 flex items-center gap-2 text-sm sticky top-0 bg-white/70 dark:bg-gray-950/70 backdrop-blur z-10">
        {breadcrumbs.map((b, i) => (
          <span key={b.path}>
            {i>0 && <span className="opacity-50">/</span>}<button className="hover:underline" onClick={()=>{setPath(b.path); setOffset(0)}}>{b.name}</button>
          </span>
        ))}
        <div className="ml-auto flex items-center gap-2">
          <div className="inline-flex rounded-md border border-gray-200 dark:border-gray-700 overflow-hidden">
            <button className={(view==='table'? 'bg-gray-100 dark:bg-gray-800 ': 'bg-transparent ') + 'px-2 py-1.5 text-xs flex items-center gap-1'} onClick={()=>setView('table')}>
              <Bars3BottomLeftIcon className="w-4 h-4" /> Table
            </button>
            <button className={(view==='grid'? 'bg-gray-100 dark:bg-gray-800 ': 'bg-transparent ') + 'px-2 py-1.5 text-xs flex items-center gap-1'} onClick={()=>setView('grid')}>
              <Squares2X2Icon className="w-4 h-4" /> Grid
            </button>
          </div>
          <button className="btn" onClick={()=>handleUpload(namespace, pvc, path, setError, ()=>setPath(path))}>Upload here</button>
          <button className="btn" onClick={()=>handleEmptyDir(namespace, pvc, path, setError, ()=>setPath(path))}>Empty dir</button>
        </div>
      </div>
      <div className="flex-1 overflow-auto">
        {view === 'table' ? (
          <table className="w-full text-sm">
            <thead className="text-left sticky top-0 bg-gray-50/90 dark:bg-gray-900/90 backdrop-blur">
              <tr><th className="p-2">Name</th><th>Size</th><th>Modified</th><th>Owner</th><th>Group</th><th>Mode</th><th></th></tr>
            </thead>
            <tbody>
              {filtered.map(e => (
                <tr key={e.path} className="border-b border-gray-100 dark:border-gray-900 hover:bg-gray-50 dark:hover:bg-gray-900/70 transition">
                  <td className="p-2">
                    <div className="flex items-center gap-2">
                      {e.isDir ? <FolderIcon className="w-5 h-5 text-gray-500 dark:text-gray-400"/> : <DocumentIcon className="w-5 h-5 text-gray-500 dark:text-gray-400"/>}
                      {e.isDir ? (
                        <button className="text-blue-600 dark:text-blue-400 font-medium" onClick={()=>{setPath(e.path); setOffset(0)}}>{e.name}</button>
                      ) : e.name}
                    </div>
                  </td>
                  <td className="p-2">{e.isDir ? '-' : formatSize(e.size)}</td>
                  <td className="p-2">{new Date(e.mod).toLocaleString()}</td>
                  <td className="p-2">{e.uid ?? '-'}</td>
                  <td className="p-2">{e.gid ?? '-'}</td>
                  <td className="p-2">{formatMode(e.mode)}</td>
                  <td className="p-2 text-right">
                    <ContextMenu
                      onDownload={!e.isDir ? ()=>downloadWithProgress(namespace, pvc, e.path, setProgress, setError) : undefined}
                      onDelete={()=>handleDelete(namespace, pvc, e.path, !!e.isDir, setError, setPath)}
                      onUpload={e.isDir ? ()=>handleUpload(namespace, pvc, e.path, setError, ()=>setPath(e.path)) : undefined}
                      onInfo={!e.isDir ? ()=>setPreview(e) : undefined}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div className="p-3 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-3">
            {filtered.map(e => (
              <div key={e.path} className="panel p-3 transition will-change-transform hover:-translate-y-0.5 hover:shadow-md">
                <div className="flex items-start gap-2">
                  {e.isDir ? <FolderIcon className="w-6 h-6 text-gray-500 dark:text-gray-400"/> : <DocumentIcon className="w-6 h-6 text-gray-500 dark:text-gray-400"/>}
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-gray-900 dark:text-gray-100 font-medium">
                      {e.isDir ? (
                        <button className="hover:underline" onClick={()=>{setPath(e.path); setOffset(0)}}>{e.name}</button>
                      ) : e.name}
                    </div>
                    <div className="text-xs text-gray-500 dark:text-gray-400 mt-1 flex gap-3">
                      <span>{e.isDir ? '—' : formatSize(e.size)}</span>
                      <span>{new Date(e.mod).toLocaleDateString()}</span>
                      <span>gid {e.gid ?? '-'}</span>
                    </div>
                  </div>
                  <div>
                    <ContextMenu
                      onDownload={!e.isDir ? ()=>downloadWithProgress(namespace, pvc, e.path, setProgress, setError) : undefined}
                      onDelete={()=>handleDelete(namespace, pvc, e.path, !!e.isDir, setError, setPath)}
                      onUpload={e.isDir ? ()=>handleUpload(namespace, pvc, e.path, setError, ()=>setPath(e.path)) : undefined}
                      onInfo={!e.isDir ? ()=>setPreview(e) : undefined}
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
      {preview && (
        <PreviewPane entry={preview} namespace={namespace} pvc={pvc} onClose={()=>setPreview(null)} />
      )}
      <div className="border-t border-gray-200 dark:border-gray-900 p-2 flex items-center justify-between text-sm">
        <div>{offset+1}-{Math.min(offset+limit, total)} of {total}</div>
        <div className="flex gap-2">
          <button disabled={!canPrev} className="px-2 py-1 border rounded disabled:opacity-50" onClick={()=>setOffset(Math.max(0, offset-limit))}>Prev</button>
          <button disabled={!canNext} className="px-2 py-1 border rounded disabled:opacity-50" onClick={()=>setOffset(offset+limit)}>Next</button>
        </div>
      </div>
      {!!error && (
        <div className="fixed bottom-4 right-4 bg-red-600 text-white px-3 py-2 rounded shadow-lg" onClick={()=>setError('')}>{error}</div>
      )}
      {progress>0 && progress<100 && (
        <div className="fixed bottom-4 left-1/2 -translate-x-1/2 bg-gray-800 text-white px-3 py-2 rounded shadow-lg">
          Downloading... {progress.toFixed(0)}%
        </div>
      )}
    </div>
  )
}

function formatSize(n: number) {
  const units = ['B','KB','MB','GB','TB']
  let i = 0; let v = n
  while (v >= 1024 && i < units.length-1) { v /= 1024; i++ }
  return `${v.toFixed(1)} ${units[i]}`
}

function downloadWithProgress(ns:string, pvc:string, filePath:string, setProgress:(n:number)=>void, setError:(s:string)=>void) {
  const url = `/api/v1/download?ns=${encodeURIComponent(ns)}&pvc=${encodeURIComponent(pvc)}&path=${encodeURIComponent(filePath)}`
  const xhr = new XMLHttpRequest()
  xhr.open('GET', url)
  xhr.responseType = 'blob'
  xhr.onprogress = (e) => {
    if (e.lengthComputable) {
      setProgress((e.loaded / e.total) * 100)
    }
  }
  xhr.onload = () => {
    setProgress(100)
    const blob = xhr.response
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = filePath.split('/').pop() || 'file'
    a.click()
    URL.revokeObjectURL(a.href)
    setTimeout(()=>setProgress(0), 1000)
  }
  xhr.onerror = () => {
    setError('Download failed')
    setProgress(0)
  }
  xhr.send()
}

function handleDelete(ns:string, pvc:string, p:string, isDir:boolean, setError:(s:string)=>void, refresh:(p:string)=>void) {
  const url = `/api/v1/file?ns=${encodeURIComponent(ns)}&pvc=${encodeURIComponent(pvc)}&path=${encodeURIComponent(p)}`
  fetch(url, { method: 'DELETE' }).then(r => {
    if (!r.ok) throw new Error(`Delete failed: ${r.status}`)
    // refresh parent directory
    const parent = p.split('/').slice(0,-1).join('/') || '/'
    refresh(parent)
  }).catch(e=>setError(String(e)))
}

function handleUpload(ns:string, pvc:string, dir:string, setError:(s:string)=>void, onDone:()=>void) {
  const input = document.createElement('input')
  input.type = 'file'
  input.multiple = true
  input.onchange = async () => {
    if (!input.files || input.files.length===0) return
    const form = new FormData()
    for (const f of Array.from(input.files)) form.append('file', f)
    const url = `/api/v1/upload?ns=${encodeURIComponent(ns)}&pvc=${encodeURIComponent(pvc)}&path=${encodeURIComponent(dir)}`
    try {
      const r = await fetch(url, { method: 'POST', body: form })
      if (!r.ok) throw new Error(`Upload failed: ${r.status}`)
      onDone()
    } catch (e:any) {
      setError(String(e))
    }
  }
  input.click()
}

function handleEmptyDir(ns:string, pvc:string, dir:string, setError:(s:string)=>void, onDone:()=>void) {
  const url = `/api/v1/empty-dir?ns=${encodeURIComponent(ns)}&pvc=${encodeURIComponent(pvc)}&path=${encodeURIComponent(dir)}`
  fetch(url, { method: 'POST' }).then(r => {
    if (!r.ok) throw new Error(`Empty dir failed: ${r.status}`)
    onDone()
  }).catch(e=>setError(String(e)))
}

function formatMode(mode?: number) {
  if (mode==null) return '-'
  const m = mode & 0o777
  const to = (n:number)=>[(n&4?'r':'-'),(n&2?'w':'-'),(n&1?'x':'-')].join('')
  return to((m>>6)&7)+' '+to((m>>3)&7)+' '+to(m&7)
}



