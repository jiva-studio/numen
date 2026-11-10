import { describe, expect, it } from 'vitest'
import { SLOT, type TabKindRegistration } from '@numen/plugin-sdk'
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
  ctx.ui.slots.setEntries(SLOT.fileTreeContext, ROW_MENU)
  return { ctx, recorded, folders }
}

const createWindow = () => {
  let getKinds = (): readonly TabKindRegistration[] => []
  const tabOpens: (readonly [string, string])[] = []
  const tabs: { kind: string; state: object }[] = []
  const navigateTab = async (kind: string, path: string): Promise<string> => {
    const registration = getKinds().find((one) => one.kind === kind)!
    const last = tabs.filter((one) => one.kind === kind).at(-1)
    if (last) {
      await registration.navigate?.(last.state, path, '')
      return kind
    }
    tabOpens.push([kind, path])
    tabs.push({ kind, state: registration.open!(path, `${kind}-${tabs.length}`) })
    return kind
  }
  const bind = (ctx: { getRegisteredTabKinds(): TabKindRegistration[] }): void => {
    getKinds = () => ctx.getRegisteredTabKinds()
  }
  return { tabOpens, tabs, navigateTab, bind }
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

  it('holds the tab kind, the vault and its two commands, and releases them all', () => {
    const { ctx } = createContext()
    activate(ctx)

    expect(ctx.plugin.subscriptions).toHaveLength(4)

    ctx.dispose()
    expect(ctx.plugin.subscriptions).toHaveLength(0)
    expect(ctx.getRegisteredTabKinds()).toHaveLength(0)
    expect(ctx.getRegisteredCommands()).toHaveLength(0)
  })

  const createWindowContext = () => {
    const { host } = createHost({ folders: { '': [note('A.md'), note('B.md')] } })
    const strip = createWindow()
    const ctx = createMockPluginContext({
      pluginId: 'md.numen.views.file.tree',
      workspace: host.workspace,
      files: host.files,
      navigateTab: strip.navigateTab,
    })
    strip.bind(ctx)
    activate(ctx)
    return { ctx, strip }
  }

  it('registers the commands of the files', () => {
    const { ctx } = createWindowContext()

    expect(ctx.getRegisteredCommands().map((one) => one.id)).toStrictEqual([
      'tree.open',
      'tree.renameFile',
    ])
  })

  it('opens one files tab however many times the command runs', async () => {
    const { ctx, strip } = createWindowContext()

    await ctx.triggerCommand('tree.open')
    await ctx.triggerCommand('tree.open')
    await ctx.triggerCommand('tree.open', {
      path: 'B.md',
      title: 'B.md',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(strip.tabOpens).toStrictEqual([[FILES, '']])
    expect(strip.tabs).toHaveLength(1)
  })

  it('shows the active note in a files tab that is already open', async () => {
    const { ctx, strip } = createWindowContext()
    await ctx.triggerCommand('tree.open')
    await settle()

    await ctx.triggerCommand('tree.open', {
      path: 'B.md',

      title: 'B.md',

      type: 'note',

      isFolder: false,

      others: [],

      tab: '',
    })

    const state = strip.tabs[0]!.state as FilesTabState
    expect(state.list.state.selectedPaths.value).toStrictEqual(['B.md'])
  })

  it('shows the active note in the files tab it opens', async () => {
    const { ctx, strip } = createWindowContext()

    await ctx.triggerCommand('tree.open', {
      path: 'A.md',

      title: 'A.md',

      type: 'note',

      isFolder: false,

      others: [],

      tab: '',
    })
    await settle()

    expect(strip.tabOpens).toStrictEqual([[FILES, 'A.md']])
    const state = strip.tabs[0]!.state as FilesTabState
    expect(state.list.state.selectedPaths.value).toStrictEqual(['A.md'])
  })

  it('selects nothing in a files tab when no note is active', async () => {
    const { ctx, strip } = createWindowContext()
    await ctx.triggerCommand('tree.open')
    await settle()

    await ctx.triggerCommand('tree.open')

    const state = strip.tabs[0]!.state as FilesTabState
    expect(state.list.state.selectedPaths.value).toStrictEqual([])
  })

  it('reads the folders of every files tab again when the vault changes', async () => {
    const { ctx, folders } = createContext()
    activate(ctx)
    const state = ctx.getRegisteredTabKinds()[0]!.open!('', 'files') as FilesTabState
    await settle()
    expect(state.list.queries.rows.value.map((row) => row.entry.path)).toStrictEqual(['A.md'])

    folders[''] = [note('B.md')]
    ctx.events.emitVaultChange({ paths: [], renamed: [{ from: 'A.md', to: 'B.md' }] })
    await settle()

    expect(state.list.queries.rows.value.map((row) => row.entry.path)).toStrictEqual(['B.md'])
  })

  it('draws the menu of a row from the entries of its slot, and runs their commands', async () => {
    const { ctx, state } = await openTab()

    state.menu.open({ path: 'A.md', at: { x: 0, y: 0 } })
    expect(state.menu.getItems('A.md').map((one) => one.id)).toContain(TRASH.key)
    state.menu.choose(TRASH.key)

    expect(ctx.ui.commands.getRuns()).toStrictEqual([
      {
        command: 'app:file.remove',
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
    state.menu.open({ path: 'A.md', at: { x: 0, y: 0 } })

    await ctx.triggerCommand('tree.renameFile', {
      path: 'A.md',
      title: 'A.md',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(state.renaming.path.value).toBe('A.md')
  })

  it('puts nothing in a field for the root', async () => {
    const { ctx, state } = await openTab()
    state.menu.open({ path: null, at: { x: 0, y: 0 } })

    await ctx.triggerCommand('tree.renameFile', {
      path: '',
      title: '',
      type: 'note',
      isFolder: false,
      others: [],
      tab: '',
    })

    expect(state.renaming.path.value).toBeNull()
  })

  it('has no command to make a folder, which the Create folder entry runs through the window', async () => {
    const { ctx, state } = await openTab()
    ctx.ui.commands.setResult('app:folder.create', 'New folder')

    state.menu.open({ path: null, at: { x: 0, y: 0 } })
    state.menu.choose(NEW_MENU.entries[1]!.key)
    await settle()

    expect(ctx.getRegisteredCommands().map((one) => one.id)).not.toContain('tree.createFolder')
    expect(ctx.ui.commands.getRuns().map((one) => one.command)).toStrictEqual(['app:folder.create'])
    expect(state.renaming.path.value).toBe('New folder')
  })

  it('has no command to make a note, which the New note entry runs through the window', async () => {
    const { ctx, state } = await openTab()
    ctx.ui.commands.setResult('app:note.create', 'Untitled note 1.md')

    state.menu.open({ path: null, at: { x: 0, y: 0 } })
    state.menu.choose(NEW_MENU.entries[0]!.key)
    await settle()

    expect(ctx.getRegisteredCommands().map((one) => one.id)).not.toContain('tree.createNote')
    expect(ctx.ui.commands.getRuns().map((one) => one.command)).toStrictEqual(['app:note.create'])
    expect(state.renaming.path.value).toBe('Untitled note 1.md')
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
