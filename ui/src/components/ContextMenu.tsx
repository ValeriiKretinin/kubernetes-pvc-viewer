import { Fragment } from 'react'
import { Menu, Transition } from '@headlessui/react'
import { clsx } from 'clsx'

type Props = { onDownload?: () => void; onDelete?: () => void; onUpload?: () => void }

export function ContextMenu({ onDownload, onDelete, onUpload }: Props) {
  return (
    <Menu as="div" className="relative inline-block text-left">
      <Menu.Button className="px-2 py-1 border rounded">Actions</Menu.Button>
      <Transition as={Fragment} enter="transition ease-out duration-100" enterFrom="transform opacity-0 scale-95" enterTo="transform opacity-100 scale-100" leave="transition ease-in duration-75" leaveFrom="transform opacity-100 scale-100" leaveTo="transform opacity-0 scale-95">
        <Menu.Items className="absolute right-0 z-10 mt-2 w-40 origin-top-right rounded-md bg-white dark:bg-gray-800 shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none">
          {onDownload && <Menu.Item>{({ active }) => (
            <button className={clsx('w-full text-left px-3 py-2', active && 'bg-gray-100 dark:bg-gray-700')} onClick={onDownload}>Download</button>
          )}</Menu.Item>}
          {onUpload && <Menu.Item>{({ active }) => (
            <button className={clsx('w-full text-left px-3 py-2', active && 'bg-gray-100 dark:bg-gray-700')} onClick={onUpload}>Upload</button>
          )}</Menu.Item>}
          {onDelete && <Menu.Item>{({ active }) => (
            <button className={clsx('w-full text-left px-3 py-2 text-red-600', active && 'bg-gray-100 dark:bg-gray-700')} onClick={onDelete}>Delete</button>
          )}</Menu.Item>}
        </Menu.Items>
      </Transition>
    </Menu>
  )
}


