/** The keys of a window through the one handler, over the registry of the plugin host. All names are invented. */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPluginHost, getWhenValues } from '@numen/plugin-host'
import { useCommands, type CommandsDepsOptions } from './useCommands'

const h = vi.hoisted(() => ({
  startCommand: vi.fn(),
  setCommandsOpen: vi.fn(),
  setSearchOpen: vi.fn(),
  runInvocation: vi.fn(() => Promise.resolve()),
}))

vi.mock('@/features/command-palette', () => ({
  ANSWER_WORDS: {},
  findShortcutKeys: () => undefined,
  runInvocation: h.runInvocation,
  createInvocation: (id: string, on: { path: string }) => ({ id, path: on.path }),
  registerCommand: vi.fn(),
  useSearch: () => ({ setOpen: h.setSearchOpen, open: { value: false } }),
  useCommandPalette: () => ({
    startCommand: h.startCommand,
    getObjection: vi.fn(),
    setOpen: h.setCommandsOpen,
    open: { value: false },
  }),
}))
vi.mock('./commandDeps', () => ({ createCommandDeps: () => ({}) }))

const target = { path: 'Tides.md', title: 'Tides', isFolder: false, others: [], tab: '' }

const windowValues = { 'window.isFieldFocused': false, 'window.isPaletteOpen': false }

const create = (
  keybindings: readonly {
    key: string
    command: string
    when?: string
    values?: Record<string, string>
  }[],
) => {
  const pluginHost = createPluginHost({
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getSlotWhenValues: () => getWhenValues({ window: windowValues, target: undefined, contentTypes: {} }),
  })
  pluginHost.registerKeybindings('org.example.ledger', keybindings)
  return useCommands({
    core: {},
    words: {},
    windowTabs: { onKeyPress: () => false },
    getTarget: () => target,
    knows: {},
    paletteLists: {},
    runs: {},
    coverage: () => ({}),
    rest: {},
    writeMessage: vi.fn(),
    pluginHost,
  } as unknown as CommandsDepsOptions)
}

const press = (code: string, init: KeyboardEventInit = {}) =>
  new KeyboardEvent('keydown', { code, cancelable: true, ...init })

beforeEach(() => {
  vi.clearAllMocks()
  windowValues['window.isFieldFocused'] = false
  windowValues['window.isPaletteOpen'] = false
  h.startCommand.mockReturnValue(null)
})

describe('a letter typed into a field', () => {
  it('fires a bare key whose condition does not name the field', () => {
    const one = create([{ key: 'x', command: 'app:note.create' }])
    windowValues['window.isFieldFocused'] = true
    const event = press('KeyX')

    one.onKeyDown(event)

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith('app:note.create', target)
    expect(event.defaultPrevented).toBe(true)
  })

  it('fires no bare key that guards itself with window.isFieldFocused being false', () => {
    const one = create([{ key: 'x', command: 'app:note.create', when: '!window.isFieldFocused' }])
    windowValues['window.isFieldFocused'] = true
    const inField = press('KeyX')

    one.onKeyDown(inField)

    expect(h.startCommand).not.toHaveBeenCalled()
    expect(inField.defaultPrevented).toBe(false)

    windowValues['window.isFieldFocused'] = false
    const free = press('KeyX')
    one.onKeyDown(free)

    expect(h.startCommand).toHaveBeenCalledOnce()
    expect(free.defaultPrevented).toBe(true)
  })
})

describe('a key while a palette is open', () => {
  it('fires no key that guards itself with window.isPaletteOpen being false', () => {
    const one = create([{ key: 'x', command: 'app:note.create', when: '!window.isPaletteOpen' }])
    windowValues['window.isPaletteOpen'] = true
    const event = press('KeyX')

    one.onKeyDown(event)

    expect(h.startCommand).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(false)
  })

  it('still fires a key of the application, which names no palette', () => {
    const one = create([])
    windowValues['window.isPaletteOpen'] = true

    one.onKeyDown(press('KeyN', { ctrlKey: true }))

    expect(h.startCommand).toHaveBeenCalledExactlyOnceWith('app:note.create', target)
  })
})

describe('the default of a key press', () => {
  it('is prevented only where a keybinding matched', () => {
    const one = create([{ key: 'x', command: 'app:note.create' }])
    const matched = press('KeyX')
    const unmatched = press('KeyY')

    one.onKeyDown(matched)
    one.onKeyDown(unmatched)

    expect(matched.defaultPrevented).toBe(true)
    expect(unmatched.defaultPrevented).toBe(false)
    expect(h.startCommand).toHaveBeenCalledOnce()
  })

  it('is prevented for the keys of the application, which come from commands.json', () => {
    const one = create([])
    const search = press('KeyK', { ctrlKey: true })
    const commands = press('KeyP', { ctrlKey: true })
    const shifted = press('KeyK', { ctrlKey: true, shiftKey: true })

    one.onKeyDown(search)
    one.onKeyDown(commands)
    one.onKeyDown(shifted)

    expect(search.defaultPrevented).toBe(true)
    expect(commands.defaultPrevented).toBe(true)
    expect(shifted.defaultPrevented).toBe(false)
    expect(h.setSearchOpen).toHaveBeenCalledWith(true)
    expect(h.setCommandsOpen).toHaveBeenCalledWith(true)
  })
})

describe('a key bound to a command of the application with values', () => {
  it('runs the command over the front target with the values, offered or not', () => {
    const one = create([
      {
        key: 'mod+shift+p',
        command: 'app:file.reveal',
        values: { kind: 'plex' },
        when: 'target.isNote',
      },
    ])
    const event = press('KeyP', { ctrlKey: true, shiftKey: true })

    one.onKeyDown(event)

    expect(h.runInvocation.mock.calls[0]?.[0]).toEqual({
      id: 'app:file.reveal',
      path: 'Tides.md',
      values: { kind: 'plex' },
    })
    expect(h.startCommand).not.toHaveBeenCalled()
    expect(event.defaultPrevented).toBe(true)
  })
})
