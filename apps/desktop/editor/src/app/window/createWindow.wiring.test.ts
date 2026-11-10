/**
 * The window put together: what each part is handed of the others, and what
 * the window answers about the tab in front.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createLostConnection } from '@numen/core-client/testing'
import { Puzzle } from '@lucide/vue'
import { getNamedIcon } from '@/shared/ui/icons'
import { COMMAND, commands as commandRegistry } from '@/entities/command'
import { createInvocation } from '@/features/command-palette'
import { answerCore } from '@/testing/coreAnswers'
import { ref } from 'vue'
import { createWindow } from './createWindow'
import type * as BackgroundTasksModule from '@/features/background-tasks'
import type * as WindowModule from '@/entities/window'
import type * as TabModule from '@/entities/tab'
import type * as TabLinkingModule from '@/features/tab-linking'
import type * as TabMoveModule from '@/features/tab-move'
import type * as HotkeysModule from '@/features/hotkeys'

/** What each part was made with, and what it was made as. */
const h = vi.hoisted(() => ({
  args: {} as Record<string, any>,
  f: {} as Record<string, any>,
  core: {
    name: 'core',
    readOpenTabs: vi.fn(async () => ({ tabs: [], activeId: '' })),
    cancelTask: vi.fn(async (_id: string) => undefined),
    writeTabLinks: vi.fn(async (_windowId: string, _change: unknown) => undefined),
    syncTabPath: vi.fn(async (_sync: unknown) => undefined),
    readDetachedTab: vi.fn(async (_windowId: string, _tabId: string): Promise<unknown> => null),
    highlightWindow: vi.fn(async (_windowId: string, _isOn: boolean) => undefined),
  },
}))

vi.mock('@/features/tab-linking', async (importOriginal) => ({
  ...(await importOriginal<typeof TabLinkingModule>()),
  useTabLinks: (args: unknown) => {
    h.args.tabLinks = args
    return h.f.tabLinks
  },
  createTabReports: (args: unknown) => {
    h.args.tabReports = args
    return h.f.tabReports
  },
  createTabLinkSession: (args: unknown) => {
    h.args.tabLinkSession = args
    return h.f.tabLinkSession
  },
  getWindowReport: () => h.f.windowReport.value,
  writeTabLinks: (windowId: string, change: unknown) => h.core.writeTabLinks(windowId, change),
  syncTabPath: (sync: unknown) => h.core.syncTabPath(sync),
  highlightWindow: (windowId: string, isOn: boolean) => h.core.highlightWindow(windowId, isOn),
}))
vi.mock('@/features/tab-move', async (importOriginal) => ({
  ...(await importOriginal<typeof TabMoveModule>()),
  readDetachedTab: (windowId: string, tabId: string) => h.core.readDetachedTab(windowId, tabId),
  createTabMover: (args: unknown) => {
    h.args.tabMover = args
    return h.f.tabMover
  },
  createMovedTabOpener: (args: unknown) => {
    h.args.movedTabOpener = args
    return h.f.openMovedTab
  },
}))
vi.mock('@/shared/model/notices/messages', () => ({ createMessageLog: () => h.f.log }))
vi.mock('@/entities/tab', async (importOriginal) => ({
  ...(await importOriginal<typeof TabModule>()),
  createWorkspace: vi.fn(),
  fileOpeners: (args: unknown) => {
    h.args.openers = args
    return h.f.tabOpeners
  },
  createWindowTabs: (_panes: unknown, services: unknown) => {
    h.args.windowTabs = services
    return h.f.windowTabs
  },
  createTabSync: (args: unknown) => {
    h.args.tabSync = args
    h.args.tabSyncResult = { name: 'tabSync', applySync: vi.fn(), applyGroupPaths: vi.fn() }
    return h.args.tabSyncResult
  },
  createTabNavigation: () => h.f.navigation,
  useOpenTabs: (args: unknown) => {
    h.args.openTabs = args
    return h.f.openTabs
  },
  readOpenTabs: () => h.core.readOpenTabs(),
}))
vi.mock('../tabs/createNoteEditors', () => ({
  createNoteEditors: (args: unknown) => {
    h.args.editing = args
    return h.f.editing
  },
}))
vi.mock('@/entities/settings', async (importOriginal) => ({
  ...(await importOriginal<object>()),
  createSettingsStore: () => h.f.settings.rest,
}))
vi.mock('@/pages/settings/model/useSettings', () => ({
  useSettings: (args: unknown) => {
    h.args.settings = args
    return h.f.settings
  },
}))
vi.mock('@/entities/vault/model/useVaults', () => ({ useVaults: () => h.f.vaults }))
vi.mock('@/features/window-display', () => ({
  useWindowDisplay: (args: unknown) => {
    h.args.display = args
    return h.f.display
  },
}))
vi.mock('@/features/background-tasks', async (original) => ({
  ...(await original<typeof BackgroundTasksModule>()),
  useNoticeTasks: (args: unknown) => {
    h.args.noticeTasks = args
    return { value: [{ id: 'notice:1' }] }
  },
  createBackgroundTaskHost: (args: unknown) => {
    h.args.backgroundTasks = args
    return { name: 'backgroundTasks' }
  },
}))
vi.mock('@/features/hotkeys', async (original) => ({
  ...(await original<typeof HotkeysModule>()),
  useAppHotkeys: vi.fn(),
}))
vi.mock('@/entities/window', async (original) => ({
  ...(await original<typeof WindowModule>()),
  cancelTask: (id: string) => h.core.cancelTask(id),
  focusWindow: vi.fn(async () => undefined),
  closeWindow: vi.fn(async () => undefined),
  closeWindowWhenEmpty: (args: unknown) => {
    h.args.emptyWindowClose = args
  },
}))
vi.mock('../plugins/pluginSupport', () => ({
  createWindowPluginStatus: () => h.f.status,
  createPluginSupport: (args: unknown) => {
    h.args.plugins = args
    return { pluginHost: h.f.pluginHost, pluginLoader: h.f.pluginLoader }
  },
  getFrontendPluginIds: (plugins: { id: string }[]) => plugins.map((p) => p.id),
  watchPluginSettings: (settings: unknown, loader: unknown, onError: (err: unknown) => void) => {
    h.args.watchPlugins = { settings, loader, onError }
  },
}))
vi.mock('../bootstrap/createAppBootstrap', () => ({
  createAppBootstrap: (args: unknown) => {
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

const createKind = (on: unknown) => ({ name: 'kind', address: { getTarget: vi.fn(() => on) } })

beforeEach(() => {
  front.value = null
  h.args = {}
  const writer = vi.fn()
  h.f = {
    writer,
    log: { getWriter: vi.fn(() => writer) },
    tabLinks: { name: 'tabLinks' },
    status: { isPluginStarting: vi.fn((_pluginId: string) => false) },
    tabReports: {
      reportTabs: vi.fn(),
      waitForReports: vi.fn(async () => undefined),
      stop: vi.fn(),
      resume: vi.fn(),
    },
    tabLinkSession: {
      start: vi.fn(async () => undefined),
      close: vi.fn(),
      reportTabs: vi.fn(),
      reopen: vi.fn(async () => undefined),
      acceptMove: vi.fn(),
    },
    windowReport: ref<unknown>({ tabs: [], window: {} }),
    tabMover: {
      detachTab: vi.fn(async () => undefined),
      detachTabInPlace: vi.fn(async () => undefined),
      moveTab: vi.fn(async () => undefined),
    },
    openMovedTab: vi.fn(async () => undefined),
    tabOpeners: { openFileAt: vi.fn(() => Promise.resolve()) },
    windowTabs: {
      panes: {
        layout: ref(null),
        releaseTab: vi.fn(() => true),
        finishClose: vi.fn(),
        reconcileTab: vi.fn(),
      },
      tabs: {
        tabs: ref<{ id: string; title: string }[]>([]),
        getTab: vi.fn((id: string) =>
          front.value?.id === id ? { kind: front.value.kind } : undefined,
        ),
        openTabOfKind: vi.fn(),
      },
      groups: { handle: { front: () => front.value }, registerKinds: vi.fn() },
      restored: {
        openSavedTab: vi.fn(),
        restoreTabs: vi.fn(),
        openPlacedTabs: vi.fn(async () => undefined),
        close: vi.fn(),
      },
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
      vault: {
        chunks: {},
        embedded: {},
        isEmbedding: {},
        unwatched: ref(''),
        error: ref(''),
      },
      health: { failure: ref(null), lost: ref('') },
      tasks: ref([{ id: 'task:1' }]),
      opening: { path: ref('opening') },
      start: vi.fn(),
      close: vi.fn(),
    },
    settings: {
      kept: {},
      nesting: {},
      kinds: [],
      paletteLists: { getStepGroups: vi.fn(() => []), previewItem: vi.fn() },
      settingPalette: { getCommands: () => [], choose: vi.fn(async () => undefined) },
      rest: {
        state: { plugins: ref([{ id: 'p1', name: 'Plugin One' }]) },
        read: { getCommittedSetting: vi.fn(() => undefined) },
        write: { writeSetting: vi.fn(async () => undefined) },
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
    pluginHost: {
      notifyVaultChange: vi.fn(),
      keybindings: {
        getKeyFor: vi.fn(() => undefined),
        registerKeybindings: vi.fn(),
        unregisterKeybindings: vi.fn(),
      },
      commands: {
        hasCommand: vi.fn(() => false),
        handlers: { isSharedCommand: vi.fn(() => false) },
        executeCommand: vi.fn(async () => undefined),
        handleShortcut: vi.fn(() => false),
        hasCommandItems: vi.fn(() => false),
        listCommandItems: vi.fn(async () => []),
        getPaletteCommands: vi.fn(() => [
          { id: 'org.example.sample:share.run', title: 'Share', icon: 'share' },
        ]),
      },
      registered: { getRegisteredSettingsViews: vi.fn(() => ['view-1']) },
      slotViews: { getSlotView: vi.fn(() => ref(null)) },
    },
    pluginLoader: {
      trackPlugins: vi.fn(async () => undefined),
      start: vi.fn(async () => undefined),
      status: {
        waitForInterface: vi.fn(async () => undefined),
        waitForTabPlugins: vi.fn(async () => undefined),
      },
      close: vi.fn(async () => undefined),
    },
    navigation: { travel: vi.fn(), leavePath: vi.fn(async () => undefined) },
  }
  answerCore({})
})

describe('what the window answers about the tab in front', () => {
  it('is nothing where no tab is in front', () => {
    const window = createWindow()

    expect(window.palette.getTarget()).toEqual({
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
    const window = createWindow()

    expect(window.palette.getTarget()).toEqual({
      tab: 't1',
      kind,
      path: 'a',
      title: 'A',
      file: 'a.md',
      vault: { id: 'vault' },
      isReady: true,
    })
    expect(kind.address.getTarget).toHaveBeenCalledWith({ target: 1 })
  })

  it('remains over nothing where the tab in front is not retained', () => {
    front.value = { id: 'gone', kind: createKind({ file: 'a.md' }), state: {} }
    h.f.windowTabs.tabs.getTab.mockReturnValue(undefined)
    const window = createWindow()

    expect(window.palette.getTarget()).toMatchObject({ tab: 'gone', path: '', title: '', file: '' })
  })

  it('is not ready while the window has failed', () => {
    h.f.display.health.failure.value = 'failed'
    const window = createWindow()

    expect(window.palette.getTarget().isReady).toBe(false)
  })
})

describe('the parts of the window', () => {
  it('write their messages under the name command', () => {
    createWindow()

    expect(h.f.log.getWriter).toHaveBeenCalledWith('command')
  })

  it('receive the paths that changed, in every editor and in the plugins', async () => {
    createWindow()
    const paths = ['a.md']
    const renamedItems = new Map([['a.md', 'b.md']])

    await h.args.display.onVaultChanged(paths, renamedItems)

    expect(h.f.editing.notes.applyPathChanges).toHaveBeenCalledWith(paths, renamedItems)
    expect(h.f.pluginHost.notifyVaultChange).toHaveBeenCalledWith({ paths, renamed: renamedItems })
  })

  it('are requested to travel to a path through the tab navigation', () => {
    createWindow()

    h.args.display.travelTo('a.md')

    expect(h.f.navigation.travel).toHaveBeenCalledWith('a.md')
  })

  it('are requested to open a file at the spans through the openers', () => {
    createWindow()
    const spans = [{ start: 1, end: 2 }]

    h.args.display.openFileAt('a.md', spans)

    expect(h.f.tabOpeners.openFileAt).toHaveBeenCalledWith('a.md', spans)
  })

  it('are requested to reload the page', () => {
    const reload = vi.fn()
    vi.stubGlobal('location', { ...globalThis.location, reload })
    createWindow()

    h.args.display.reload()

    expect(reload).toHaveBeenCalledOnce()
    vi.unstubAllGlobals()
  })

  it('detach a tab through the mover when the command of the tab runs', async () => {
    const window = createWindow()
    const target = { ...window.palette.getTarget(), tab: 't1' }

    const run = commandRegistry.get(COMMAND.detachTab)
    await run?.(createInvocation(COMMAND.detachTab, target), window.palette.commandDeps)

    expect(h.f.tabMover.detachTabInPlace).toHaveBeenCalledWith('t1')
  })

  it('move a tab to a window through the mover when the command of the tab runs', async () => {
    const window = createWindow()
    const target = { ...window.palette.getTarget(), tab: 't1' }

    const run = commandRegistry.get(COMMAND.moveTabToWindow)
    await run?.(
      createInvocation(COMMAND.moveTabToWindow, target, 'window-b'),
      window.palette.commandDeps,
    )

    expect(h.f.tabMover.moveTab).toHaveBeenCalledWith('t1', 'window-b')
  })

  it('give the plugins the icon of a command a plugin declares, and the plugin icon to one that declares none', () => {
    createWindow()

    const { icons } = h.args.plugins.getServices().workspace
    expect(icons.getCommand('org.example.sample:share.run')).toBe(getNamedIcon('share'))
    expect(icons.getCommand('org.example.sample:none.run')).toBe(Puzzle)
  })
})

describe('a tab closed', () => {
  it('is released and removed from the layout when no hold is passed', () => {
    const window = createWindow()

    expect(window.tabs.closeTab('t1')).toBe(true)

    expect(h.f.windowTabs.panes.releaseTab).toHaveBeenCalledWith('t1')
    expect(h.f.windowTabs.panes.finishClose).toHaveBeenCalledWith('t1')
  })

  it('returns false where the tab was not released', () => {
    h.f.windowTabs.panes.releaseTab.mockReturnValue(false)
    const window = createWindow()

    expect(window.tabs.closeTab('t1')).toBe(false)
  })

  it('calls the hold where the tab was not released', () => {
    h.f.windowTabs.panes.releaseTab.mockReturnValue(false)
    const window = createWindow()
    const hold = vi.fn()

    window.tabs.closeTab('t1', hold)

    expect(hold).toHaveBeenCalledOnce()
    expect(h.f.windowTabs.panes.finishClose).not.toHaveBeenCalled()
  })

  it('does not call the hold or finishClose where a hold is passed and the tab was released', () => {
    const window = createWindow()
    const hold = vi.fn()

    window.tabs.closeTab('t1', hold)

    expect(hold).not.toHaveBeenCalled()
    expect(h.f.windowTabs.panes.finishClose).not.toHaveBeenCalled()
  })

  it('needs no hold where the tab was not released', () => {
    h.f.windowTabs.panes.releaseTab.mockReturnValue(false)
    const window = createWindow()

    expect(() => window.tabs.closeTab('t1')).not.toThrow()
    expect(h.f.windowTabs.panes.finishClose).not.toHaveBeenCalled()
  })
})

describe('tab detach and move wiring', () => {
  it('calls the mover with the window id', async () => {
    const window = createWindow()

    window.tabs.moves.detachTab('t1', { x: 1, y: 2 })
    window.tabs.moves.detachTabInPlace('t1')
    window.tabs.moves.moveTab('t1', 'window-b')

    expect(h.f.tabMover.detachTab).toHaveBeenCalledWith('t1', { x: 1, y: 2 })
    expect(h.f.tabMover.detachTabInPlace).toHaveBeenCalledWith('t1')
    expect(h.f.tabMover.moveTab).toHaveBeenCalledWith('t1', 'window-b')
    expect(h.args.tabMover.windowId).toBe(window.tabs.moves.windowId)
  })

  it('closes the source tab through the tab closer', () => {
    createWindow()

    h.args.tabMover.closeTab('t1')

    expect(h.f.windowTabs.panes.releaseTab).toHaveBeenCalledWith('t1')
    expect(h.f.windowTabs.panes.finishClose).toHaveBeenCalledWith('t1')
  })

  it('reports a tab as asking when its kind is asking', () => {
    h.f.editing.reached.asking = vi.fn(() => false)
    h.f.windowTabs.tabs.getTab.mockReturnValue({
      kind: { events: { isAsking: () => true } },
      state: {},
    })
    const window = createWindow()

    expect(window.tabs.moves.isTabAsking('t1')).toBe(true)
  })
})

describe('moved tab wiring', () => {
  it('opens a moved tab through the opener, accepts it through the session, and sets the highlight', async () => {
    const window = createWindow()
    const moved = {
      tab: { id: 'plex:2', kind: 'plex', path: 'Tides.md', title: 'Tides', state: '' },
      moveId: 'move-1',
    }

    h.args.tabLinkSession.openMovedTab(moved)
    h.args.tabLinkSession.setHighlighted(true)

    await vi.waitFor(() => expect(h.f.openMovedTab).toHaveBeenCalledWith(moved))
    expect(h.args.movedTabOpener.acceptMove).toBe(h.f.tabLinkSession.acceptMove)
    expect(window.tabs.isHighlighted.value).toBe(true)
  })

  it('sends HighlightWindow through the core', async () => {
    const window = createWindow()

    window.tabs.moves.highlightWindow('window-b', true)

    await vi.waitFor(() => expect(h.core.highlightWindow).toHaveBeenCalledWith('window-b', true))
  })

  it('sends a highlight request after the previous one completes', async () => {
    let finishOn = (): void => undefined
    h.core.highlightWindow.mockImplementationOnce(
      () => new Promise<undefined>((resolve) => (finishOn = () => resolve(undefined))),
    )
    const window = createWindow()

    window.tabs.moves.highlightWindow('window-b', true)
    window.tabs.moves.highlightWindow('window-b', false)
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
    createWindow()

    expect(h.args.bootstrap.startEditing()).toBeUndefined()
    expect(h.f.editing.fileFlush.start).toHaveBeenCalledOnce()
  })

  it('closes the window, the editors, the tabs and the settings', () => {
    createWindow()

    h.args.bootstrap.closeWindow()
    h.args.bootstrap.closeEditing()
    h.args.bootstrap.closeTabs()
    h.args.bootstrap.closeSettings()

    expect(h.f.display.close).toHaveBeenCalledOnce()
    expect(h.f.editing.close).toHaveBeenCalledOnce()
    expect(h.f.windowTabs.restored.close).toHaveBeenCalledOnce()
    expect(h.f.settings.close).toHaveBeenCalledOnce()
  })
})

describe('the services a plugin is given', () => {
  it('are the ones the kinds made, the window task list, the background tasks, the commands with those over a tab, the vaults and the window', () => {
    createWindow()

    const { taskList, backgroundTasks, commands, vaults, window, ...made } =
      h.args.plugins.getServices()
    expect(Object.keys(made).sort()).toStrictEqual([
      'agent',
      'buffers',
      'files',
      'noteBuffer',
      'notes',
      'relations',
      'workspace',
    ])
    expect(made.noteBuffer).toBe(h.f.editing.notes)
    expect(taskList).toBe(h.f.display.tasks)
    expect(vaults?.getState()).toBeDefined()
    expect(commands?.getIds()).toStrictEqual(
      expect.arrayContaining([COMMAND.createVault, 'org.example.sample:share.run']),
    )
    expect(window?.getState()).toStrictEqual({ isMaximized: false, isFullscreen: false })
    expect(backgroundTasks).toStrictEqual({ name: 'backgroundTasks' })
  })

  it('lets a tab wait while the plugin status says its own plugin is starting', () => {
    createWindow()
    h.f.status.isPluginStarting.mockImplementation((pluginId: string) => pluginId === 'p1')

    expect(h.args.windowTabs.isPluginStarting('p1')).toBe(true)
    expect(h.args.windowTabs.isPluginStarting('p2')).toBe(false)
  })

  it('wires plugin support, windowTabs, settings, kinds, commands, background tasks, and lifecycle callbacks', async () => {
    const window = createWindow()

    // createWindowTabs options
    await h.args.windowTabs.enablePlugin('p1')
    expect(h.f.settings.rest.write.writeSetting).toHaveBeenCalledWith(['p1', 'enabled'], true)

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

    const resolveAddresses = vi
      .fn()
      .mockReturnValueOnce({
        found: [{ raw: '[[Target]]', address: 'name://Target', text: 'Target' }],
        resolved: [{ written: 'name://Target', path: 'resolved.md', isCrossed: false }],
      })
      .mockReturnValueOnce({ found: [], resolved: [] })
    answerCore({ notes: { resolveAddresses } })
    expect(await h.args.plugins.resolveLink('from.md', 'name://Target')).toBe('resolved.md')
    expect(resolveAddresses.mock.calls[0]?.[0]).toMatchObject({
      from: 'from.md',
      text: '[[Target]]',
    })
    expect(await h.args.plugins.resolveLink('from.md', 'Plain')).toBe('')

    // createTabSync & watchPluginSettings & useOpenTabs error callbacks
    h.f.writer.mockClear()
    h.args.tabSync.onError(createLostConnection('tab sync err'))
    expect(h.f.writer).toHaveBeenCalledWith('Numen is not responding; retrying', 'error')

    h.args.watchPlugins.onError(createLostConnection('watch plugins err'))
    expect(h.f.writer).toHaveBeenCalledWith(
      'Plugins were not tracked: Numen is not responding; retrying',
      'error',
    )

    expect(h.args.openTabs.isWriting.value).toBe(false)
    expect(window.lifecycle.isLayoutReady.value).toBe(false)
    h.args.openTabs.reportError(createLostConnection('save tabs err'))
    expect(h.f.writer).toHaveBeenCalledWith(
      'The open tabs were not saved: Numen is not responding; retrying',
      'error',
    )

    // useSettings callbacks
    h.args.settings.onSizeChanged()
    expect(h.f.editing.noted.measureAll).toHaveBeenCalledOnce()
    expect(h.args.settings.getSettingsViews('p1')).toEqual(['view-1'])

    // background tasks & noticeTasks
    expect(h.args.noticeTasks.window).toStrictEqual({
      lost: h.f.display.health.lost,
      unwatched: h.f.display.vault.unwatched,
      error: h.f.display.vault.error,
    })
    expect(h.args.backgroundTasks.tasks.value).toEqual([{ id: 'task:1' }, { id: 'notice:1' }])
    await h.args.backgroundTasks.cancelTask('task:1')
    expect(h.core.cancelTask).toHaveBeenCalledWith('task:1')
    h.args.backgroundTasks.reportCancelError(createLostConnection('cancel failed'))
    expect(h.f.writer).toHaveBeenCalledWith(
      'The task was not cancelled: Numen is not responding; retrying',
      'error',
    )
    h.args.backgroundTasks.openSettings('p1')
    expect(h.f.settings.configured.openSettings).toHaveBeenCalledWith('p1')

    // Returned window methods
    window.openSettings('p1', 'ocr')
    expect(h.f.settings.configured.openSettings).toHaveBeenCalledWith('p1', 'ocr')

    h.f.windowTabs.restored.openPlacedTabs.mockImplementationOnce(
      async (onErr: (kind: string, err: unknown) => void) => {
        onErr('graph', createLostConnection('graph failed'))
      },
    )
    window.lifecycle.openCurrentVault()
    await Promise.resolve()
    expect(h.f.writer).toHaveBeenCalledWith(
      'The graph tab did not open: Numen is not responding; retrying',
      'error',
    )

    h.f.windowTabs.restored.openPlacedTabs.mockRejectedValueOnce(
      createLostConnection('all tabs failed'),
    )
    window.lifecycle.openCurrentVault()
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

    await h.args.bootstrap.closePlugins()
    expect(h.f.pluginLoader.close).toHaveBeenCalledOnce()

    h.args.bootstrap.reportError(createLostConnection('boot failed'))
    expect(h.f.writer).toHaveBeenCalledWith(
      'The window did not start cleanly: Numen is not responding; retrying',
      'error',
    )

    h.f.editing.fileFlush.start.mockRejectedValueOnce(createLostConnection('flush died'))
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
      const window = createWindow()

      // Empty activeVault id returns early
      h.f.vaults.activeVault.value = { id: '' }
      await h.args.bootstrap.startLayout()
      expect(window.lifecycle.isLayoutReady.value).toBe(false)

      // Saved tabs present -> restores them
      h.f.vaults.activeVault.value = { id: 'vault' }
      h.core.readOpenTabs.mockResolvedValueOnce({
        tabs: [{ id: 'note:a', kind: 'note', title: 'A', path: 'a.md', state: '' }] as never,
        activeId: 'note:a',
      })
      await h.args.bootstrap.startLayout()
      expect(h.f.windowTabs.restored.restoreTabs).toHaveBeenCalledWith(
        [{ id: 'note:a', kind: 'note', title: 'A', path: 'a.md', saved: undefined }],
        'note:a',
      )
      expect(window.lifecycle.isLayoutReady.value).toBe(true)
      expect(h.args.openTabs.isWriting.value).toBe(true)

      // readDetachedTab fails -> the detach window opens no tab and does not restore the vault tabs
      vi.stubGlobal('location', new URL('http://localhost/?secondary&id=files:1'))
      const secondaryWin = createWindow()
      h.core.readOpenTabs.mockClear()

      h.core.readDetachedTab.mockRejectedValueOnce(new Error('offline'))
      await h.args.bootstrap.startLayout()
      expect(h.core.readDetachedTab).toHaveBeenCalledWith(h.args.tabLinkSession.windowId, 'files:1')
      expect(h.f.openMovedTab).not.toHaveBeenCalled()
      expect(h.core.readOpenTabs).not.toHaveBeenCalled()
      expect(h.f.windowTabs.restored.openPlacedTabs).not.toHaveBeenCalled()
      expect(secondaryWin.lifecycle.isLayoutReady.value).toBe(true)
      expect(h.args.openTabs.isWriting.value).toBe(false)
    } finally {
      vi.unstubAllGlobals()
    }
  })
})

describe('tab links wiring', () => {
  it('writes a link after the pending tab reports', async () => {
    let finishReports: () => void = () => undefined
    h.f.tabReports.waitForReports.mockImplementationOnce(
      () => new Promise<undefined>((resolve) => (finishReports = () => resolve(undefined))),
    )
    createWindow()

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
    createWindow()
    const windowId = h.args.tabLinkSession.windowId

    await h.args.tabLinks.write({ unlink: 'plex:1' })

    expect(windowId).toMatch(/^[0-9a-f-]{36}$/)
    expect(h.core.writeTabLinks).toHaveBeenCalledWith(windowId, { unlink: 'plex:1' })
    expect(h.args.tabLinkSession.tabLinks).toBe(h.f.tabLinks)
    expect(h.args.tabSync.tabLinks).toBe(h.f.tabLinks)
  })

  it('sends the new path of a linked tab to the core', async () => {
    createWindow()
    const sync = { sourceTabId: 'plex:1', path: 'Orbits.md', title: 'Orbits' }

    await h.args.tabSync.sendSync(sync)

    expect(h.core.syncTabPath).toHaveBeenCalledWith(sync)
  })

  it('starts with the window and stops when it closes', async () => {
    createWindow()

    await h.args.bootstrap.startWindow()
    expect(h.f.tabLinkSession.start).toHaveBeenCalledOnce()

    h.args.bootstrap.closeWindow()
    expect(h.f.tabLinkSession.close).toHaveBeenCalledOnce()
    expect(h.f.display.close).toHaveBeenCalledOnce()
  })

  it('reports the tabs again when the window report changes', async () => {
    createWindow()

    h.f.windowReport.value = { tabs: [], window: { isMain: true, tabCount: 1 } }
    await vi.waitFor(() => expect(h.f.tabLinkSession.reportTabs).toHaveBeenCalledOnce())
    expect(h.args.tabReports.getReport()).toStrictEqual(h.f.windowReport.value)
  })

  it('reports an error when the core rejects a link', () => {
    createWindow()

    h.args.tabLinks.onError(createLostConnection('refused'))

    expect(h.f.writer).toHaveBeenCalledWith(
      'The tabs were not linked: Numen is not responding; retrying',
      'error',
    )
  })

  it('places the tabs of a new vault once the plugins of its tabs started or failed, while the others go on', async () => {
    try {
      vi.stubGlobal('location', new URL('http://localhost/'))
      createWindow()
      h.f.vaults.activeVault.value = { id: 'vault' }
      h.core.readOpenTabs.mockResolvedValueOnce({ tabs: [], activeId: '' })
      let settleTabPlugins: () => void = () => undefined
      const tabPluginsSettled = new Promise<undefined>((resolve) => {
        settleTabPlugins = () => resolve(undefined)
      })
      h.f.pluginLoader.status.waitForTabPlugins.mockImplementation(() => tabPluginsSettled)
      h.f.pluginLoader.start.mockImplementationOnce(() => new Promise<undefined>(() => undefined))
      h.f.windowTabs.restored.openPlacedTabs.mockClear()
      h.args.bootstrap.startPlugins().catch(() => undefined)

      const layout = h.args.bootstrap.startLayout()
      await new Promise((resolve) => setTimeout(resolve, 0))
      expect(h.f.windowTabs.restored.openPlacedTabs).not.toHaveBeenCalled()
      settleTabPlugins()
      await layout

      expect(h.f.windowTabs.restored.openPlacedTabs).toHaveBeenCalledOnce()
    } finally {
      vi.unstubAllGlobals()
    }
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
      const window = createWindow()

      await h.args.bootstrap.startLayout()

      expect(h.core.readDetachedTab).toHaveBeenCalledWith(
        h.args.tabLinkSession.windowId,
        'plex:4f1c',
      )
      await vi.waitFor(() => expect(h.f.openMovedTab).toHaveBeenCalledWith(detached))
      expect(h.core.readOpenTabs).not.toHaveBeenCalled()
      expect(window.lifecycle.isLayoutReady.value).toBe(true)
      expect(h.args.emptyWindowClose.tabCount()).toBe(0)
    } finally {
      vi.unstubAllGlobals()
    }
  })

  it('closes the tab links session when the page hides', () => {
    createWindow()

    globalThis.dispatchEvent(new Event('pagehide'))

    expect(h.f.tabLinkSession.close).toHaveBeenCalled()
  })

  it('reopens the tab links session when the page is restored from the back-forward cache', () => {
    createWindow()
    h.f.tabLinkSession.reopen.mockClear()

    globalThis.dispatchEvent(Object.assign(new Event('pageshow'), { persisted: false }))
    expect(h.f.tabLinkSession.reopen).not.toHaveBeenCalled()

    globalThis.dispatchEvent(Object.assign(new Event('pageshow'), { persisted: true }))
    expect(h.f.tabLinkSession.reopen).toHaveBeenCalled()
  })

  it('passes the group paths of each snapshot to the tab sync', () => {
    createWindow()

    expect(h.args.tabLinkSession.applyGroupPaths).toBe(h.args.tabSyncResult.applyGroupPaths)
  })

  it('does not close the main window when its last tab closes', () => {
    createWindow()

    expect(h.args.emptyWindowClose).toBeUndefined()
  })
})
