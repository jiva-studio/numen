import { describe, expect, it } from 'vitest'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, FilesTab, FILES } from './index'
import { createHost } from './fixtures/host'
import type { Entry, FilesTabState } from './types'

/** A moment for whatever the tab asked the vault for to come back. */
const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const note = (path: string): Entry => ({
  path,
  name: path,
  isFolder: false,
  kind: 'note',
  type: 'note',
})

const createContext = () => {
  const folders: Record<string, readonly Entry[]> = { '': [note('A.md')] }
  const { host, recorded } = createHost({ folders })
  const ctx = createMockPluginContext({
    pluginId: 'md.numen.views.file.tree',
    workspace: host.workspace,
    files: host.files,
  })
  return { ctx, recorded, folders }
}

describe('view-file-tree plugin', () => {
  it('registers the files tab kind upon activation', () => {
    const { ctx } = createContext()
    activate(ctx)

    const tabKinds = ctx.getRegisteredTabKinds()
    expect(tabKinds).toHaveLength(1)
    expect(tabKinds[0]?.kind).toBe(FILES)
    expect(tabKinds[0]?.component).toBe(FilesTab)
    expect(tabKinds[0]?.open).toBeTypeOf('function')
  })

  it('holds two subscriptions, the tab kind and the vault, and releases both', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.subscriptions).toHaveLength(2)

    ctx.dispose()
    expect(ctx.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('reads the folders of every files tab again when the vault changes', async () => {
    const { ctx, folders } = createContext()
    activate(ctx)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('', 'files') as FilesTabState
    await settle()
    expect(state.list.rows.value.map((row) => row.entry.path)).toStrictEqual(['A.md'])

    folders[''] = [note('B.md')]
    ctx.emitVaultChange({ paths: [], renamed: [{ from: 'A.md', to: 'B.md' }] })
    await settle()

    expect(state.list.rows.value.map((row) => row.entry.path)).toStrictEqual(['B.md'])
  })

  it('refuses to activate in a window that offers no workspace or files', () => {
    expect(() =>
      activate(createMockPluginContext({ pluginId: 'md.numen.views.file.tree' })),
    ).toThrow('a files tab needs the window it is drawn in and the files of the vault')
  })

  it('activates through the default plugin lifecycle', async () => {
    const { ctx } = createContext()
    await plugin.activate(ctx)

    expect(ctx.getRegisteredTabKinds()[0]?.kind).toBe(FILES)
  })
})
