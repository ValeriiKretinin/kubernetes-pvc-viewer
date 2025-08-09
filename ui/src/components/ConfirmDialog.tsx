type Props = {
  open: boolean
  title: string
  description?: string
  confirmText?: string
  cancelText?: string
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({ open, title, description, confirmText = 'Confirm', cancelText = 'Cancel', onConfirm, onCancel }: Props) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-[10000] flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onCancel} />
      <div className="relative w-full max-w-md mx-4 rounded-xl border border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900 shadow-2xl p-5">
        <div className="text-lg font-semibold text-strong mb-1">{title}</div>
        {description && <div className="text-sm text-muted-weak mb-4">{description}</div>}
        <div className="flex justify-end gap-2">
          <button className="px-3 py-1.5 rounded-md border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800" onClick={onCancel}>{cancelText}</button>
          <button className="px-3 py-1.5 rounded-md border border-transparent bg-blue-600 text-white hover:bg-blue-500" onClick={onConfirm}>{confirmText}</button>
        </div>
      </div>
    </div>
  )
}


