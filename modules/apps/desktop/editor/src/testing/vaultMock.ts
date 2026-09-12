/**
 * The vault the window reaches, answering out of `said` and writing down what
 * it was asked in `requests`.
 */
import { vi } from 'vitest'
import { requests } from './requests'
import { folders, getFileKind, held, listed, outside, said } from './answers'
import type { Tab } from '@/entities/tab'

vi.mock('@/app/vault', () => ({
  vaults: {
    list: async () => listed,
    choose: async () => {
      requests.chose += 1
      return ''
    },
    add: async () => ({ vault: null, error: null }),
    rename: async () => ({ vault: null, error: null }),
    remove: async () => null,
    open: async (id: string) => {
      requests.opened.push(id)
      return null
    },
  },
  core: {
    vaults: async () => {
      if (!said.listable) throw new Error('the vaults are not there')
      return listed
    },
    state: async () => ({
      name: 'Vault',
      path: '/vaults/Physics',
      scan: {
        isReady: said.ready,
        error: said.error,
        unwatchedPath: '',
      },
      coverage: {
        chunkCount: 0n,
        embeddedCount: BigInt(said.embedded),
        isEmbedding: false,
      },
    }),
    agentUnreachable: async () => '',
    getInitialOpenPath: async () => (said.opening ? { path: said.opening } : null),
    neighbourhood: async (path: string) => ({
      focus: { path, title: path.replace(/\.md$/, '') },
      related: [],
    }),
    read: async () => ({ body: 'what is written', at: 'a1' }),
    write: async () => ({ at: 'a2' }),
    create: async ({ title }: { title: string }) => {
      requests.made.push(title)
      return { path: `${title}.md`, error: null }
    },
    rename: async (path: string, title: string) => {
      requests.renamed.push(`${path} ${title}`)
      return { path, title, by: 'frontmatter', moved: null, error: null }
    },
    remove: async (path: string, destroy?: boolean) => {
      requests.removed.push(`${path} ${destroy ?? false}`)
      return { trashed: `.trash/${path}`, dangling: [], error: null }
    },
    list: async (folder: string) => folders[folder] ?? [],
    move: async (from: string, to: string) => {
      requests.moved.push(`${from} ${to}`)
      return { moved: null, error: null }
    },
    createFolder: async (path: string) => {
      requests.folders.push(path)
      return null
    },
    createUrl: async (url: string, folder: string) => {
      requests.urls.push(`${url} ${folder}`)
      return { path: folder ? `${folder}/made.url` : 'made.url', error: null }
    },
    changes: held,
    editing: held,
    tasks: held,
    focus: outside.stream,
    writeOpenTabs: async (open: { tabs: readonly Tab[]; front: string }) => {
      requests.openTabs.push(open)
    },
    quitting: held,
    flushed: async () => {},
    names: async () => said.names,
    search: async () => said.passages,
    fileKinds: async (paths: readonly string[]) =>
      new Map(paths.map((path) => [path, getFileKind(path)])),
    headings: async () => new Map(),
  },
}))
