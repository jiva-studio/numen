/**
 * The vault mock the window uses, responding from `vaultState` and recording requests
 * in `requests`.
 */
import { vi } from 'vitest'
import { asValue } from '@numen/core-client'
import { requests } from './requests'
import {
  folders,
  getSettingInForce,
  getFileKind,
  keepSetting,
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
    list: () => Promise.resolve(mockVaultList),
    choose: () => {
      requests.chose += 1
      return Promise.resolve('')
    },
    add: () => Promise.resolve(asValue(null)),
    rename: () => Promise.resolve(asValue(null)),
    remove: () => Promise.resolve(null),
    open: (id: string) => {
      requests.opened.push(id)
      return Promise.resolve(null)
    },
  } satisfies Partial<Vaults>,
  core: {
    vaults: () =>
      mockVaultState.listable
        ? Promise.resolve(mockVaultList)
        : Promise.reject(new Error('the vaults are not there')),
    state: () =>
      Promise.resolve({
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
    agentUnreachable: () => Promise.resolve(''),
    getInitialOpenPath: () =>
      Promise.resolve(mockVaultState.opening ? { path: mockVaultState.opening } : null),
    neighbourhood: (path: string) =>
      Promise.resolve({
        focus: { path, title: path.replace(/\.md$/, '') },
        focusType: 'note' as const,
        related: [],
        connections: [],
      }),
    read: () => Promise.resolve(asValue({ body: 'what is written', at: 'a1' })),
    write: () => Promise.resolve(asValue({ body: '', at: 'a2' })),
    create: ({ title }: { title: string }) => {
      requests.made.push(title)
      return Promise.resolve(asValue({ path: `${title}.md` }))
    },
    rename: (path: string, title: string) => {
      requests.renamed.push(`${path} ${title}`)
      return Promise.resolve(asValue({ path, title, hasFrontmatter: true, moved: null }))
    },
    remove: (path: string, destroy?: boolean) => {
      requests.removed.push(`${path} ${destroy ?? false}`)
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
    getSettings: () =>
      Promise.resolve({
        written: '{}',
        path: '/vaults/Physics/.numen/settings.json',
        models: [],
        plugins: [],
        app: mockVaultState.app.map(getSettingInForce),
      }),
    updateSettings: (edits: readonly { at: readonly string[]; value: string }[]) => {
      if (mockVaultState.writeError) return Promise.reject(new Error(mockVaultState.writeError))
      for (const one of edits) {
        requests.settingsWritten.push(`${one.at.join('.')} ${one.value}`)
        keepSetting(one.at.join('.'), JSON.parse(one.value))
      }
      return Promise.resolve()
    },
    watchSettings: pendingStream,
    watchVaultChanges: pendingStream,
    watchEdits: pendingStream,
    watchTasks: pendingStream,
    watchFocus: outside.stream,
    writeOpenTabs: (open: { tabs: readonly Tab[]; activeId: string }) => {
      requests.openTabs.push(open)
      return Promise.resolve()
    },
    readOpenTabs: () => Promise.resolve(mockVaultState.savedTabs),
    watchQuit: pendingStream,
    reportFlush: () => Promise.resolve(),
    detachTab: () =>
      mockVaultState.detachError
        ? Promise.reject(new Error(mockVaultState.detachError))
        : Promise.resolve({ ok: true as const, value: undefined }),
    watchTabLinks: pendingStream,
    cancelTabLinksWatch: () => Promise.resolve(),
    writeTabLinks: () => Promise.resolve(),
    syncTabPath: () => Promise.resolve(),
    readDetachedTab: (_windowId: string, tabId: string) =>
      Promise.resolve(
        mockVaultState.detachedTab?.id === tabId
          ? { tab: mockVaultState.detachedTab, moveId: 'move-1' }
          : null,
      ),
    moveTab: () => Promise.resolve({ ok: true, value: undefined }),
    acceptTabMove: () => Promise.resolve({ ok: true, value: undefined }),
    highlightWindow: () => Promise.resolve(),
    names: () => Promise.resolve(mockVaultState.names),
    search: () => Promise.resolve(mockVaultState.passages),
    fileKinds: (paths: readonly string[]) =>
      Promise.resolve(new Map(paths.map((path) => [path, getFileKind(path)]))),
    headings: () => Promise.resolve(new Map()),
  } satisfies Partial<VaultCore>,
}))
