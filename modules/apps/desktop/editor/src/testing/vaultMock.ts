/**
 * The vault the window reaches, answering out of `said` and writing down what
 * it was asked in `asked`.
 */
import { vi } from 'vitest'
import { asked } from './asked'
import { folders, getFileKind, held, listed, outside, said } from './answers'
import type { Tab } from '@/entities/tab'

vi.mock('@/app/vault', () => ({
  vaults: {
    list: async () => listed,
    choose: async () => {
      asked.chose += 1
      return ''
    },
    add: async () => ({ vault: null, error: null }),
    rename: async () => ({ vault: null, error: null }),
    remove: async () => null,
    open: async (id: string) => {
      asked.opened.push(id)
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
      asked.made.push(title)
      return { path: `${title}.md`, error: null }
    },
    rename: async (path: string, title: string) => {
      asked.renamed.push(`${path} ${title}`)
      return { path, title, by: 'frontmatter', moved: null, error: null }
    },
    remove: async (path: string, destroy?: boolean) => {
      asked.removed.push(`${path} ${destroy ?? false}`)
      return { trashed: `.trash/${path}`, dangling: [], error: null }
    },
    list: async (folder: string) => folders[folder] ?? [],
    move: async (from: string, to: string) => {
      asked.moved.push(`${from} ${to}`)
      return { moved: null, error: null }
    },
    createFolder: async (path: string) => {
      asked.folders.push(path)
      return null
    },
    createUrl: async (url: string, folder: string) => {
      asked.urls.push(`${url} ${folder}`)
      return { path: folder ? `${folder}/made.url` : 'made.url', error: null }
    },
    changes: held,
    editing: held,
    tasks: held,
    focus: outside.stream,
    writeOpenTabs: async (open: { tabs: readonly Tab[]; front: string }) => {
      asked.openTabs.push(open)
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
