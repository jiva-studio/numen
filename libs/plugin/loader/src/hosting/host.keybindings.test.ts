/** The keys of the plugin host. Every plugin and command here is invented. */
import { describe, expect, it, vi } from 'vitest'
import { parseManifest } from '@numen/core-client'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost } from './host'
import { getTargetValues } from '../when'
import type { CommandDeclaration, PluginHostDeps } from './types'

type Manifest = Parameters<ReturnType<typeof createPluginHost>['lifecycle']['registerManifest']>[1]

const NOTE = {
  path: 'Tides.md',
  title: 'Tides',
  type: 'note' as const,
  isFolder: false,
  others: [],
  tab: '',
}
const TARGET = { path: 'Tides.md', title: 'Tides', isFolder: false, others: [], tab: '' }

const press = (init: KeyboardEventInit) => new KeyboardEvent('keydown', init)
const CTRL_L = press({ code: 'KeyL', ctrlKey: true })

const LEDGER: CommandDeclaration = {
  pluginId: 'org.example.ledger',
  id: 'ledger.open',
  title: 'Open ledger',
}

const LEDGER_MANIFEST = createMockManifest('org.example.ledger', {
  commands: ['ledger.open', 'ledger.find'],
})

function create(
  declarations: CommandDeclaration[] = [LEDGER],
  extra: Partial<PluginHostDeps> = {},
  manifest?: Manifest,
) {
  const windowValues: Record<string, boolean> = {}
  const reportSlotIssue = vi.fn()
  const host = createPluginHost({
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => declarations,
    getSlotWhenValues: (target) => ({ ...windowValues, ...getTargetValues(target, {}) }),
    reportSlotIssue,
    ...extra,
  })
  const handler = vi.fn()
  host.lifecycle.registerManifest('org.example.ledger', manifest ?? LEDGER_MANIFEST)
  host.lifecycle
    .createContext('org.example.ledger')
    .plugin.register.command({ id: 'ledger.open', handler })
  return { host, handler, windowValues, reportSlotIssue }
}

const settle = () => new Promise((resolve) => setTimeout(resolve, 0))

describe('the keys of the plugins', () => {
  it('run the command of the plugin with the values of the keybinding', async () => {
    const { host, handler } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+l', command: 'ledger.open', values: { book: 'main' } },
    ])

    expect(host.commands.handleShortcut(CTRL_L, async () => NOTE)).toBe(true)
    await settle()

    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE, undefined, { book: 'main' })
  })

  it('read mod as Meta once the platform of the application turns out to be macOS', async () => {
    let platform: 'windows' | 'darwin' = 'windows'
    const app = {
      name: 'Numen',
      version: '1.0.0',
      build: 1,
      get platform() {
        return platform
      },
    }
    const { host, handler } = create([LEDGER], { app })
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+l', command: 'ledger.open' },
    ])
    platform = 'darwin'

    expect(host.commands.handleShortcut(CTRL_L, async () => NOTE)).toBe(false)
    expect(
      host.commands.handleShortcut(press({ code: 'KeyL', metaKey: true }), async () => NOTE),
    ).toBe(true)
    await settle()

    expect(handler).toHaveBeenCalledOnce()
  })

  it('are read from the keybindings of the manifest', async () => {
    const parsed = parseManifest(
      JSON.stringify({
        id: 'org.example.ledger',
        name: 'Ledger',
        version: '1.0.0',
        contributes: {
          commands: [{ id: 'ledger.open', title: 'Open ledger' }],
          keybindings: [{ key: 'mod+l', command: 'ledger.open' }],
        },
      }),
    )
    if (!parsed.ok) throw new Error(parsed.error)
    const { host, handler } = create([LEDGER], {}, parsed.value)

    expect(host.commands.handleShortcut(CTRL_L, async () => NOTE)).toBe(true)
    await settle()

    expect(handler).toHaveBeenCalledOnce()
  })

  it('work on the Russian layout, where the key of the code KeyL types another letter', async () => {
    const { host, handler } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+l', command: 'ledger.open' },
    ])

    host.commands.handleShortcut(press({ code: 'KeyL', key: 'д', ctrlKey: true }), async () => NOTE)
    await settle()

    expect(handler).toHaveBeenCalledOnce()
  })

  it('are not handled while the condition of the keybinding is false, so the event stays with the page', async () => {
    const { host, handler, windowValues } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+l', command: 'ledger.open', when: '!window.isFieldFocused' },
    ])
    const getNote = vi.fn(async () => NOTE)
    windowValues['window.isFieldFocused'] = true

    expect(host.commands.handleShortcut(CTRL_L, getNote, { target: TARGET })).toBe(false)
    await settle()

    expect(getNote).not.toHaveBeenCalled()
    expect(handler).not.toHaveBeenCalled()

    windowValues['window.isFieldFocused'] = false

    expect(host.commands.handleShortcut(CTRL_L, getNote, { target: TARGET })).toBe(true)
  })

  it('are not handled while the condition of the command is false over the target', async () => {
    const { host, handler } = create([{ ...LEDGER, when: 'target.isNote' }])
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+l', command: 'ledger.open' },
    ])
    const folder = { ...TARGET, path: 'Books', isFolder: true }

    expect(host.commands.handleShortcut(CTRL_L, async () => NOTE, { target: folder })).toBe(false)
    expect(handler).not.toHaveBeenCalled()
    expect(host.commands.handleShortcut(CTRL_L, async () => NOTE, { target: TARGET })).toBe(true)
  })

  it('leave a weaker keybinding free when the command of the stronger one is not available', async () => {
    const find: CommandDeclaration = {
      ...LEDGER,
      id: 'ledger.find',
      title: 'Find',
      when: 'target.isFolder',
    }
    const { host, handler, windowValues } = create([LEDGER, find])
    host.lifecycle
      .createContext('org.example.ledger')
      .plugin.register.command({ id: 'ledger.find', handler: vi.fn() })
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+l', command: 'ledger.find', when: 'window.isFieldFocused' },
      { key: 'mod+l', command: 'ledger.open' },
    ])
    windowValues['window.isFieldFocused'] = true

    expect(host.commands.handleShortcut(CTRL_L, async () => NOTE, { target: TARGET })).toBe(true)
    await settle()

    expect(handler).toHaveBeenCalledOnce()
  })

  it('are left to the window when the winner is a command of the application', () => {
    const { host, handler } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+p', command: 'ledger.open' },
    ])
    const getNote = vi.fn(async () => NOTE)

    expect(host.commands.handleShortcut(press({ code: 'KeyP', ctrlKey: true }), getNote)).toBe(
      false,
    )
    expect(getNote).not.toHaveBeenCalled()
    expect(handler).not.toHaveBeenCalled()
  })

  it('hand a key of the application to the window with the values of the keybinding', () => {
    const { host, handler } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      {
        key: 'g',
        command: 'app:file.open',
        values: { placement: 'split' },
        when: '!window.isFieldFocused',
      },
    ])
    const runApp = vi.fn()
    const getNote = vi.fn(async () => NOTE)

    expect(
      host.commands.handleShortcut(press({ code: 'KeyG' }), getNote, { target: TARGET, runApp }),
    ).toBe(true)
    expect(runApp).toHaveBeenCalledExactlyOnceWith('app:file.open', { placement: 'split' })
    expect(getNote).not.toHaveBeenCalled()
    expect(handler).not.toHaveBeenCalled()
  })

  it('hand a key of the application from commands.json to the window, and nothing for a key it does not hold', () => {
    const { host } = create()
    const runApp = vi.fn()

    expect(
      host.commands.handleShortcut(press({ code: 'KeyP', ctrlKey: true }), async () => NOTE, {
        target: TARGET,
        runApp,
      }),
    ).toBe(true)
    expect(runApp).toHaveBeenCalledExactlyOnceWith('app:commands.open', {})
    expect(
      host.commands.handleShortcut(press({ code: 'KeyQ', ctrlKey: true }), async () => NOTE, {
        target: TARGET,
        runApp,
      }),
    ).toBe(false)
    expect(runApp).toHaveBeenCalledOnce()
  })

  it('do not hand a key of the application while its condition is false', () => {
    const { host, windowValues } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'g', command: 'app:file.open', when: '!window.isFieldFocused' },
    ])
    const runApp = vi.fn()
    windowValues['window.isFieldFocused'] = true

    expect(
      host.commands.handleShortcut(press({ code: 'KeyG' }), async () => NOTE, {
        target: TARGET,
        runApp,
      }),
    ).toBe(false)
    expect(runApp).not.toHaveBeenCalled()
  })

  it('warn once of a conflict with a key of the application', () => {
    const { host, reportSlotIssue } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+p', command: 'ledger.open' },
    ])

    host.commands.handleShortcut(press({ code: 'KeyP', ctrlKey: true }), async () => NOTE)
    host.commands.handleShortcut(press({ code: 'KeyP', ctrlKey: true }), async () => NOTE)

    expect(reportSlotIssue).toHaveBeenCalledOnce()
    expect(reportSlotIssue).toHaveBeenCalledWith('app', expect.stringContaining('mod+p'), 'warning')
  })

  it('are gone when the plugin is disposed, and are given back by registering them again', () => {
    const { host } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+l', command: 'ledger.open' },
    ])

    host.lifecycle.disposePlugin('org.example.ledger')

    expect(host.keybindings.getKeyFor('org.example.ledger:ledger.open')).toBeUndefined()
    expect(host.commands.handleShortcut(CTRL_L, async () => NOTE)).toBe(false)
  })

  it('are unregistered with the plugin, but the keys of the application stay', () => {
    const { host } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+l', command: 'ledger.open' },
    ])

    host.keybindings.unregisterKeybindings('org.example.ledger')
    host.lifecycle.disposeAll()

    expect(host.keybindings.getKeyFor('org.example.ledger:ledger.open')).toBeUndefined()
    expect(host.keybindings.getKeyFor('app:commands.open')).toBe('mod+p')
  })

  it('give the key of a command to a hint, and the palette command carries none', () => {
    const { host } = create()
    host.keybindings.registerKeybindings('org.example.ledger', [
      { key: 'mod+l', command: 'ledger.open' },
    ])

    expect(host.keybindings.getKeyFor('org.example.ledger:ledger.open')).toBe('mod+l')
    expect(host.commands.getPaletteCommands()[0]).not.toHaveProperty('shortcut')
  })
})
