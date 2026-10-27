/**
 * The window put together: what each part is handed of the others, and what
 * the window answers about the tab in front.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useWindow } from './useWindow'

/** What each part was made with, and what it was made as. */
const h = vi.hoisted(() => ({
  args: {} as Record<string, any>,
  f: {} as Record<string, any>,
  core: { name: 'core', fileKinds: vi.fn() },
}))

vi.mock('../vault', () => ({ core: h.core }))
vi.mock('@/features/command-palette', () => ({ runSupport: () => ({ name: 'runs' }) }))
vi.mock('@/features/tab-linking', () => ({ useTabLinks: () => h.f.tabLinks }))
vi.mock('@/shared/notices/messages', () => ({ createMessageLog: () => h.f.log }))
vi.mock('@/entities/tab', () => ({
  createWorkspace: vi.fn(),
  fileOpeners: (args: unknown) => {
    h.args.openers = args
    return h.f.tabOpeners
  },
  useWindowTabs: () => h.f.windowTabs,
}))
vi.mock('../tabs/useTabSync', () => ({ createTabSync: () => ({ name: 'tabSync' }) }))
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
vi.mock('../tabs/useOpenTabs', () => ({ useOpenTabs: () => h.f.openTabs }))
vi.mock('../commands/useCommands', () => ({
  useCommands: (args: unknown) => {
    h.args.commands = args
    return h.f.commands
  },
}))
vi.mock('./useWindowDisplay', () => ({
  useWindowDisplay: (_core: unknown, args: unknown) => {
    h.args.display = args
    return h.f.display
  },
}))
vi.mock('../shell/useWindowNotices', () => ({ useWindowNotices: () => ({ name: 'notices' }) }))
vi.mock('./useWindowKinds', () => ({
  useWindowKinds: (args: unknown) => {
    h.args.kinds = args
    return h.f.kinds
  },
}))
vi.mock('../shell/useWindowStatusBar', () => ({
  useWindowStatusBar: () => ({ name: 'statusBar' }),
}))
vi.mock('../commands/useAppHotkeys', () => ({ useAppHotkeys: vi.fn() }))
vi.mock('../plugins/useWindowPlugins', () => ({
  createPluginSupport: (args: unknown) => {
    h.args.plugins = args
    return { pluginHost: h.f.pluginHost, pluginLoader: {} }
  },
  watchPluginSettings: vi.fn(),
}))
vi.mock('./useAppBootstrap', () => ({
  useAppBootstrap: (args: unknown) => {
    h.args.bootstrap = args
  },
}))

interface FrontTab {
  id: string
  kind: unknown
  state?: unknown
}

/** The tab in front, as the window's tabs hand it out. */
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
    tabLinks: { removeClosedTab: vi.fn() },
    tabOpeners: { openFileAt: vi.fn() },
    windowTabs: {
      layout: ref(null),
      tabs: ref<{ id: string; title: string }[]>([]),
      handle: { front: () => front.value },
      getTab: vi.fn((id: string) =>
        front.value?.id === id ? { kind: front.value.kind } : undefined,
      ),
      releaseTab: vi.fn(() => true),
      reconcileTab: vi.fn(),
      openTabOfKind: vi.fn(),
      close: vi.fn(),
    },
    editing: {
      notes: { applyPathChanges: vi.fn() },
      changes: { reportChange: vi.fn() },
      noted: { measureAll: vi.fn() },
      reached: { getTabAt: vi.fn(() => null) },
      getTitle: vi.fn(() => 'a title'),
      making: {},
      fileFlush: { start: vi.fn() },
      close: vi.fn(),
    },
    display: {
      chunks: {},
      embedded: {},
      isEmbedding: {},
      tasks: ref([]),
      failure: ref(null),
      opening: ref('opening'),
      start: vi.fn(),
      close: vi.fn(),
    },
    settings: {
      kept: {},
      dressed: {},
      oneName: {},
      nesting: {},
      rest: {},
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
    openTabs: { getTabIcon: vi.fn() },
    commands: {
      commands: { applyRenames: vi.fn() },
      runCommand: vi.fn(),
      commandDeps: { name: 'commandDeps' },
      palette: {},
      onKeyDown: vi.fn(),
      showNote: vi.fn(() => Promise.resolve(true)),
      getNoteCommands: vi.fn(() => [{ id: 'plugin:share', text: 'Share', group: 'note' }]),
      pluginWays: ref([]),
    },
    pluginHost: {
      notifyVaultChange: vi.fn(),
      getPaletteCommands: vi.fn(() => [{ id: 'plugin:share', title: 'Share', icon: 'icon' }]),
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
      source: null,
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
      source: 'src',
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
    const spans = [{ from: 1, to: 2 }]

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

    await expect(h.args.kinds.showNote('a.md')).resolves.toBe(true)
    expect(h.f.commands.showNote).toHaveBeenCalledWith('a.md')
    expect(h.args.kinds.getNoteCommands()).toStrictEqual([
      { id: 'plugin:share', text: 'Share', group: 'note' },
    ])
    expect(h.args.kinds.getPluginCommandIcon('plugin:share')).toBe('icon')
    expect(h.args.kinds.getPluginCommandIcon('plugin:none')).toBeUndefined()
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
      h.f.windowTabs.tabs.value = [{ id: 'deck:a', title: 'Animals' }]
      h.f.windowTabs.getTab.mockReturnValue({
        state: { path: 'a.md' },
        kind: { flushPending: () => Promise.resolve(), getOpenTab: () => ({ path: 'a.md' }) },
      })
    }

    it('is the tab it is open in', () => {
      stand()
      useWindow()

      expect(h.args.commands.knows.getTabAt('a.md')).toBe('deck:a')
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
  it('is forgotten by the links and released', () => {
    const window = useWindow()

    window.closeTab('t1')

    expect(h.f.tabLinks.removeClosedTab).toHaveBeenCalledWith('t1')
    expect(h.f.windowTabs.releaseTab).toHaveBeenCalledWith('t1')
  })

  it('calls the hold where the tab was not released', () => {
    h.f.windowTabs.releaseTab.mockReturnValue(false)
    const window = useWindow()
    const hold = vi.fn()

    window.closeTab('t1', hold)

    expect(hold).toHaveBeenCalledOnce()
  })

  it('does not call the hold where the tab was released', () => {
    const window = useWindow()
    const hold = vi.fn()

    window.closeTab('t1', hold)

    expect(hold).not.toHaveBeenCalled()
  })

  it('needs no hold where the tab was not released', () => {
    h.f.windowTabs.releaseTab.mockReturnValue(false)
    const window = useWindow()

    expect(() => window.closeTab('t1')).not.toThrow()
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

describe('the layout started with an initial tab', () => {
  it('opens tab of kind with at payload from URL search params', async () => {
    h.f.windowTabs.openTabOfKind.mockResolvedValue('note:hello.md')
    const originalLocation = globalThis.location
    try {
      // @ts-expect-error mocking location for test
      delete globalThis.location
      globalThis.location = new URL(
        'http://localhost/?window=secondary&tab=note:hello.md',
      ) as unknown as Location
      useWindow()

      await h.args.bootstrap.startLayout()

      expect(h.f.windowTabs.openTabOfKind).toHaveBeenCalledWith('note', 'hello.md')
      expect(h.f.windowTabs.layout.value).toEqual({
        root: { kind: 'pane', id: 'main', tabs: ['note:hello.md'], active: 'note:hello.md' },
        axis: 'horizontal',
        focus: 'main',
      })
    } finally {
      globalThis.location = originalLocation
    }
  })
})

describe('the services a plugin is given', () => {
  it('are the ones the kinds made and the window task list, handed to the plugins', () => {
    useWindow()

    const { taskList, ...made } = h.args.plugins.services
    expect(made).toStrictEqual(h.f.kinds.services)
    expect(taskList).toBe(h.f.display.tasks)
  })
})
