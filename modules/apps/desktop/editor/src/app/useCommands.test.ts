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
}))

vi.mock('@/features/command-palette', () => ({
  ANSWER_WORDS: { name: 'answer words' },
  isChord: h.isChord,
  findChordCommand: h.findChordCommand,
  runInvocation: h.runInvocation,
  useSearch: () => ({ setOpen: h.setPaletteOpen }),
  useCommandPalette: () => ({
    startCommand: h.startCommand,
    getObjection: h.getObjection,
    setOpen: h.setCommandsOpen,
    open: h.isCommandsOpen,
  }),
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
  return { written: content, press, ...useCommands(options) }
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
    expect(one.written).toStrictEqual([])
  })

  it('closes the search palette where the command palette is open and starts nothing', () => {
    h.startCommand.mockReturnValue(null)
    h.isCommandsOpen.value = true
    const one = setup()

    one.runCommand('save', target)

    expect(h.runInvocation).not.toHaveBeenCalled()
    expect(h.setPaletteOpen).toHaveBeenCalledExactlyOnceWith(false)
    expect(one.written).toStrictEqual([])
  })

  it('writes the objection as an error where the command palette is closed and starts nothing', () => {
    h.startCommand.mockReturnValue(null)
    h.getObjection.mockReturnValue('Nothing to save')
    const one = setup()

    one.runCommand('save', target)

    expect(h.getObjection).toHaveBeenCalledWith('save', target)
    expect(h.runInvocation).not.toHaveBeenCalled()
    expect(h.setPaletteOpen).not.toHaveBeenCalled()
    expect(one.written).toStrictEqual([['Nothing to save', 'error']])
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
