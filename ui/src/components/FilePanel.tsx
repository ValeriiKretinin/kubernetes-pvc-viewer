import { useEffect, useMemo, useRef, useState } from 'react'
import { ContextMenu } from './ContextMenu'

type Entry = { name: string; path: string; isDir: boolean; size: number; mod: string; uid?: number; gid?: number; mode?: number }

type Props = { namespace: string; pvc: string }

export function FilePanel({ namespace, pvc }: Props) {
  const [path, setPath] = useState<string>('/')
  const [entries, setEntries] = useState<Entry[]>([])
  const [total, setTotal] = useState<number>(0)
  const [offset, setOffset] = useState<number>(0)
  const [preview, setPreview] = useState<Entry|null>(null)
  const [error, setError] = useState<string>('')
  const [progress, setProgress] = useState<number>(0)
  const limit = 200

  useEffect(() => { setPath('/') }, [namespace, pvc])

  useEffect(() => {
    if (!namespace || !pvc) return
    const q = new URLSearchParams({ ns: namespace, pvc, path, limit: String(limit), offset: String(offset) })
    fetch(`/api/v1/tree?${q.toString()}`)
      .then(async r => {
        if (!r.ok) throw new Error(`API ${r.status}`)
        setTotal(Number(r.headers.get('X-Total-Count')||'0'))
        try { return await r.json() } catch { return [] }
      })
      .then(setEntries).catch(e=>setError(String(e)))
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

  return (
    <div className="h-full flex flex-col bg-white dark:bg-gray-950">
      <div className="border-b border-gray-200 dark:border-gray-800 px-4 py-2 flex items-center gap-2 text-sm sticky top-0 bg-white/70 dark:bg-gray-950/70 backdrop-blur z-10">
        {breadcrumbs.map((b, i) => (
          <span key={b.path}>
            {i>0 && <span className="opacity-50">/</span>}<button className="hover:underline" onClick={()=>{setPath(b.path); setOffset(0)}}>{b.name}</button>
          </span>
        ))}
      </div>
      <div className="flex-1 overflow-auto">
        <table className="w-full text-sm">
          <thead className="text-left sticky top-0 bg-gray-50 dark:bg-gray-900">
            <tr><th className="p-2">Name</th><th>Size</th><th>Modified</th><th>Owner</th><th>Group</th><th>Mode</th><th></th></tr>
          </thead>
          <tbody>
            {entries.map(e => (
              <tr key={e.path} className="border-b border-gray-100 dark:border-gray-900 hover:bg-gray-50 dark:hover:bg-gray-900/70 transition">
                <td className="p-2">
                  {e.isDir ? (
                    <button className="text-blue-600 dark:text-blue-400 font-medium" onClick={()=>{setPath(e.path); setOffset(0)}}>{e.name}</button>
                  ) : e.name}
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
                  />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {preview && (
        <div className="border-t border-gray-200 dark:border-gray-900 p-3">
          <div className="font-medium mb-2">Preview: {preview.name}</div>
        </div>
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

function formatMode(mode?: number) {
  if (mode==null) return '-'
  const m = mode & 0o777
  const to = (n:number)=>[(n&4?'r':'-'),(n&2?'w':'-'),(n&1?'x':'-')].join('')
  return to((m>>6)&7)+' '+to((m>>3)&7)+' '+to(m&7)
}



