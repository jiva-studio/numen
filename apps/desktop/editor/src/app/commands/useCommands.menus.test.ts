/** Running an entry of the menu of a search result, and what the window hands the search. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { formatErrorMessage } from '@numen/core-client'
import type { SlotHost, SlotItem } from '@numen/plugin-host'
import type { CommandTarget, ResultChoice } from '@/features/command-palette'
import { useCommands, type CommandsDepsOptions } from './useCommands'

const h = vi.hoisted(() => ({
  startCommand: vi.fn(),
  getObjection: vi.fn(),
  setCommandsOpen: vi.fn(),
  setPaletteOpen: vi.fn(),
  isCommandsOpen: { value: false },
  runInvocation: vi.fn(() => Promise.resolve()),
  searchOptions: null as null | {
    menu?: { getEntries(): readonly unknown[] }
    getTarget?: () => unknown
  },
  getExtraCommands: null as null | (() => readonly unknown[]),
}))

vi.mock('@/features/command-palette', () => ({
  ANSWER_WORDS: { name: 'answer words' },
  isChord: vi.fn(),
  findChordCommand: vi.fn(),
  findShortcutKeys: (shortcut: string) => ({ letter: shortcut }),
  runInvocation: h.runInvocation,
  registerCommand: vi.fn(),
  useSearch: (_core: unknown, _words: unknown, options: typeof h.searchOptions) => {
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
vi.mock('./commandDeps', () => ({ createCommandDeps: () => ({ name: 'command deps' }) }))

const ATLAS = 'org.example.atlas'
const SHOW = `${ATLAS}:graph.showNote`

const front = {
  tab: 'tab-1',
  kind: 'note',
  path: 'Animals.md',
  title: 'Animals',
  file: 'Animals.md',
  vault: { id: 'v', name: 'V' },
  isReady: true,
} as unknown as CommandTarget

const choice = (command: string, line = -1): ResultChoice => ({
  command,
  over: { path: 'Reef.md', file: 'Reef.md', title: 'Reef', ...(line >= 0 ? { line } : {}) },
  input: '',
})

const entry = (command: string, title: string): SlotItem => ({
  kind: 'command',
  key: `${ATLAS}/command/${command}`,
  pluginId: ATLAS,
  command,
  title,
  shortcut: '',
  hasItems: false,
})

function setup(
  options: {
    slots?: Pick<SlotHost, 'getEntries'>
    hasCommand?: (id: string) => boolean
    target?: CommandTarget
  } = {},
) {
  const executeCommand = vi.fn(async () => true)
  const writeMessage = vi.fn()
  const pluginHost = {
    getPaletteCommands: () => [{ id: SHOW, title: 'Show note in plex' }],
    hasCommandItems: () => false,
    hasCommand: options.hasCommand ?? ((id: string) => id === SHOW),
    executeCommand,
    handleShortcut: () => false,
  }
  const fileKinds = vi.fn(async (paths: readonly string[]) => {
    return new Map(paths.map((one) => [one, { type: 'sample' }]))
  })
  const used = useCommands({
    core: { fileKinds },
    words: {},
    windowTabs: { onKeyPress: () => false },
    getTarget: () => options.target ?? front,
    knows: {},
    paletteLists: {},
    runs: {},
    coverage: () => ({}),
    rest: {},
    writeMessage,
    pluginHost,
    ...(options.slots ? { slots: options.slots } : {}),
  } as unknown as CommandsDepsOptions)
  return { executeCommand, writeMessage, ...used }
}

beforeEach(() => {
  vi.clearAllMocks()
  h.isCommandsOpen.value = false
  h.searchOptions = null
})

describe('running the action of a search result that is an entry of its menu', () => {
  it('runs the command of a plugin once over the note it was chosen on, not the tab in front', async () => {
    const one = setup()

    one.runChoice(choice(SHOW))
    await vi.waitFor(() => expect(one.executeCommand).toHaveBeenCalled())

    expect(one.executeCommand).toHaveBeenCalledExactlyOnceWith(SHOW, {
      path: 'Reef.md',
      title: 'Reef',
      type: 'sample',
      isFolder: false,
      others: [],
      tab: '',
    })
    expect(h.startCommand).not.toHaveBeenCalled()
  })

  it('starts a command of the editor over the note it was chosen on, as a shortcut does', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup()

    one.runChoice(choice('app:note.rename'))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith(
      'app:note.rename',
      expect.objectContaining({
        path: 'Reef.md',
        file: 'Reef.md',
        title: 'Reef',
        tab: '',
        kind: null,
      }),
    )
    expect(h.runInvocation).toHaveBeenCalledExactlyOnceWith(
      { id: 'inv' },
      { name: 'command deps' },
      { name: 'answer words' },
    )
  })

  it('opens the palette at the step a command needs, runs nothing, and closes the search', () => {
    h.startCommand.mockImplementation(() => {
      h.isCommandsOpen.value = true
      return null
    })
    const one = setup()

    one.runChoice(choice('app:note.create'))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith(
      'app:note.create',
      expect.objectContaining({ path: 'Reef.md' }),
    )
    expect(h.runInvocation).not.toHaveBeenCalled()
    expect(one.executeCommand).not.toHaveBeenCalled()
    expect(h.setPaletteOpen).toHaveBeenCalledExactlyOnceWith(false)
    expect(one.writeMessage).not.toHaveBeenCalled()
  })

  it('writes the message of a plugin command that fails, as the palette writes it', async () => {
    const one = setup()
    const failure = new Error('the plex could not be drawn')
    one.executeCommand.mockRejectedValueOnce(failure)

    one.runChoice(choice(SHOW))
    await vi.waitFor(() => expect(one.writeMessage).toHaveBeenCalled())

    expect(one.writeMessage).toHaveBeenCalledExactlyOnceWith(formatErrorMessage(failure), 'error')
  })

  it('writes the message of an editor command that fails, as the palette writes it', async () => {
    const failure = new Error('the note could not be renamed')
    h.startCommand.mockReturnValue({ id: 'inv' })
    h.runInvocation.mockRejectedValueOnce(failure)
    const one = setup()

    one.runChoice(choice('app:note.rename'))
    await vi.waitFor(() => expect(one.writeMessage).toHaveBeenCalled())

    expect(one.writeMessage).toHaveBeenCalledExactlyOnceWith(formatErrorMessage(failure), 'error')
  })

  it('runs the command of a plugin that has no handler yet through the palette, which waits for it', async () => {
    let finish: () => void = () => undefined
    h.startCommand.mockReturnValue({ id: 'inv' })
    h.runInvocation.mockReturnValueOnce(new Promise<void>((resolve) => (finish = resolve)))
    const one = setup({ hasCommand: () => false })

    one.runChoice(choice(SHOW))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith(
      SHOW,
      expect.objectContaining({ path: 'Reef.md' }),
    )
    expect(h.runInvocation).toHaveBeenCalledTimes(1)
    expect(one.writeMessage).not.toHaveBeenCalled()
    finish()
    await Promise.resolve()
    expect(one.writeMessage).not.toHaveBeenCalled()
  })
})

describe('opening the note of a search result as a command', () => {
  const resultTarget = expect.objectContaining({
    path: 'Reef.md',
    file: 'Reef.md',
    title: 'Reef',
    tab: '',
    kind: null,
  })

  it('opens the chosen note when no note is in front', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const nothingInFront = { ...front, tab: '', kind: null, path: '', title: '', file: '' }
    const one = setup({ target: nothingInFront as CommandTarget })

    one.runChoice(choice('app:note.open'))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith('app:note.open', resultTarget)
    expect(h.runInvocation).toHaveBeenCalledExactlyOnceWith(
      { id: 'inv' },
      { name: 'command deps' },
      { name: 'answer words' },
    )
    expect(one.writeMessage).not.toHaveBeenCalled()
  })

  it('opens the chosen note at the line of the heading or passage', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup()

    one.runChoice(choice('app:note.open', 7))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith(
      'app:note.open',
      expect.objectContaining({ path: 'Reef.md', line: 7 }),
    )
  })

  it('opens the chosen note at its first line where the line is zero', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup()

    one.runChoice(choice('app:note.open', 0))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith(
      'app:note.open',
      expect.objectContaining({ path: 'Reef.md', line: 0 }),
    )
  })

  it('opens a note found by its name from the start, with no line', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup()

    one.runChoice(choice('app:note.open', -1))

    expect(h.startCommand).toHaveBeenCalledTimes(1)
    const target = h.startCommand.mock.calls[0]?.[1] as { line?: number }
    expect(target.line).toBeUndefined()
  })

  it('gives a plugin command the path, title and type of the note and no line', async () => {
    const one = setup()

    one.runChoice(choice(SHOW, 7))
    await vi.waitFor(() => expect(one.executeCommand).toHaveBeenCalled())

    expect(one.executeCommand).toHaveBeenCalledExactlyOnceWith(SHOW, {
      path: 'Reef.md',
      title: 'Reef',
      type: 'sample',
      isFolder: false,
      others: [],
      tab: '',
    })
  })

  it('opens the chosen note and not the note in front', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup()

    one.runChoice(choice('app:note.open'))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith('app:note.open', resultTarget)
    expect(h.startCommand).not.toHaveBeenCalledWith(
      'app:note.open',
      expect.objectContaining({ path: 'Animals.md' }),
    )
  })

  it('reaches the editor, not the plugin host, for the command of the editor', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup({ hasCommand: () => false })

    one.runChoice(choice('app:note.open'))

    expect(one.executeCommand).not.toHaveBeenCalled()
  })
})

describe('running the action of a chosen document', () => {
  const document: ResultChoice = {
    command: 'app:file.open',
    over: { path: '', file: 'library/a.bin', title: 'Atlas', selector: { page: 3 } },
    input: '',
  }

  it('starts the command that opens it over the file and the selector, not the tab in front', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup({ hasCommand: () => false })

    one.runChoice(document)

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith(
      'app:file.open',
      expect.objectContaining({
        path: '',
        file: 'library/a.bin',
        title: 'Atlas',
        tab: '',
        kind: null,
        selector: { page: 3 },
      }),
    )
    expect(h.runInvocation).toHaveBeenCalledExactlyOnceWith(
      { id: 'inv' },
      { name: 'command deps' },
      { name: 'answer words' },
    )
    expect(one.executeCommand).not.toHaveBeenCalled()
    expect(one.writeMessage).not.toHaveBeenCalled()
  })

  it('says why nothing opened where the command is unavailable, as for any command', () => {
    h.startCommand.mockReturnValue(null)
    h.getObjection.mockReturnValue('Open a vault first')
    const one = setup({ hasCommand: () => false })

    one.runChoice(document)

    expect(h.runInvocation).not.toHaveBeenCalled()
    expect(one.writeMessage).toHaveBeenCalledExactlyOnceWith('Open a vault first', 'error')
  })
})

describe('running the action of the row that creates a note', () => {
  const create = (command: string, input = 'Entropy'): ResultChoice => ({
    command,
    over: null,
    input,
  })

  it.each([
    'app:note.create',
    'app:note.createChild',
    'app:note.createParent',
    'app:note.createJump',
  ])('starts %s over the window as it is, with the text typed as the input', (command) => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup({ hasCommand: () => false })

    one.runChoice(create(command))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith(command, front, 'Entropy')
    expect(h.runInvocation).toHaveBeenCalledExactlyOnceWith(
      { id: 'inv' },
      { name: 'command deps' },
      { name: 'answer words' },
    )
  })

  it('starts the command with no input where no text was typed', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup({ hasCommand: () => false })

    one.runChoice(create('app:note.create', ''))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith('app:note.create', front)
  })

  it('creates nothing and says why where no vault is open, as for any command', () => {
    h.startCommand.mockReturnValue(null)
    h.getObjection.mockReturnValue('Open a vault first')
    const one = setup({ hasCommand: () => false })

    one.runChoice(create('app:note.create'))

    expect(h.runInvocation).not.toHaveBeenCalled()
    expect(one.writeMessage).toHaveBeenCalledExactlyOnceWith('Open a vault first', 'error')
  })
})

describe('running a command with a text typed for it', () => {
  it('passes the text on to the palette that starts the command', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup({ hasCommand: () => false })

    one.runCommand('app:note.create', front, 'Entropy')

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith('app:note.create', front, 'Entropy')
  })

  it('leaves a command of a plugin to run over the note, ignoring the text', async () => {
    const one = setup()

    one.runCommand(SHOW, front, 'Entropy')
    await vi.waitFor(() => expect(one.executeCommand).toHaveBeenCalled())

    expect(one.executeCommand).toHaveBeenCalledExactlyOnceWith(SHOW, {
      path: 'Animals.md',
      title: 'Animals',
      type: 'sample',
      isFolder: false,
      others: [],
      tab: 'tab-1',
    })
    expect(h.startCommand).not.toHaveBeenCalled()
  })
})

describe('the search the window builds', () => {
  it('reads its menu from the commands of the slot of a result, over a note', () => {
    const getEntries = vi.fn((slot: string) =>
      slot === 'palette.searchResult'
        ? [entry(SHOW, 'Show note in plex'), { kind: 'separator' as const, key: 'gap' }]
        : [entry('x:y.z', 'Wrong')],
    )

    setup({ slots: { getEntries } })

    expect(h.searchOptions?.menu?.getEntries()).toEqual([
      { command: SHOW, title: 'Show note in plex' },
    ])
    expect(getEntries).toHaveBeenCalledWith('palette.searchResult', {
      path: '',
      title: '',
      isFolder: false,
      others: [],
    })
  })

  it('reads the target of the window for the row that creates a note', () => {
    setup()

    expect(h.searchOptions?.getTarget?.()).toBe(front)
  })

  it('has no entry in its menu where the window has no slots', () => {
    setup()

    expect(h.searchOptions?.menu?.getEntries() ?? []).toEqual([])
  })
})

describe('the commands of the palette', () => {
  it('are the same whatever the plugins place in slots', () => {
    const listCommands = () =>
      ((h.getExtraCommands?.() ?? []) as { id: string; text: string; group: string }[]).map(
        ({ id, text, group }) => ({ id, text, group }),
      )
    setup()
    const baseline = listCommands()
    const slots: Pick<SlotHost, 'getEntries'> = {
      getEntries: () => [entry(SHOW, 'Show note in plex')],
    }

    setup({ slots })
    const withPlacements = listCommands()

    expect(withPlacements).toEqual(baseline)
    expect(withPlacements.map((one) => one.id)).toContain(SHOW)
  })
})
