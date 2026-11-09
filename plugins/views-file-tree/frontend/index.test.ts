import { describe, expect, it } from 'vitest'
import { SLOT } from '@numen/plugin-sdk'
import { createMockPluginContext } from '@numen/plugin-sdk/testing'
import plugin, { activate, FilesTab, FILES } from './index'
import { createHost, NEW_MENU, ROW_MENU, TRASH } from './fixtures/host'
import type { Entry, FilesTabState } from './types'

const settle = () => new Promise((isDone) => setTimeout(isDone, 0))

const note = (path: string): Entry => ({
  path,
  name: path,
  isFolder: false,
  isNote: true,
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
  ctx.slots.setEntries(SLOT.fileTreeContext, ROW_MENU)
  return { ctx, recorded, folders }
}

const openTab = async () => {
  const { ctx, recorded, folders } = createContext()
  activate(ctx)
  const state = ctx.getRegisteredTabKinds()[0]!.open!('', 'files') as FilesTabState
  await settle()
  return { ctx, recorded, folders, state }
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

  it('holds the tab kind, the vault and its four commands, and releases them all', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.subscriptions).toHaveLength(6)

    ctx.dispose()
    expect(ctx.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredCommands()).toHaveLength(0)
  })

  it('puts the files in front, and a note shown in them', async () => {
    const openCalls: (readonly [string, string | undefined])[] = []
    const navigateCalls: (readonly [string, string])[] = []
    const { host } = createHost({ folders: { '': [] } })
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.file.tree',
      workspace: host.workspace,
      files: host.files,
      openTab: (kind, path) => {
        openCalls.push([kind, path])
        return Promise.resolve('')
      },
      navigateTab: (kind, path) => {
        navigateCalls.push([kind, path])
        return Promise.resolve('')
      },
    })
    activate(ctx)

    expect(ctx.getRegisteredCommands().map((one) => one.id)).toStrictEqual([
      'tree.open',
      'tree.revealNote',
      'tree.renameFile',
      'tree.createFolder',
    ])
    await ctx.triggerCommand('tree.open')
    await ctx.triggerCommand('tree.revealNote', {
      path: 'notes/A.md',
      title: 'A',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(openCalls).toStrictEqual([[FILES, undefined]])
    expect(navigateCalls).toStrictEqual([[FILES, 'notes/A.md']])
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

  it('draws the menu of a row from the entries of its slot, and runs their commands', async () => {
    const { ctx, state } = await openTab()

    state.openMenu({ path: 'A.md', at: { x: 0, y: 0 } })
    expect(state.getMenuItems('A.md').map((one) => one.id)).toContain(TRASH.key)
    state.chooseMenuItem(TRASH.key)

    expect(ctx.commands.getRuns()).toStrictEqual([
      {
        command: 'app:note.trash',
        target: {
          path: 'A.md',
          title: 'A.md',
          isFolder: false,
          others: [],
          tab: '',
          noteType: 'note',
        },
        itemId: undefined,
      },
    ])
  })

  it('puts the name of the row the menu was opened on in a field', async () => {
    const { ctx, state } = await openTab()
    state.openMenu({ path: 'A.md', at: { x: 0, y: 0 } })

    await ctx.triggerCommand('tree.renameFile', {
      path: 'A.md',
      title: 'A.md',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(state.renamingPath.value).toBe('A.md')
  })

  it('puts nothing in a field for the root', async () => {
    const { ctx, state } = await openTab()
    state.openMenu({ path: null, at: { x: 0, y: 0 } })

    await ctx.triggerCommand('tree.renameFile', {
      path: '',
      title: '',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(state.renamingPath.value).toBeNull()
  })

  it('makes a folder at the row the menu was opened on', async () => {
    const { ctx, recorded, state } = await openTab()
    state.openMenu({ path: 'A.md', at: { x: 0, y: 0 } })

    await ctx.triggerCommand('tree.createFolder', {
      path: '',
      title: '',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(recorded.made).toStrictEqual(['folder New folder'])
  })

  it('has no command to make a note, which the New note entry runs through the window', async () => {
    const { ctx, state } = await openTab()
    ctx.commands.setResult('app:note.create', 'Untitled note 1.md')

    state.openMenu({ path: null, at: { x: 0, y: 0 } })
    state.chooseMenuItem(NEW_MENU.entries[0]!.key)
    await settle()

    expect(ctx.getRegisteredCommands().map((one) => one.id)).not.toContain('tree.createNote')
    expect(ctx.commands.getRuns().map((one) => one.command)).toStrictEqual(['app:note.create'])
    expect(state.renamingPath.value).toBe('Untitled note 1.md')
  })

  it('does nothing for a command of the tree while no files tab is open', async () => {
    const { ctx, recorded } = createContext()
    activate(ctx)

    await ctx.triggerCommand('tree.renameFile', {
      path: 'A.md',
      title: 'A.md',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(recorded.made).toStrictEqual([])
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
