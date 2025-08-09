import { useEffect, useMemo, useState } from 'react'

type Entry = { name: string; path: string; isDir: boolean; size: number; mod: string }

type Props = { entry: Entry; namespace: string; pvc: string; onClose: ()=>void }

export function PreviewPane({ entry, namespace, pvc, onClose }: Props) {
  const [blobUrl, setBlobUrl] = useState<string>('')
  const url = useMemo(() => `/api/v1/download?ns=${encodeURIComponent(namespace)}&pvc=${encodeURIComponent(pvc)}&path=${encodeURIComponent(entry.path)}`, [namespace, pvc, entry.path])
  const ext = entry.name.split('.').pop()?.toLowerCase() || ''

  useEffect(() => {
    let revoked = false
    const load = async () => {
      try {
        const r = await fetch(url)
        const b = await r.blob()
        const u = URL.createObjectURL(b)
        if (!revoked) setBlobUrl(u)
      } catch {}
    }
    load()
    return () => { revoked = true; if (blobUrl) URL.revokeObjectURL(blobUrl) }
  }, [url])

  const isImage = ['png','jpg','jpeg','gif','webp'].includes(ext)
  const isPdf = ext === 'pdf'
  const isText = ['txt','log','json','yaml','yml','md','csv'].includes(ext)

  return (
    <div className="border-t border-gray-200 dark:border-gray-800 p-3 flex gap-3 items-start">
      <div className="font-medium text-strong">Preview: {entry.name}</div>
      <button className="ml-auto btn" onClick={onClose}>Close</button>
      <div className="w-full">
        {isImage && blobUrl && <img src={blobUrl} alt={entry.name} className="max-h-96 rounded shadow" />}
        {isPdf && blobUrl && <iframe src={blobUrl} className="w-full h-96 rounded shadow bg-white" />}
        {isText && blobUrl && <TextViewer url={url} />}
        {!isImage && !isPdf && !isText && (
          <div className="text-sm text-muted">No preview available for this file type.</div>
        )}
      </div>
    </div>
  )
}

function TextViewer({ url }: { url: string }) {
  const [text, setText] = useState<string>('')
  useEffect(() => { fetch(url).then(r=>r.text()).then(setText).catch(()=>{}) }, [url])
  return (
    <pre className="bg-gray-50 dark:bg-gray-900 p-3 rounded shadow max-h-96 overflow-auto text-sm whitespace-pre-wrap text-strong">{text}</pre>
  )
}


