/** How a command runs, and what a key chord asks for. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CommandTarget } from '@/features/command-palette'
import { useCommands, type CommandsDepsOptions } from './useCommands'

const h = vi.hoisted(() => ({
  startCommand: vi.fn(),
  getObjection: vi.fn(),
  setCommandsOpen: vi.fn(),
  setPaletteOpen: vi.fn(),
  isCommandsOpen: { value: false },
  runInvocation: vi.fn(),
  isChord: vi.fn(),
  findChordCommand: vi.fn(),
  depsOptions: null as unknown,
  registerCommand: vi.fn(),
  getExtraCommands: null as null | (() => readonly unknown[]),
  searchOptions: null as null | { getTravel?: () => string },
}))

vi.mock('@/features/command-palette', () => ({
  ANSWER_WORDS: { name: 'answer words' },
  isChord: h.isChord,
  findChordCommand: h.findChordCommand,
  findShortcutKeys: (shortcut: string) => ({ letter: shortcut }),
  runInvocation: h.runInvocation,
  registerCommand: h.registerCommand,
  useSearch: (_core: unknown, _words: unknown, options: { getTravel?: () => string }) => {
    h.searchOptions = options
    return { setOpen: h.setPaletteOpen }
  },
  useCommandPalette: (...args: unknown[]) => {
    h.getExtraCommands = args[7] as () => readonly unknown[]
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

const setup = (preventHeld = false) => {
  const content: [string, string][] = []
  const press = vi.fn(() => preventHeld)
  const options = {
    core: {},
    words: {},
    windowTabs: { onKeyPress: press },
    getTarget: () => target,
    knows: {},
    kept: {},
    runs: {},
    coverage: () => ({}),
    writeMessage: (text: string, tone: string) => content.push([text, tone]),
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
  const PRESET = { id: 'plugin:preset', title: 'Open preset', scope: 'note' as const }
  const GRAPH = {
    id: 'plugin:graph',
    title: 'New graph',
    shortcut: 'mod+shift+x',
    icon: 'graph icon',
  }
  const SHOW = {
    id: 'plugin:show',
    title: 'Show in the graph',
    scope: 'note' as const,
    isNoteDestination: true,
  }

  const setupPlugin = (declared: readonly object[] = [PRESET]) => {
    const executeCommand = vi.fn(async () => true)
    const pluginHost = {
      getPaletteCommands: () => declared,
      hasCommand: (id: string) => id === 'preset',
      executeCommand,
      handleShortcut: () => false,
    }
    const fileKinds = vi.fn(
      async (paths: readonly string[]) => new Map(paths.map((one) => [one, { type: 'deck' }])),
    )
    const options = {
      core: { fileKinds },
      words: {},
      windowTabs: { onKeyPress: () => false },
      getTarget: () => target,
      knows: {},
      kept: {},
      runs: {},
      coverage: () => ({}),
      rest: {},
      writeMessage: vi.fn(),
      pluginHost,
    } as unknown as CommandsDepsOptions
    return { executeCommand, ...useCommands(options) }
  }

  const findOffer = () => {
    const offers = (h.getExtraCommands?.() ?? []) as {
      id: string
      group: string
      isOffered: (target: CommandTarget) => boolean
    }[]
    const offer = offers.find((one) => one.id === PRESET.id)
    if (!offer) throw new Error('the palette has no plugin command')
    return offer
  }

  it('is offered among the note commands only while a note is in front', () => {
    setupPlugin()
    const offer = findOffer()

    expect(offer.group).toBe('note')
    expect(offer.isOffered({ isReady: true, path: 'a.md' } as unknown as CommandTarget)).toBe(true)
    expect(offer.isOffered({ isReady: true, path: '' } as unknown as CommandTarget)).toBe(false)
  })

  it('runs from the palette with the note and the type the vault files it as', async () => {
    const one = setupPlugin()
    const [id, handler] = h.registerCommand.mock.calls[0] as [
      string,
      (invocation: { path: string; title: string }) => Promise<void>,
    ]

    await handler({ path: 'Animals.md', title: 'Animals' })

    expect(id).toBe(PRESET.id)
    expect(one.executeCommand).toHaveBeenCalledExactlyOnceWith(PRESET.id, {
      path: 'Animals.md',
      title: 'Animals',
      type: 'deck',
    })
  })

  it('draws a plugin command with the keystroke and the icon it declares', () => {
    setupPlugin([GRAPH])
    const offers = (h.getExtraCommands?.() ?? []) as {
      id: string
      keys?: unknown
      icon?: unknown
    }[]
    const offer = offers.find((one) => one.id === GRAPH.id)

    expect(offer?.keys).toStrictEqual({ letter: 'mod+shift+x' })
    expect(offer?.icon).toBe('graph icon')
  })

  it('offers the keyed commands over the window as ways in, and those over a note to the menus', () => {
    const one = setupPlugin([PRESET, GRAPH, { id: 'plugin:tree', title: 'Show the tree' }])

    expect(one.pluginWays.value).toStrictEqual([
      { id: GRAPH.id, text: GRAPH.title, keys: { letter: 'mod+shift+x' }, icon: 'graph icon' },
    ])
    expect(one.getNoteCommands()).toStrictEqual([
      { id: PRESET.id, text: PRESET.title, group: 'note' },
    ])
  })

  it('puts a note in front through the command declared for it, and offers it to the search', async () => {
    const one = setupPlugin([PRESET, SHOW])

    await expect(one.showNote('Animals.md')).resolves.toBe(true)

    expect(one.executeCommand).toHaveBeenCalledExactlyOnceWith(SHOW.id, {
      path: 'Animals.md',
      title: '',
      type: 'deck',
    })
    expect(h.searchOptions?.getTravel?.()).toBe(SHOW.title)
  })

  it('puts no note in front where no plugin declares a command for it', async () => {
    const one = setupPlugin([PRESET])

    await expect(one.showNote('Animals.md')).resolves.toBe(false)

    expect(one.executeCommand).not.toHaveBeenCalled()
    expect(h.searchOptions?.getTravel?.()).toBe('')
  })

  it('runs by id with the note of the target', async () => {
    const one = setupPlugin()

    one.runCommand('preset', { path: 'Animals.md', title: 'Animals' } as unknown as CommandTarget)
    await vi.waitFor(() => expect(one.executeCommand).toHaveBeenCalled())

    expect(one.executeCommand).toHaveBeenCalledExactlyOnceWith('preset', {
      path: 'Animals.md',
      title: 'Animals',
      type: 'deck',
    })
    expect(h.startCommand).not.toHaveBeenCalled()
  })
})
