/**
 * The window put together: what each part is handed of the others, and what
 * the window answers about the tab in front.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getNamedIcon } from '@/shared/icons'
import { ref } from 'vue'
import { useWindow } from './useWindow'

/** What each part was made with, and what it was made as. */
const h = vi.hoisted(() => ({
  args: {} as Record<string, any>,
  f: {} as Record<string, any>,
  core: {
    name: 'core',
    fileKinds: vi.fn(),
    readOpenTabs: vi.fn(async () => ({ tabs: [], activeId: '' })),
    resolveLinks: vi.fn(async () => [{ path: 'resolved.md' }]),
    cancelTask: vi.fn(async () => undefined),
    detachTab: vi.fn(),
    writeTabLinks: vi.fn(async () => undefined),
    syncTabPath: vi.fn(async () => undefined),
    readDetachedTab: vi.fn(async (): Promise<unknown> => null),
    highlightWindow: vi.fn(async () => undefined),
  },
}))

vi.mock('../vault', () => ({ core: h.core }))
vi.mock('@/features/command-palette', () => ({
  runSupport: () => ({ name: 'runs' }),
  createCommands: () => [{ id: 'app:vault.create' }],
}))
vi.mock('@/features/tab-linking', () => ({
  useTabLinks: (args: unknown) => {
    h.args.tabLinks = args
    return h.f.tabLinks
  },
}))
vi.mock('@/shared/notices/messages', () => ({ createMessageLog: () => h.f.log }))
vi.mock('@/entities/tab', () => ({
  createWorkspace: vi.fn(),
  fileOpeners: (args: unknown) => {
    h.args.openers = args
    return h.f.tabOpeners
  },
  useWindowTabs: (args: unknown) => {
    h.args.windowTabs = args
    return h.f.windowTabs
  },
}))
vi.mock('../tabs/useTabSync', () => ({
  createTabSync: (args: unknown) => {
    h.args.tabSync = args
    h.args.tabSyncResult = { name: 'tabSync', applySync: vi.fn(), applyGroupPaths: vi.fn() }
    return h.args.tabSyncResult
  },
}))
vi.mock('../tabs/useNoteEditors', () => ({
  createNoteEditors: (args: unknown) => {
    h.args.editing = args
    return h.f.editing
  },
}))
vi.mock('./useSettings', () => ({
  useSettings: (args: unknown) => {
    h.args.settings = args
    return h.f.settings
  },
}))
vi.mock('./useVaults', () => ({ useVaults: () => h.f.vaults }))
vi.mock('../tabs/useOpenTabs', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../tabs/useOpenTabs')>()
  return {
    ...actual,
    useOpenTabs: (args: unknown) => {
      h.args.openTabs = args
      return h.f.openTabs
    },
  }
})
vi.mock('../commands/useCommands', () => ({
  useCommands: (args: unknown) => {
    h.args.commands = args
    return h.f.commands
  },
}))
vi.mock('../commands/fileMenuCommands', () => ({
  useFileMenuCommands: (args: unknown) => {
    h.args.fileMenu = args
    return h.f.fileMenu
  },
}))
vi.mock('./useWindowDisplay', () => ({
  useWindowDisplay: (_core: unknown, args: unknown) => {
    h.args.display = args
    return h.f.display
  },
}))
vi.mock('../shell/useNoticeTasks', () => ({
  useNoticeTasks: (args: unknown) => {
    h.args.noticeTasks = args
    return { value: [{ id: 'notice:1' }] }
  },
}))
vi.mock('./useWindowKinds', () => ({
  useWindowKinds: (args: unknown) => {
    h.args.kinds = args
    return h.f.kinds
  },
}))
vi.mock('../shell/useWindowStatusBar', () => ({
  useWindowStatusBar: (args: unknown) => {
    h.args.statusBar = args
    return { name: 'statusBar' }
  },
}))
vi.mock('../commands/useAppHotkeys', () => ({ useAppHotkeys: vi.fn() }))
vi.mock('../tabs/tabLinkSession', () => ({
  createTabLinkSession: (args: unknown) => {
    h.args.tabLinkSession = args
    return h.f.tabLinkSession
  },
  getWindowReport: () => h.f.windowReport.value,
}))
vi.mock('../tabs/tabMove', () => ({
  createTabMover: (args: unknown) => {
    h.args.tabMover = args
    return h.f.tabMover
  },
  createMovedTabOpener: (args: unknown) => {
    h.args.movedTabOpener = args
    return h.f.openMovedTab
  },
}))
vi.mock('./windowRuntime', () => ({
  focusWindow: vi.fn(async () => undefined),
  closeWindow: vi.fn(async () => undefined),
  closeWindowWhenEmpty: (args: unknown) => {
    h.args.emptyWindowClose = args
  },
}))
vi.mock('../plugins/useWindowPlugins', () => ({
  createPluginSupport: (args: unknown) => {
    h.args.plugins = args
    return { pluginHost: h.f.pluginHost, pluginLoader: h.f.pluginLoader }
  },
  getFrontendPluginIds: (plugins: { id: string }[]) => plugins.map((p) => p.id),
  watchPluginSettings: (settings: unknown, loader: unknown, onError: (err: unknown) => void) => {
    h.args.watchPlugins = { settings, loader, onError }
  },
}))
vi.mock('./useAppBootstrap', () => ({
  useAppBootstrap: (args: unknown) => {
    h.args.bootstrap = args
    return { failure: { value: '' }, isStarted: { value: true }, retry: vi.fn() }
  },
}))

interface FrontTab {
  id: string
  kind: unknown
  state?: unknown
}

/** The front tab, as the window's tabs provide it. */
const front = ref<FrontTab | null>(null)

const createKind = (on: unknown) => ({ name: 'kind', getTarget: vi.fn(() => on) })

beforeEach(() => {
  front.value = null
  h.args = {}
  h.core.fileKinds.mockReset()
  const writer = vi.fn()
  h.f = {
    writer,
    log: { getWriter: vi.fn(() => writer) },
    tabLinks: { name: 'tabLinks' },
    tabLinkSession: {
      start: vi.fn(async () => undefined),
      close: vi.fn(),
      reportTabs: vi.fn(),
      reopen: vi.fn(async () => undefined),
      acceptMove: vi.fn(),
      waitForReports: vi.fn(async () => undefined),
    },
    windowReport: ref<unknown>({ tabs: [], window: {} }),
    tabMover: {
      detachTab: vi.fn(async () => undefined),
      detachTabAt: vi.fn(async () => undefined),
      moveTab: vi.fn(async () => undefined),
    },
    openMovedTab: vi.fn(async () => undefined),
    tabOpeners: { openFileAt: vi.fn(() => Promise.resolve()) },
    windowTabs: {
      layout: ref(null),
      tabs: ref<{ id: string; title: string }[]>([]),
      handle: { front: () => front.value },
      getTab: vi.fn((id: string) =>
        front.value?.id === id ? { kind: front.value.kind } : undefined,
      ),
      releaseTab: vi.fn(() => true),
      finishClose: vi.fn(),
      reconcileTab: vi.fn(),
      openTabOfKind: vi.fn(),
      openSavedTab: vi.fn(),
      restoreTabs: vi.fn(),
      openPlacedTabs: vi.fn(async () => undefined),
      close: vi.fn(),
    },
    editing: {
      notes: { applyPathChanges: vi.fn() },
      changes: { reportChange: vi.fn() },
      noted: { measureAll: vi.fn() },
      reached: { getTabAt: vi.fn(() => null) },
      getTitle: vi.fn(() => 'a title'),
      making: {},
      fileFlush: { start: vi.fn(() => Promise.resolve()) },
      close: vi.fn(),
    },
    display: {
      chunks: {},
      embedded: {},
      isEmbedding: {},
      tasks: ref([{ id: 'task:1' }]),
      failure: ref(null),
      opening: ref('opening'),
      start: vi.fn(),
      close: vi.fn(),
    },
    settings: {
      kept: {},
      nesting: {},
      rest: {
        plugins: ref([{ id: 'p1', name: 'Plugin One' }]),
        writeSetting: vi.fn(async () => undefined),
      },
      configured: { openSettings: vi.fn() },
      start: vi.fn(),
      close: vi.fn(),
    },
    vaults: {
      reload: vi.fn(),
      activeVault: ref({ id: 'vault' }),
      shown: ref({ id: 'vault' }),
      coverage: {},
      setVaultName: vi.fn(),
      loadVaults: vi.fn(),
      vaultList: ref({ vaults: [], showing: '' }),
      listed: {},
    },
    openTabs: { getTabIcon: vi.fn(), pause: vi.fn(async () => undefined), resume: vi.fn() },
    fileMenu: {
      getForFile: vi.fn(() => ['file-item']),
      getIcon: vi.fn((id: string) => (id === 'file:cmd' ? 'file-icon' : undefined)),
    },
    commands: {
      commands: { applyRenames: vi.fn() },
      runCommand: vi.fn(),
      commandDeps: { name: 'commandDeps' },
      palette: {},
      onKeyDown: vi.fn(),
      getNoteCommands: vi.fn(() => [
        { id: 'org.example.sample:share.run', text: 'Share', group: 'note' },
      ]),
      pluginWays: ref([]),
    },
    pluginHost: {
      notifyVaultChange: vi.fn(),
      getPaletteCommands: vi.fn(() => [
        { id: 'org.example.sample:share.run', title: 'Share', icon: 'share' },
      ]),
      getRegisteredSettingsViews: vi.fn(() => ['view-1']),
      getContextMenuItems: vi.fn(() => ['ctx-item']),
      runContextMenu: vi.fn(),
      getContextMenuIcon: vi.fn(() => 'ctx-icon'),
      statusViews: ref([]),
      getSlotView: vi.fn(() => ref(null)),
    },
    pluginLoader: {
      trackPlugins: vi.fn(async () => undefined),
      start: vi.fn(async () => undefined),
      waitForInterface: vi.fn(async () => undefined),
      close: vi.fn(async () => undefined),
    },
    kinds: {
      travel: vi.fn(),
      leavePath: vi.fn(),
      dragged: ref([]),
      services: { workspace: { name: 'workspace' }, notes: { name: 'notes' } },
      made: {},
      destinations: {},
    },
  }
})

describe('what the window answers about the tab in front', () => {
  it('is nothing where no tab is in front', () => {
    const window = useWindow()

    expect(window.getTarget()).toEqual({
      tab: '',
      kind: null,
      path: '',
      title: '',
      file: '',
      vault: { id: 'vault' },
      isReady: true,
    })
  })

  it('is what the kind of the tab says it remains over', () => {
    const kind = createKind({ file: 'a.md', path: 'a', title: 'A', source: 'src' })
    front.value = { id: 't1', kind, state: { target: 1 } }
    const window = useWindow()

    expect(window.getTarget()).toEqual({
      tab: 't1',
      kind,
      path: 'a',
      title: 'A',
      file: 'a.md',
      vault: { id: 'vault' },
      isReady: true,
    })
    expect(kind.getTarget).toHaveBeenCalledWith({ target: 1 })
  })

  it('remains over nothing where the tab in front is not retained', () => {
    front.value = { id: 'gone', kind: createKind({ file: 'a.md' }), state: {} }
    h.f.windowTabs.getTab.mockReturnValue(undefined)
    const window = useWindow()

    expect(window.getTarget()).toMatchObject({ tab: 'gone', path: '', title: '', file: '' })
  })

  it('is not ready while the window has failed', () => {
    h.f.display.failure.value = 'failed'
    const window = useWindow()

    expect(window.getTarget().isReady).toBe(false)
  })
})

describe('the parts of the window', () => {
  it('write their messages under the name command', () => {
    useWindow()

    expect(h.f.log.getWriter).toHaveBeenCalledWith('command')
    expect(h.args.commands.writeMessage).toBe(h.f.writer)
  })

  it('are told the paths that changed, in every editor and in the commands', async () => {
    useWindow()
    const paths = ['a.md']
    const renamedItems = new Map([['a.md', 'b.md']])

    await h.args.display.onVaultChanged(paths, renamedItems)

    expect(h.f.editing.notes.applyPathChanges).toHaveBeenCalledWith(paths, renamedItems)
    expect(h.f.commands.commands.applyRenames).toHaveBeenCalledWith(renamedItems)
    expect(h.f.pluginHost.notifyVaultChange).toHaveBeenCalledWith({ paths, renamed: renamedItems })
  })

  it('are requested to travel to a path through the kinds', () => {
    useWindow()

    h.args.display.travelTo('a.md')

    expect(h.f.kinds.travel).toHaveBeenCalledWith('a.md')
  })

  it('are requested to open a file at the spans through the openers', () => {
    useWindow()
    const spans = [{ start: 1, end: 2 }]

    h.args.display.openFileAt('a.md', spans)

    expect(h.f.tabOpeners.openFileAt).toHaveBeenCalledWith('a.md', spans)
  })

  it('are requested to reload through the vaults', () => {
    useWindow()

    h.args.display.reload()

    expect(h.f.vaults.reload).toHaveBeenCalledOnce()
  })

  it('hand the kinds the commands dependencies as they stand', () => {
    useWindow()

    expect(h.args.kinds.commandDeps()).toBe(h.f.commands.commandDeps)
  })

  it('hand the kinds what the commands declare of the plugins', async () => {
    useWindow()

    expect(h.args.kinds.getNoteCommands()).toStrictEqual([
      { id: 'org.example.sample:share.run', text: 'Share', group: 'note' },
    ])
    expect(h.args.kinds.getPluginCommandIcon('org.example.sample:share.run')).toBe(
      getNamedIcon('share'),
    )
    expect(h.args.kinds.getPluginCommandIcon('org.example.sample:none.run')).toBeUndefined()
  })

  it('hand the commands the kinds as they stand', () => {
    useWindow()

    h.args.commands.travel('a.md')
    h.args.commands.leavePath('a.md', 'b.md')
    expect(h.f.kinds.travel).toHaveBeenCalledWith('a.md')
    expect(h.f.kinds.leavePath).toHaveBeenCalledWith('a.md', 'b.md')
    expect(h.args.commands.getOpeningNote()).toBe('opening')
  })
})

describe('what the window knows of a note', () => {
  it('is the tab it is open in', () => {
    h.f.editing.reached.getTabAt.mockReturnValue('t9')
    useWindow()

    expect(h.args.commands.knows.getTabAt('a.md')).toBe('t9')
    expect(h.f.editing.reached.getTabAt).toHaveBeenCalledWith('a.md')
  })

  it('is its title from the tab it is open in', () => {
    h.f.editing.reached.getTabAt.mockReturnValue('t9')
    useWindow()

    expect(h.args.commands.knows.getTitle('a.md')).toBe('a title')
    expect(h.f.editing.getTitle).toHaveBeenCalledWith('t9')
  })

  it('is no title where it is open in no tab', () => {
    useWindow()

    expect(h.args.commands.knows.getTitle('a.md')).toBe('')
  })

  describe('in a tab of a kind that edits files', () => {
    const stand = () => {
      h.f.windowTabs.tabs.value = [{ id: 'sample:a', title: 'Animals' }]
      h.f.windowTabs.getTab.mockReturnValue({
        state: { path: 'a.md' },
        kind: { flushPending: () => Promise.resolve(), getOpenTab: () => ({ path: 'a.md' }) },
      })
    }

    it('is the tab it is open in', () => {
      stand()
      useWindow()

      expect(h.args.commands.knows.getTabAt('a.md')).toBe('sample:a')
    })

    it('is its title from that tab', () => {
      stand()
      h.f.editing.getTitle.mockReturnValue('')
      useWindow()

      expect(h.args.commands.knows.getTitle('a.md')).toBe('Animals')
    })
  })
})

describe('a tab closed', () => {
  it('is released and removed from the layout when no hold is passed', () => {
    const window = useWindow()

    expect(window.closeTab('t1')).toBe(true)

    expect(h.f.windowTabs.releaseTab).toHaveBeenCalledWith('t1')
    expect(h.f.windowTabs.finishClose).toHaveBeenCalledWith('t1')
  })

  it('returns false where the tab was not released', () => {
    h.f.windowTabs.releaseTab.mockReturnValue(false)
    const window = useWindow()

    expect(window.closeTab('t1')).toBe(false)
  })

  it('calls the hold where the tab was not released', () => {
    h.f.windowTabs.releaseTab.mockReturnValue(false)
    const window = useWindow()
    const hold = vi.fn()

    window.closeTab('t1', hold)

    expect(hold).toHaveBeenCalledOnce()
    expect(h.f.windowTabs.finishClose).not.toHaveBeenCalled()
  })

  it('does not call the hold or finishClose where a hold is passed and the tab was released', () => {
    const window = useWindow()
    const hold = vi.fn()

    window.closeTab('t1', hold)

    expect(hold).not.toHaveBeenCalled()
    expect(h.f.windowTabs.finishClose).not.toHaveBeenCalled()
  })

  it('needs no hold where the tab was not released', () => {
    h.f.windowTabs.releaseTab.mockReturnValue(false)
    const window = useWindow()

    expect(() => window.closeTab('t1')).not.toThrow()
    expect(h.f.windowTabs.finishClose).not.toHaveBeenCalled()
  })
})

describe('tab detach and move wiring', () => {
  it('calls the mover with the window id', () => {
    const window = useWindow()

    window.detachTab('t1', { x: 1, y: 2 })
    window.detachTabAt('t1', { x: 3, y: 4 })
    window.moveTab('t1', 'window-b')

    expect(h.f.tabMover.detachTab).toHaveBeenCalledWith('t1', { x: 1, y: 2 })
    expect(h.f.tabMover.detachTabAt).toHaveBeenCalledWith('t1', { x: 3, y: 4 })
    expect(h.f.tabMover.moveTab).toHaveBeenCalledWith('t1', 'window-b')
    expect(h.args.tabMover.windowId).toBe(window.windowId)
    expect(h.args.tabMover.core).toBe(h.core)
  })

  it('closes the source tab through the tab closer', () => {
    useWindow()

    h.args.tabMover.closeTab('t1')

    expect(h.f.windowTabs.releaseTab).toHaveBeenCalledWith('t1')
    expect(h.f.windowTabs.finishClose).toHaveBeenCalledWith('t1')
  })

  it('reports a tab as asking when its kind is asking', () => {
    h.f.editing.reached.asking = vi.fn(() => false)
    h.f.windowTabs.getTab.mockReturnValue({ kind: { isAsking: () => true }, state: {} })
    const window = useWindow()

    expect(window.isTabAsking('t1')).toBe(true)
  })
})

describe('moved tab wiring', () => {
  it('opens a moved tab through the opener, accepts it through the session, and sets the highlight', async () => {
    const window = useWindow()
    const moved = {
      tab: { id: 'plex:2', kind: 'plex', path: 'Tides.md', title: 'Tides', state: '' },
      moveId: 'move-1',
    }

    h.args.tabLinkSession.openMovedTab(moved)
    h.args.tabLinkSession.setHighlighted(true)

    await vi.waitFor(() => expect(h.f.openMovedTab).toHaveBeenCalledWith(moved))
    expect(h.args.movedTabOpener.acceptMove).toBe(h.f.tabLinkSession.acceptMove)
    expect(window.isHighlighted.value).toBe(true)
  })

  it('sends HighlightWindow through the core', async () => {
    const window = useWindow()

    window.highlightWindow('window-b', true)

    await vi.waitFor(() => expect(h.core.highlightWindow).toHaveBeenCalledWith('window-b', true))
  })

  it('sends a highlight request after the previous one completes', async () => {
    let finishOn = (): void => undefined
    h.core.highlightWindow.mockImplementationOnce(
      () => new Promise<undefined>((resolve) => (finishOn = () => resolve(undefined))),
    )
    const window = useWindow()

    window.highlightWindow('window-b', true)
    window.highlightWindow('window-b', false)
    await vi.waitFor(() => expect(h.core.highlightWindow).toHaveBeenCalledOnce())
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(h.core.highlightWindow).toHaveBeenCalledOnce()
    finishOn()

    await vi.waitFor(() => expect(h.core.highlightWindow).toHaveBeenCalledTimes(2))
    expect(h.core.highlightWindow.mock.calls).toStrictEqual([
      ['window-b', true],
      ['window-b', false],
    ])
  })
})

describe('the window started and closed', () => {
  it('starts the file flush, and returns nothing of it', () => {
    useWindow()

    expect(h.args.bootstrap.startEditing()).toBeUndefined()
    expect(h.f.editing.fileFlush.start).toHaveBeenCalledOnce()
  })

  it('closes the window, the editors, the tabs and the settings', () => {
    useWindow()

    h.args.bootstrap.closeWindow()
    h.args.bootstrap.closeEditing()
    h.args.bootstrap.closeTabs()
    h.args.bootstrap.closeSettings()

    expect(h.f.display.close).toHaveBeenCalledOnce()
    expect(h.f.editing.close).toHaveBeenCalledOnce()
    expect(h.f.windowTabs.close).toHaveBeenCalledOnce()
    expect(h.f.settings.close).toHaveBeenCalledOnce()
  })
})

describe('the services a plugin is given', () => {
  it('are the ones the kinds made, the window task list, the commands, the vaults and the window', () => {
    useWindow()

    const { taskList, commands, vaults, window, ...made } = h.args.plugins.services
    expect(made).toStrictEqual(h.f.kinds.services)
    expect(taskList).toBe(h.f.display.tasks)
    expect(vaults?.getState()).toBeDefined()
    expect(commands?.getIds()).toStrictEqual(['app:vault.create', 'org.example.sample:share.run'])
    expect(window?.getState()).toStrictEqual({ isMaximized: false, isFullscreen: false })
  })

  it('stops the tabs waiting for their plugins once the plugins of the interface have started', async () => {
    useWindow()
    h.f.pluginLoader.start.mockImplementationOnce(() => new Promise<undefined>(() => undefined))

    const started = h.args.bootstrap.startPlugins()
    expect(started).toBeInstanceOf(Promise)
    await new Promise((r) => setTimeout(r, 0))

    expect(h.args.windowTabs.arePluginsStarting.value).toBe(false)
  })

  it('wires plugin support, windowTabs, settings, kinds, commands, statusBar, and lifecycle callbacks', async () => {
    const window = useWindow()

    // useWindowTabs options
    expect(h.args.windowTabs.arePluginsStarting.value).toBe(true)
    await h.args.windowTabs.enablePlugin('p1')
    expect(h.f.settings.rest.writeSetting).toHaveBeenCalledWith(['p1', 'enabled'], true)

    const dispatched: CustomEvent[] = []
    const listener = (e: Event) => dispatched.push(e as CustomEvent)
    globalThis.addEventListener('numen:open-settings', listener)
    try {
      h.args.windowTabs.openPlugin('p1')
      expect(dispatched).toHaveLength(1)
      expect(dispatched[0]?.detail).toEqual({ pluginId: 'p1' })
    } finally {
      globalThis.removeEventListener('numen:open-settings', listener)
    }

    // createPluginSupport callbacks
    h.args.plugins.reportError('p1', new Error('plugin err'))
    expect(h.f.log.getWriter).toHaveBeenCalledWith('plugin:p1')
    expect(h.f.writer).toHaveBeenCalledWith(expect.stringContaining('Plugin One'), 'error')

    h.args.plugins.reportLoadError('unknown-plugin', new Error('load err'))
    expect(h.f.writer).toHaveBeenCalledWith(expect.stringContaining('unknown-plugin'), 'error')

    h.args.plugins.reportCommandIssue('p1', 'the command tree.open has no handler')
    expect(h.f.writer).toHaveBeenCalledWith(
      expect.stringContaining('the command tree.open has no handler'),
      'warning',
    )
    expect(h.args.plugins.getCommandDeclarations()).toBeUndefined()

    expect(await h.args.plugins.resolveLink('from.md', 'name://Target')).toBe('resolved.md')
    expect(h.core.resolveLinks).toHaveBeenCalledWith('from.md', '[[Target]]')
    h.core.resolveLinks.mockResolvedValueOnce([])
    expect(await h.args.plugins.resolveLink('from.md', 'Plain')).toBe('')

    // createTabSync & watchPluginSettings & useOpenTabs error callbacks
    h.f.writer.mockClear()
    h.args.tabSync.onError(new Error('tab sync err'))
    expect(h.f.writer).toHaveBeenCalledWith('Numen is not responding; retrying', 'error')

    h.args.watchPlugins.onError(new Error('watch plugins err'))
    expect(h.f.writer).toHaveBeenCalledWith(
      'Plugins were not tracked: Numen is not responding; retrying',
      'error',
    )

    expect(h.args.openTabs.isWriting.value).toBe(false)
    expect(window.isLayoutReady.value).toBe(false)
    h.args.openTabs.reportError(new Error('save tabs err'))
    expect(h.f.writer).toHaveBeenCalledWith(
      'The open tabs were not saved: Numen is not responding; retrying',
      'error',
    )

    // useSettings callbacks
    h.args.settings.onSizeChanged()
    expect(h.f.editing.noted.measureAll).toHaveBeenCalledOnce()
    expect(h.args.settings.getSettingsViews('p1')).toEqual(['view-1'])

    // useFileMenuCommands callbacks
    h.args.fileMenu.onRun('notes/a.md')
    expect(h.f.pluginHost.notifyVaultChange).toHaveBeenCalledWith({ paths: ['notes/a.md'] })
    h.f.writer.mockClear()
    h.args.fileMenu.onError(new Error('file menu err'))
    expect(h.f.writer).toHaveBeenCalledWith('Numen is not responding; retrying', 'error')

    // useWindowKinds callbacks
    const target = window.getTarget()
    h.args.kinds.runCommand('cmd-1', target)
    expect(h.f.commands.runCommand).toHaveBeenCalledWith('cmd-1', target)
    expect(h.args.kinds.getContextMenuItems(target)).toEqual(['ctx-item'])
    h.args.kinds.runContextMenu('ctx-1', target)
    expect(h.f.pluginHost.runContextMenu).toHaveBeenCalledWith('ctx-1', target)
    expect(h.args.kinds.getFileMenuItems('a.md')).toEqual(['file-item'])
    expect(h.args.kinds.getContextMenuIcon('ctx-1')).toBe('ctx-icon')
    expect(h.args.kinds.getPluginCommandIcon('file:cmd')).toBe('file-icon')

    // tabsHold callbacks
    await h.args.commands.tabsHold.pause()
    expect(h.f.openTabs.pause).toHaveBeenCalledOnce()
    h.args.commands.tabsHold.resume()
    expect(h.f.openTabs.resume).toHaveBeenCalledOnce()

    // statusBar & noticeTasks
    expect(h.args.noticeTasks.window).toBe(h.f.display)
    expect(h.args.statusBar.tasks.value).toEqual([{ id: 'task:1' }, { id: 'notice:1' }])
    await h.args.statusBar.cancelTask('task:1')
    expect(h.core.cancelTask).toHaveBeenCalledWith('task:1')
    h.args.statusBar.reportCancelError(new Error('cancel failed'))
    expect(h.f.writer).toHaveBeenCalledWith(
      'The task was not cancelled: Numen is not responding; retrying',
      'error',
    )
    h.args.statusBar.onAction('p1')
    expect(h.f.settings.configured.openSettings).toHaveBeenCalledWith('p1')

    // Returned window methods
    window.runCommand('cmd-2', target)
    expect(h.f.commands.runCommand).toHaveBeenCalledWith('cmd-2', target)
    window.openSettings('p1', 'ocr')
    expect(h.f.settings.configured.openSettings).toHaveBeenCalledWith('p1', 'ocr')

    h.f.windowTabs.openPlacedTabs.mockImplementationOnce(
      async (onErr: (kind: string, err: unknown) => void) => {
        onErr('graph', new Error('graph failed'))
      },
    )
    window.openCurrentVault()
    await Promise.resolve()
    expect(h.f.writer).toHaveBeenCalledWith(
      'The graph tab did not open: Numen is not responding; retrying',
      'error',
    )

    h.f.windowTabs.openPlacedTabs.mockRejectedValueOnce(new Error('all tabs failed'))
    window.openCurrentVault()
    await new Promise((r) => setTimeout(r, 10))
    expect(h.f.writer).toHaveBeenCalledWith(
      'The tabs did not open: Numen is not responding; retrying',
      'error',
    )

    // Bootstrap lifecycle callbacks
    await h.args.bootstrap.loadVaults()
    expect(h.f.vaults.loadVaults).toHaveBeenCalledOnce()
    await h.args.bootstrap.startSettings()
    expect(h.f.settings.start).toHaveBeenCalledOnce()
    await h.args.bootstrap.startWindow()
    expect(h.f.display.start).toHaveBeenCalledOnce()

    await h.args.bootstrap.startPlugins()
    expect(h.f.pluginLoader.trackPlugins).toHaveBeenCalledWith(['p1'])
    expect(h.f.pluginLoader.start).toHaveBeenCalledOnce()
    expect(h.args.windowTabs.arePluginsStarting.value).toBe(false)

    await h.args.bootstrap.closePlugins()
    expect(h.f.pluginLoader.close).toHaveBeenCalledOnce()

    h.args.bootstrap.reportError(new Error('boot failed'))
    expect(h.f.writer).toHaveBeenCalledWith(
      'The window did not start cleanly: Numen is not responding; retrying',
      'error',
    )

    h.f.editing.fileFlush.start.mockRejectedValueOnce(new Error('flush died'))
    h.args.bootstrap.startEditing()
    await new Promise((r) => setTimeout(r, 10))
    expect(h.f.writer).toHaveBeenCalledWith(
      'Saving on quit stopped: Numen is not responding; retrying',
      'error',
    )
  })

  it('restores saved tabs or opens placed tabs in startLayout and handles secondary windows', async () => {
    try {
      vi.stubGlobal('location', new URL('http://localhost/'))
      const window = useWindow()

      // Empty activeVault id returns early
      h.f.vaults.activeVault.value = { id: '' }
      await h.args.bootstrap.startLayout()
      expect(window.isLayoutReady.value).toBe(false)

      // Saved tabs present -> restores them
      h.f.vaults.activeVault.value = { id: 'vault' }
      h.core.readOpenTabs.mockResolvedValueOnce({
        tabs: [{ id: 'note:a', kind: 'note', title: 'A', path: 'a.md', state: '' }] as never,
        activeId: 'note:a',
      })
      await h.args.bootstrap.startLayout()
      expect(h.f.windowTabs.restoreTabs).toHaveBeenCalledWith(
        [{ id: 'note:a', kind: 'note', title: 'A', path: 'a.md', saved: undefined }],
        'note:a',
      )
      expect(window.isLayoutReady.value).toBe(true)
      expect(h.args.openTabs.isWriting.value).toBe(true)

      // readDetachedTab fails -> the detach window opens no tab and does not restore the vault tabs
      vi.stubGlobal('location', new URL('http://localhost/?secondary&id=files:1'))
      const secondaryWin = useWindow()
      h.core.readOpenTabs.mockClear()

      h.core.readDetachedTab.mockRejectedValueOnce(new Error('offline'))
      await h.args.bootstrap.startLayout()
      expect(h.core.readDetachedTab).toHaveBeenCalledWith(h.args.tabLinkSession.windowId, 'files:1')
      expect(h.f.openMovedTab).not.toHaveBeenCalled()
      expect(h.core.readOpenTabs).not.toHaveBeenCalled()
      expect(h.f.windowTabs.openPlacedTabs).not.toHaveBeenCalled()
      expect(secondaryWin.isLayoutReady.value).toBe(true)
      expect(h.args.openTabs.isWriting.value).toBe(false)
    } finally {
      vi.unstubAllGlobals()
    }
  })
})

describe('tab links wiring', () => {
  it('writes a link after the pending tab reports', async () => {
    let finishReports: () => void = () => undefined
    h.f.tabLinkSession.waitForReports.mockImplementationOnce(
      () => new Promise<undefined>((resolve) => (finishReports = () => resolve(undefined))),
    )
    useWindow()

    const request = h.args.tabLinks.write({ link: { source: 'plex:1', target: 'plex:2' } })
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(h.core.writeTabLinks).not.toHaveBeenCalled()
    finishReports()
    await request

    expect(h.core.writeTabLinks).toHaveBeenCalledExactlyOnceWith(h.args.tabLinkSession.windowId, {
      link: { source: 'plex:1', target: 'plex:2' },
    })
  })

  it('uses one window id for writes and the watch', async () => {
    useWindow()
    const windowId = h.args.tabLinkSession.windowId

    await h.args.tabLinks.write({ unlink: 'plex:1' })

    expect(windowId).toMatch(/^[0-9a-f-]{36}$/)
    expect(h.core.writeTabLinks).toHaveBeenCalledWith(windowId, { unlink: 'plex:1' })
    expect(h.args.tabLinkSession.tabLinks).toBe(h.f.tabLinks)
    expect(h.args.tabSync.tabLinks).toBe(h.f.tabLinks)
  })

  it('sends the new path of a linked tab to the core', async () => {
    useWindow()
    const sync = { sourceTabId: 'plex:1', path: 'Orbits.md', title: 'Orbits' }

    await h.args.tabSync.sendSync(sync)

    expect(h.core.syncTabPath).toHaveBeenCalledWith(sync)
  })

  it('starts with the window and stops when it closes', async () => {
    useWindow()

    await h.args.bootstrap.startWindow()
    expect(h.f.tabLinkSession.start).toHaveBeenCalledOnce()

    h.args.bootstrap.closeWindow()
    expect(h.f.tabLinkSession.close).toHaveBeenCalledOnce()
    expect(h.f.display.close).toHaveBeenCalledOnce()
  })

  it('reports the tabs again when the window report changes', async () => {
    useWindow()

    h.f.windowReport.value = { tabs: [], window: { isMain: true, tabCount: 1 } }
    await vi.waitFor(() => expect(h.f.tabLinkSession.reportTabs).toHaveBeenCalledOnce())
    expect(h.args.tabLinkSession.getReport()).toStrictEqual(h.f.windowReport.value)
  })

  it('reports an error when the core rejects a link', () => {
    useWindow()

    h.args.tabLinks.onError(new Error('refused'))

    expect(h.f.writer).toHaveBeenCalledWith(
      'The tabs were not linked: Numen is not responding; retrying',
      'error',
    )
  })

  it('opens a detached tab with its id and saved state', async () => {
    try {
      vi.stubGlobal('location', new URL('http://localhost/?window=secondary&id=plex:4f1c'))
      const detached = {
        tab: {
          id: 'plex:4f1c',
          kind: 'plex',
          path: 'Orbits.md',
          title: 'Orbits',
          state: '{"focus":"Kepler"}',
        },
        moveId: 'move-1',
      }
      h.core.readDetachedTab.mockResolvedValueOnce(detached)
      const window = useWindow()

      await h.args.bootstrap.startLayout()

      expect(h.core.readDetachedTab).toHaveBeenCalledWith(
        h.args.tabLinkSession.windowId,
        'plex:4f1c',
      )
      await vi.waitFor(() => expect(h.f.openMovedTab).toHaveBeenCalledWith(detached))
      expect(h.core.readOpenTabs).not.toHaveBeenCalled()
      expect(window.isLayoutReady.value).toBe(true)
      expect(h.args.emptyWindowClose.tabCount()).toBe(0)
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('closes the tab links session when the page hides', () => {
    useWindow()

    globalThis.dispatchEvent(new Event('pagehide'))

    expect(h.f.tabLinkSession.close).toHaveBeenCalled()
  })

  it('reopens the tab links session when the page is restored from the back-forward cache', () => {
    useWindow()
    h.f.tabLinkSession.reopen.mockClear()

    globalThis.dispatchEvent(Object.assign(new Event('pageshow'), { persisted: false }))
    expect(h.f.tabLinkSession.reopen).not.toHaveBeenCalled()

    globalThis.dispatchEvent(Object.assign(new Event('pageshow'), { persisted: true }))
    expect(h.f.tabLinkSession.reopen).toHaveBeenCalled()
  })

  it('passes the group paths of each snapshot to the tab sync', () => {
    useWindow()

    expect(h.args.tabLinkSession.applyGroupPaths).toBe(h.args.tabSyncResult.applyGroupPaths)
  })

  it('does not close the main window when its last tab closes', () => {
    useWindow()

    expect(h.args.emptyWindowClose).toBeUndefined()
  })
})
