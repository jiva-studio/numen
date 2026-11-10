/**
 * The file entity's part of the vault mock: the files and folders of the vault.
 */
import { asValue } from '@numen/core-client'
import { requests } from '../requests'
import { folders, getFileKind } from '../answers'
import type { VaultCore } from '@/app/vault/core'

export const fileCoreMock = {
  remove: (path: string) => {
    requests.removed.push(path)
    return Promise.resolve(asValue({ trashed: `.trash/${path}`, dangling: [] }))
  },
  list: (folder: string) => Promise.resolve(folders[folder] ?? []),
  move: (from: string, to: string) => {
    requests.moved.push(`${from} ${to}`)
    return Promise.resolve(asValue(null))
  },
  createFolder: (path: string) => {
    requests.folders.push(path)
    return Promise.resolve(null)
  },
  fileKinds: (paths: readonly string[]) =>
    Promise.resolve(new Map(paths.map((path) => [path, getFileKind(path)]))),
} satisfies Partial<VaultCore>
