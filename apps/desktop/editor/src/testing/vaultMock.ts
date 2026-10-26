/**
 * The vault the window reaches, answering out of `said` and writing down what
 * it was asked in `requests`.
 */
import { vi } from 'vitest'
import { asValue } from '@numen/wire'
import { requests } from './requests'
import {
  folders,
  getFileKind,
  mockVaultList,
  mockVaultState,
  outside,
  pendingStream,
} from './answers'
import type { Tab } from '@/entities/tab'
import type { VaultCore } from '@/app/vault/core'
import type { Vaults } from '@/entities/vault'

vi.mock('@/app/vault', () => ({
  vaults: {
    list: async () => mockVaultList,
    choose: async () => {
      requests.chose += 1
      return ''
    },
    add: async () => asValue(null),
    rename: async () => asValue(null),
    remove: async () => null,
    open: async (id: string) => {
      requests.opened.push(id)
      return null
    },
  } satisfies Partial<Vaults>,
  core: {
    vaults: async () => {
      if (!mockVaultState.listable) throw new Error('the vaults are not there')
      return mockVaultList
    },
    state: async () => ({
      id: 'physics',
      name: 'Vault',
      path: '/vaults/Physics',
      scan: {
        isReady: mockVaultState.isReady,
        error: mockVaultState.error,
        unwatchedPath: '',
      },
      coverage: {
        chunkCount: 0n,
        embeddedCount: BigInt(mockVaultState.embedded),
        isEmbedding: false as boolean,
      },
    }),
    agentUnreachable: async () => '',
    getInitialOpenPath: async () =>
      mockVaultState.opening ? { path: mockVaultState.opening } : null,
    neighbourhood: async (path: string) => ({
      focus: { path, title: path.replace(/\.md$/, '') },
      focusType: 'note' as const,
      related: [],
    }),
    read: async () => asValue({ body: 'what is written', at: 'a1' }),
    write: async () => asValue({ body: '', at: 'a2' }),
    create: async ({ title }: { title: string }) => {
      requests.made.push(title)
      return asValue({ path: `${title}.md` })
    },
    rename: async (path: string, title: string) => {
      requests.renamed.push(`${path} ${title}`)
      return asValue({ path, title, hasFrontmatter: true, moved: null })
    },
    remove: async (path: string, destroy?: boolean) => {
      requests.removed.push(`${path} ${destroy ?? false}`)
      return asValue({ trashed: `.trash/${path}`, dangling: [] })
    },
    list: async (folder: string) => folders[folder] ?? [],
    move: async (from: string, to: string) => {
      requests.moved.push(`${from} ${to}`)
      return asValue(null)
    },
    createFolder: async (path: string) => {
      requests.folders.push(path)
      return null
    },
    watchVaultChanges: pendingStream,
    watchEdits: pendingStream,
    watchTasks: pendingStream,
    watchFocus: outside.stream,
    writeOpenTabs: async (open: { tabs: readonly Tab[]; front: string }) => {
      requests.openTabs.push(open)
    },
    watchQuit: pendingStream,
    reportFlush: async () => {},
    detachTab: async () => {
      if (mockVaultState.detachError) throw new Error(mockVaultState.detachError)
    },
    names: async () => mockVaultState.names,
    search: async () => mockVaultState.passages,
    fileKinds: async (paths: readonly string[]) =>
      new Map(paths.map((path) => [path, getFileKind(path)])),
    headings: async () => new Map(),
  } satisfies Partial<VaultCore>,
}))
