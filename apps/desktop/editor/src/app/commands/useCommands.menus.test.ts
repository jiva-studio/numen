/** Running an entry of the menu of a search result, and what the window hands the search. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { formatErrorMessage } from '@numen/core-client'
import type { Menus, PlacedCommand } from '@numen/plugin-host'
import type { CommandTarget, MenuChoice } from '@/features/command-palette'
import { useCommands, type CommandsDepsOptions } from './useCommands'

const h = vi.hoisted(() => ({
  startCommand: vi.fn(),
  getObjection: vi.fn(),
  setCommandsOpen: vi.fn(),
  setPaletteOpen: vi.fn(),
  isCommandsOpen: { value: false },
  runInvocation: vi.fn(() => Promise.resolve()),
  searchOptions: null as null | { menu?: { getEntries(): readonly unknown[] } },
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

const choice = (command: string): MenuChoice => ({
  command,
  path: 'Reef.md',
  title: 'Reef',
  type: 'sample',
})

const entry = (command: string, title: string): PlacedCommand => ({
  pluginId: ATLAS,
  command,
  title,
  group: '',
})

function setup(options: { menus?: Menus; hasCommand?: (id: string) => boolean } = {}) {
  const executeCommand = vi.fn(async () => true)
  const writeMessage = vi.fn()
  const pluginHost = {
    getPaletteCommands: () => [{ id: SHOW, title: 'Show note in plex' }],
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
    getTarget: () => front,
    knows: {},
    kept: {},
    runs: {},
    coverage: () => ({}),
    rest: {},
    writeMessage,
    pluginHost,
    ...(options.menus ? { menus: options.menus } : {}),
  } as unknown as CommandsDepsOptions)
  return { executeCommand, writeMessage, ...used }
}

beforeEach(() => {
  vi.clearAllMocks()
  h.isCommandsOpen.value = false
  h.searchOptions = null
})

describe('running an entry of the menu of a search result', () => {
  it('runs the command of a plugin once over the note it was chosen on, not the tab in front', async () => {
    const one = setup()

    one.runEntry(choice(SHOW))
    await vi.waitFor(() => expect(one.executeCommand).toHaveBeenCalled())

    expect(one.executeCommand).toHaveBeenCalledExactlyOnceWith(SHOW, {
      path: 'Reef.md',
      title: 'Reef',
      type: 'sample',
    })
    expect(h.startCommand).not.toHaveBeenCalled()
  })

  it('starts a command of the editor over the note it was chosen on, as a shortcut does', () => {
    h.startCommand.mockReturnValue({ id: 'inv' })
    const one = setup()

    one.runEntry(choice('app:note.rename'))

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

    one.runEntry(choice('app:note.create'))

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

    one.runEntry(choice(SHOW))
    await vi.waitFor(() => expect(one.writeMessage).toHaveBeenCalled())

    expect(one.writeMessage).toHaveBeenCalledExactlyOnceWith(formatErrorMessage(failure), 'error')
  })

  it('writes the message of an editor command that fails, as the palette writes it', async () => {
    const failure = new Error('the note could not be renamed')
    h.startCommand.mockReturnValue({ id: 'inv' })
    h.runInvocation.mockRejectedValueOnce(failure)
    const one = setup()

    one.runEntry(choice('app:note.rename'))
    await vi.waitFor(() => expect(one.writeMessage).toHaveBeenCalled())

    expect(one.writeMessage).toHaveBeenCalledExactlyOnceWith(formatErrorMessage(failure), 'error')
  })

  it('runs the command of a plugin that has no handler yet through the palette, which waits for it', async () => {
    let finish: () => void = () => undefined
    h.startCommand.mockReturnValue({ id: 'inv' })
    h.runInvocation.mockReturnValueOnce(new Promise<void>((resolve) => (finish = resolve)))
    const one = setup({ hasCommand: () => false })

    one.runEntry(choice(SHOW))

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

describe('the search the window builds', () => {
  it('reads its menu from the entries of the place for a result', () => {
    const getEntries = vi.fn((location: string) =>
      location === 'searchResult' ? [entry(SHOW, 'Show note in plex')] : [entry('x:y.z', 'Wrong')],
    )

    setup({ menus: { getEntries } })

    expect(h.searchOptions?.menu?.getEntries()).toEqual([entry(SHOW, 'Show note in plex')])
    expect(getEntries).toHaveBeenCalledWith('searchResult')
  })

  it('has no entry in its menu where the window has no places', () => {
    setup()

    expect(h.searchOptions?.menu?.getEntries() ?? []).toEqual([])
  })
})

describe('the commands of the palette', () => {
  it('are the same whatever the plugins place in menus', () => {
    const listCommands = () =>
      ((h.getExtraCommands?.() ?? []) as { id: string; text: string; group: string }[]).map(
        ({ id, text, group }) => ({ id, text, group }),
      )
    setup()
    const baseline = listCommands()
    const menus: Menus = { getEntries: () => [entry(SHOW, 'Show note in plex')] }

    setup({ menus })
    const withPlacements = listCommands()

    expect(withPlacements).toEqual(baseline)
    expect(withPlacements.map((one) => one.id)).toContain(SHOW)
  })
})
