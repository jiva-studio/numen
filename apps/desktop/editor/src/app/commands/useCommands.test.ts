/** How a command runs, and what a key chord triggers. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CommandDeps, CommandTarget, PaletteLists } from '@/features/command-palette'
import { getNamedIcon } from '@/shared/icons'
import { createCommandRunner, useCommands, type CommandsDepsOptions } from './useCommands'

const h = vi.hoisted(() => ({
  startCommand: vi.fn(),
  getObjection: vi.fn(),
  setCommandsOpen: vi.fn(),
  setPaletteOpen: vi.fn(),
  isCommandsOpen: { value: false },
  runInvocation: vi.fn(() => Promise.resolve()),
  isChord: vi.fn(),
  findChordCommand: vi.fn(),
  depsOptions: null as unknown,
  registerCommand: vi.fn(),
  getExtraCommands: null as null | (() => readonly unknown[]),
  lists: null as null | PaletteLists,
}))

vi.mock('@/features/command-palette', () => ({
  ANSWER_WORDS: { name: 'answer words' },
  isChord: h.isChord,
  findChordCommand: h.findChordCommand,
  findShortcutKeys: (shortcut: string) => ({ letter: shortcut }),
  runInvocation: h.runInvocation,
  registerCommand: h.registerCommand,
  useSearch: () => {
    return { setOpen: h.setPaletteOpen }
  },
  useCommandPalette: (...args: unknown[]) => {
    h.getExtraCommands = args[7] as () => readonly unknown[]
    h.lists = args[4] as PaletteLists
    return {
      startCommand: h.startCommand,
      getObjection: h.getObjection,
      setOpen: h.setCommandsOpen,
      open: h.isCommandsOpen,
    }
  },
}))
vi.mock('./commandDeps', () => ({
  createCommandDeps: (options: unknown) => {
    h.depsOptions = options
    return { name: 'command deps' }
  },
}))

const target = { path: 'a.md' } as unknown as CommandTarget

const inFront = (path: string, isReady = true) =>
  ({
    tab: '',
    path,
    file: path,
    vault: { id: 'v', name: 'V' },
    isReady,
  }) as unknown as CommandTarget

const setup = (isPrevented = false) => {
  const content: [string, string][] = []
  const press = vi.fn(() => isPrevented)
  const options = {
    core: {},
    words: {},
    windowTabs: { onKeyPress: press },
    getTarget: () => target,
    knows: {},
    paletteLists: {},
    runs: {},
    coverage: () => ({}),
    rest: {},
    writeMessage: (text: string, severity: string) => content.push([text, severity]),
  } as unknown as CommandsDepsOptions
  return { messages: content, press, ...useCommands(options) }
}

const keyEvent = (key: string, overrides: Partial<KeyboardEvent> = {}) => {
  const event = {
    key,
    shiftKey: false,
    defaultPrevented: false,
    preventDefault: vi.fn(),
    ...overrides,
  }
  return event as unknown as KeyboardEvent & { preventDefault: ReturnType<typeof vi.fn> }
}

beforeEach(() => {
  vi.clearAllMocks()
  h.isCommandsOpen.value = false
  h.isChord.mockReturnValue(true)
  h.findChordCommand.mockReturnValue('save')
})

describe('running a command', () => {
  it('runs the invocation the palette starts, with the command deps', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup()

    one.runCommand('save', target)

    expect(h.startCommand).toHaveBeenCalledWith('save', target)
    expect(h.runInvocation).toHaveBeenCalledWith({ id: 'inv' }, one.commandDeps, {
      name: 'answer words',
    })
    expect(h.setPaletteOpen).not.toHaveBeenCalled()
    expect(one.messages).toStrictEqual([])
    expect(h.getExtraCommands?.()).toStrictEqual([])
  })

  it('closes the search palette where the command palette is open and starts nothing', () => {
    h.startCommand.mockReturnValue(null)
    h.isCommandsOpen.value = true
    const one = setup()

    one.runCommand('save', target)

    expect(h.runInvocation).not.toHaveBeenCalled()
    expect(h.setPaletteOpen).toHaveBeenCalledExactlyOnceWith(false)
    expect(one.messages).toStrictEqual([])
  })

  it('writes the objection as an error where the command palette is closed and starts nothing', () => {
    h.startCommand.mockReturnValue(null)
    h.getObjection.mockReturnValue('Nothing to save')
    const one = setup()

    one.runCommand('save', target)

    expect(h.getObjection).toHaveBeenCalledWith('save', target)
    expect(h.runInvocation).not.toHaveBeenCalled()
    expect(h.setPaletteOpen).not.toHaveBeenCalled()
    expect(one.messages).toStrictEqual([['Nothing to save', 'error']])
  })

  it('closes the command palette and opens the search palette for the search command', () => {
    setup()
    const options = h.depsOptions as { search: () => void }

    options.search()

    expect(h.setCommandsOpen).toHaveBeenCalledExactlyOnceWith(false)
    expect(h.setPaletteOpen).toHaveBeenCalledExactlyOnceWith(true)
  })
})

describe('a run to its end, as a plugin awaits it', () => {
  /** The runner over the palette mocks, a plugin host where one is given, and the messages written. */
  const setupRunner = (pluginHost?: object) => {
    const messages: [string, string | undefined][] = []
    const writeMessage = (text: string, severity?: string) => messages.push([text, severity])
    const runner = createCommandRunner({
      commands: {
        startCommand: h.startCommand,
        getObjection: h.getObjection,
        setOpen: h.setCommandsOpen,
        open: h.isCommandsOpen,
      } as unknown as Parameters<typeof createCommandRunner>[0]['commands'],
      palette: { setOpen: h.setPaletteOpen },
      pluginHost: pluginHost as CommandsDepsOptions['pluginHost'],
      getCommandNote: async (one) => ({ ...one, type: 'note' }) as never,
      getCommandDeps: () => ({ writeMessage }) as unknown as CommandDeps,
      writeMessage,
    })
    return { runner, messages }
  }

  it('settles when the invocation has run, and writes its error nowhere but rejects with it', async () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    h.runInvocation.mockImplementationOnce((async (_invocation: unknown, on: CommandDeps) => {
      on.writeMessage('', 'info')
      on.writeMessage('the disk is full', 'error')
    }) as unknown as () => Promise<void>)
    const { runner, messages } = setupRunner()

    await expect(runner.runCommandToEnd('save', target)).rejects.toThrow('the disk is full')
    expect(messages).toStrictEqual([['', 'info']])
  })

  it('rejects with the objection to a command that cannot start, and writes it nowhere', async () => {
    h.startCommand.mockReturnValue(null)
    h.getObjection.mockReturnValue('Nothing to save')
    const { runner, messages } = setupRunner()

    await expect(runner.runCommandToEnd('save', target)).rejects.toThrow('Nothing to save')
    expect(messages).toStrictEqual([])
  })

  it('has the host await a plugin command, and rejects with its error, which it writes nowhere', async () => {
    const executeCommand = vi.fn(async () => {
      throw new Error('plugin command failed')
    })
    const { runner, messages } = setupRunner({
      hasCommand: () => true,
      isSharedCommand: () => false,
      executeCommand,
    })

    await expect(runner.runCommandToEnd('summary', target)).rejects.toThrow('plugin command failed')
    expect(executeCommand).toHaveBeenCalledExactlyOnceWith('summary', expect.any(Object), {
      isAwaited: true,
    })
    expect(messages).toStrictEqual([])
  })
})

describe('a key press', () => {
  it('runs the command of the chord, requested for by the lowercase key', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup()
    const event = keyEvent('S', { shiftKey: true })

    one.onKeyDown(event)

    expect(h.findChordCommand).toHaveBeenCalledWith('s', true)
    expect(h.startCommand).toHaveBeenCalledWith('save', target)
    expect(event.preventDefault).toHaveBeenCalledOnce()
  })

  it('does nothing where the event was already handled', () => {
    const one = setup()
    const event = keyEvent('s', { defaultPrevented: true })

    one.onKeyDown(event)

    expect(one.press).not.toHaveBeenCalled()
    expect(h.startCommand).not.toHaveBeenCalled()
  })

  it('stops at the tabs where they take the key', () => {
    const one = setup(true)
    const event = keyEvent('s')

    one.onKeyDown(event)

    expect(event.preventDefault).toHaveBeenCalledOnce()
    expect(h.isChord).not.toHaveBeenCalled()
  })

  it('does nothing where the key is no chord', () => {
    h.isChord.mockReturnValue(false)
    const one = setup()
    const event = keyEvent('s')

    one.onKeyDown(event)

    expect(h.findChordCommand).not.toHaveBeenCalled()
    expect(event.preventDefault).not.toHaveBeenCalled()
  })

  it('does nothing where the chord has no command', () => {
    h.findChordCommand.mockReturnValue(null)
    const one = setup()
    const event = keyEvent('s')

    one.onKeyDown(event)

    expect(h.startCommand).not.toHaveBeenCalled()
    expect(event.preventDefault).not.toHaveBeenCalled()
  })
})

describe('a plugin command over a note', () => {
  const SUMMARY = {
    id: 'org.example.sample:summary.run',
    title: 'Open summary',
    when: 'target.isNote',
    shortcut: 'mod+k',
  }
  const GRAPH = {
    id: 'org.example.sample:graph.run',
    title: 'New graph',
    shortcut: 'mod+shift+x',
    icon: 'waypoints',
  }
  const setupPlugin = (
    declared: readonly object[] = [SUMMARY],
    extra: Partial<CommandsDepsOptions> = {},
  ) => {
    const executeCommand = vi.fn(async () => true)
    const handleShortcut = vi.fn(() => false)
    const writeMessage = vi.fn()
    const pluginHost = {
      getPaletteCommands: () => declared,
      hasCommandItems: () => false,
      hasCommand: (id: string) => id === 'summary',
      executeCommand,
      handleShortcut,
    }
    const fileKinds = vi.fn(
      async (paths: readonly string[]) =>
        new Map(
          paths.filter((one) => one !== 'Unknown.md').map((one) => [one, { type: 'sample' }]),
        ),
    )
    const options = {
      core: { fileKinds },
      words: {},
      windowTabs: { onKeyPress: () => false },
      getTarget: () => ({ path: 'Animals.md', title: 'Animals' }),
      knows: {},
      paletteLists: {},
      runs: {},
      coverage: () => ({}),
      rest: {},
      writeMessage,
      pluginHost,
      ...extra,
    } as unknown as CommandsDepsOptions
    return { executeCommand, handleShortcut, writeMessage, ...useCommands(options) }
  }

  const findCommand = () => {
    const entries = (h.getExtraCommands?.() ?? []) as {
      id: string
      group: string
      isAvailable: (target: CommandTarget) => boolean
    }[]
    const entry = entries.find((one) => one.id === SUMMARY.id)
    if (!entry) throw new Error('the palette has no plugin command')
    return entry
  }

  it('is offered among the note commands only while a note is in front', () => {
    setupPlugin()
    const entry = findCommand()

    expect(entry.group).toBe('note')
    expect(entry.isAvailable(inFront('a.md'))).toBe(true)
    expect(entry.isAvailable(inFront(''))).toBe(false)
  })

  it('runs from the palette with the note and the type the vault files it as', async () => {
    const one = setupPlugin()
    const [id, handler] = h.registerCommand.mock.calls[0] as [
      string,
      (invocation: { path: string; title: string }) => Promise<void>,
    ]

    await handler({ path: 'Animals.md', title: 'Animals' })

    expect(id).toBe(SUMMARY.id)
    expect(one.executeCommand).toHaveBeenCalledExactlyOnceWith(SUMMARY.id, {
      path: 'Animals.md',
      title: 'Animals',
      type: 'sample',
      isFolder: false,
      others: [],
      tab: '',
    })
  })

  it('draws a plugin command with the keystroke and the icon it declares', () => {
    setupPlugin([GRAPH])
    const entries = (h.getExtraCommands?.() ?? []) as {
      id: string
      group: string
      keys?: unknown
      icon?: unknown
      isAvailable: (target: CommandTarget) => boolean
    }[]
    const entry = entries.find((one) => one.id === GRAPH.id)

    expect(entry?.group).toBe('window')
    expect(entry?.isAvailable(inFront('', false))).toBe(true)
    expect(entry?.keys).toStrictEqual({ letter: 'mod+shift+x' })
    expect(entry?.icon).toBe(getNamedIcon('waypoints'))
  })

  it('offers the keyed commands over the window as ways in', () => {
    const one = setupPlugin([
      SUMMARY,
      GRAPH,
      { id: 'org.example.sample:tree.run', title: 'Show the tree' },
    ])

    expect(one.pluginWays.value).toStrictEqual([
      {
        id: GRAPH.id,
        text: GRAPH.title,
        keys: { letter: 'mod+shift+x' },
        icon: getNamedIcon('waypoints'),
      },
    ])
  })

  it('runs by id with the note of the target, the others selected with it, and reports errors', async () => {
    const one = setupPlugin()

    one.runCommand('summary', {
      path: 'Animals.md',
      title: 'Animals',
      isFolder: false,
      others: ['Birds.md'],
    } as unknown as CommandTarget)
    await vi.waitFor(() => expect(one.executeCommand).toHaveBeenCalled())

    expect(h.setCommandsOpen).toHaveBeenCalledWith(false)
    expect(one.executeCommand).toHaveBeenCalledExactlyOnceWith('summary', {
      path: 'Animals.md',
      title: 'Animals',
      type: 'sample',
      isFolder: false,
      others: ['Birds.md'],
      tab: '',
    })
    expect(h.startCommand).not.toHaveBeenCalled()

    one.executeCommand.mockRejectedValueOnce(new Error('plugin command failed'))
    one.runCommand('summary', { path: 'Animals.md', title: 'Animals' } as unknown as CommandTarget)
    await vi.waitFor(() => expect(one.writeMessage).toHaveBeenCalled())
    expect(one.writeMessage).toHaveBeenCalledWith(expect.any(String), 'error')
  })

  it('reads the note of every run through the one reader the window gives, a run to its end too', async () => {
    const note = {
      path: 'Animals.md',
      title: 'Animals',
      type: 'deck',
      isFolder: false,
      others: [],
      tab: '',
    }
    const readCommandNote = vi.fn(async () => note)
    const one = setupPlugin([SUMMARY], { readCommandNote })
    const target = { path: 'Animals.md', title: 'Animals' } as unknown as CommandTarget

    one.runCommand('summary', target)
    await one.runCommandToEnd('summary', target)

    expect(readCommandNote).toHaveBeenCalledTimes(2)
    expect(one.executeCommand.mock.calls.map((call) => (call as unknown[])[1])).toStrictEqual([
      note,
      note,
    ])
  })

  it('delegates shortcuts to pluginHost.handleShortcut with a lazy note getter', async () => {
    const one = setupPlugin()
    let resolvedNote: unknown = null
    one.handleShortcut.mockImplementationOnce(((
      _ev: KeyboardEvent,
      getNote: () => Promise<unknown>,
    ) => {
      getNote().then(
        (n) => {
          resolvedNote = n
        },
        () => undefined,
      )
      return true
    }) as never)
    const event = keyEvent('k')

    one.onKeyDown(event)
    await vi.waitFor(() =>
      expect(resolvedNote).toStrictEqual({
        path: 'Animals.md',
        title: 'Animals',
        type: 'sample',
        isFolder: false,
        others: [],
        tab: '',
      }),
    )
    expect(event.preventDefault).toHaveBeenCalledOnce()
    expect(h.isChord).not.toHaveBeenCalled()
  })
})

describe('a plugin command with items in the palette', () => {
  const MOVE = { id: 'org.example.tide:chart.move', title: 'Move chart to window' }
  const WINDOWS = [
    { id: 'w1', title: 'First window' },
    { id: 'w2', title: 'Second window' },
  ]

  const setupItems = () => {
    let answer: (items: readonly { id: string; title: string }[]) => void = () => undefined
    const listCommandItems = vi.fn(
      () =>
        new Promise<readonly { id: string; title: string }[]>((resolve) => {
          answer = resolve
        }),
    )
    const executeCommand = vi.fn(async () => true)
    const windowLists = { getStepGroups: vi.fn(() => []), previewItem: vi.fn() }
    const pluginHost = {
      getPaletteCommands: () => [MOVE],
      hasCommand: () => false,
      hasCommandItems: (id: string) => id === MOVE.id,
      listCommandItems,
      executeCommand,
      handleShortcut: vi.fn(() => false),
    }
    const options = {
      core: { fileKinds: async () => new Map() },
      words: {},
      windowTabs: { onKeyPress: () => false },
      getTarget: () => ({ tab: 'plex:1', path: 'Tide.md', title: 'Tide' }),
      knows: {},
      paletteLists: windowLists,
      runs: {},
      coverage: () => ({}),
      rest: {},
      writeMessage: vi.fn(),
      pluginHost,
    } as unknown as CommandsDepsOptions
    useCommands(options)
    return {
      listCommandItems,
      executeCommand,
      windowLists,
      answer: (list: typeof WINDOWS) => answer(list),
    }
  }

  const rows = (typed = '') =>
    (h.lists?.getStepGroups(MOVE.id, typed) ?? []).flatMap((group) => group.items)

  it('asks the palette for a pick list before it runs', () => {
    setupItems()
    const entry = (h.getExtraCommands?.() ?? []).find(
      (one) => (one as { id: string }).id === MOVE.id,
    ) as { input?: string }

    expect(entry.input).toBe('items')
  })

  it('lists a loading row while its items arrive, then the items of the target', async () => {
    const { listCommandItems, answer } = setupItems()

    expect(rows().map((one) => [one.title, one.isDisabled ?? false])).toStrictEqual([
      ['Loading…', true],
    ])
    answer(WINDOWS)
    await vi.waitFor(() => expect(rows()).toHaveLength(2))

    expect(rows().map((one) => one.title)).toStrictEqual(['First window', 'Second window'])
    expect(listCommandItems).toHaveBeenCalledExactlyOnceWith(MOVE.id, {
      path: 'Tide.md',
      title: 'Tide',
      isFolder: false,
      others: [],
      tab: 'plex:1',
    })
  })

  it('keeps the items that hold what is typed', async () => {
    const { answer } = setupItems()
    rows()
    answer(WINDOWS)
    await vi.waitFor(() => expect(rows()).toHaveLength(2))

    expect(rows('second').map((one) => one.id)).toStrictEqual(['w2'])
  })

  it('runs the command with the item picked as its itemId', async () => {
    const { executeCommand } = setupItems()
    const call = h.registerCommand.mock.calls.find(([id]) => id === MOVE.id) as [
      string,
      (invocation: { path: string; title: string; name: string }) => Promise<void>,
    ]

    await call[1]({ path: 'Tide.md', title: 'Tide', name: 'w2' })

    expect(executeCommand).toHaveBeenCalledExactlyOnceWith(
      MOVE.id,
      expect.objectContaining({ path: 'Tide.md' }),
      { itemId: 'w2' },
    )
  })

  it('leaves the lists of the window to a command without items', () => {
    const { windowLists } = setupItems()

    h.lists?.getStepGroups('app:setting.theme', 'dark')

    expect(windowLists.getStepGroups).toHaveBeenCalledWith('app:setting.theme', 'dark')
  })
})
