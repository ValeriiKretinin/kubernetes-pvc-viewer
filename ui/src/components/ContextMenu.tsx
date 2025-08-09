import { Fragment, useEffect } from 'react'
import { Menu, Transition } from '@headlessui/react'
import { clsx } from 'clsx'
import { ArrowDownTrayIcon, TrashIcon, InformationCircleIcon, ArrowUpOnSquareIcon } from '@heroicons/react/24/outline'

type Props = { onDownload?: () => void; onDelete?: () => void; onUpload?: () => void; onInfo?: () => void; onOpenChange?: (open: boolean)=>void }

export function ContextMenu({ onDownload, onDelete, onUpload, onInfo, onOpenChange }: Props) {
  return (
    <Menu as="div" className="relative inline-block text-left">
      {({ open }) => {
        useEffect(()=>{ onOpenChange?.(open) }, [open])
        return (
          <>
            <Menu.Button className="px-2 py-1 border rounded">Actions</Menu.Button>
            <Transition as={Fragment} enter="transition ease-out duration-100" enterFrom="transform opacity-0 scale-95" enterTo="transform opacity-100 scale-100" leave="transition ease-in duration-75" leaveFrom="transform opacity-100 scale-100" leaveTo="transform opacity-0 scale-95">
              <Menu.Items className="absolute right-0 z-[999] mt-2 w-44 origin-top-right rounded-md bg-white dark:bg-gray-800 shadow-2xl ring-1 ring-black/10 dark:ring-white/10 focus:outline-none">
          {onInfo && <Menu.Item>{({ active }) => (
            <button className={clsx('w-full text-left px-3 py-2 flex items-center gap-2', active && 'bg-gray-100 dark:bg-gray-700')} onClick={onInfo}>
              <InformationCircleIcon className="w-4 h-4" /> Info
            </button>
          )}</Menu.Item>}
          {onDownload && <Menu.Item>{({ active }) => (
            <button className={clsx('w-full text-left px-3 py-2 flex items-center gap-2', active && 'bg-gray-100 dark:bg-gray-700')} onClick={onDownload}>
              <ArrowDownTrayIcon className="w-4 h-4" /> Download
            </button>
          )}</Menu.Item>}
          {onUpload && <Menu.Item>{({ active }) => (
            <button className={clsx('w-full text-left px-3 py-2 flex items-center gap-2', active && 'bg-gray-100 dark:bg-gray-700')} onClick={onUpload}>
              <ArrowUpOnSquareIcon className="w-4 h-4" /> Upload
            </button>
          )}</Menu.Item>}
          {onDelete && <Menu.Item>{({ active }) => (
            <button className={clsx('w-full text-left px-3 py-2 text-red-600 flex items-center gap-2', active && 'bg-gray-100 dark:bg-gray-700')} onClick={onDelete}>
              <TrashIcon className="w-4 h-4" /> Delete
            </button>
          )}</Menu.Item>}
              </Menu.Items>
            </Transition>
          </>
        )
      }}
    </Menu>
  )
}


