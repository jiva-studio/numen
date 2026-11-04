import { describe, expect, it, vi } from 'vitest'
import { createPluginHost } from './host'
import { formatCommandId } from '../commandId'
import type { CommandDeclaration, PluginHostDeps } from './types'

const note = { path: 'a.md', title: 'A', type: 'note' as const }

function declare(changes: Partial<CommandDeclaration> = {}): CommandDeclaration {
  return {
    pluginId: 'md.tree',
    id: 'tree.open',
    title: 'Open file tree',
    icon: 'folder-tree',
    shortcut: 'mod+shift+e',
    ...changes,
  }
}

function createHost(declarations: CommandDeclaration[], extra: Partial<PluginHostDeps> = {}) {
  const deps: PluginHostDeps = {
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => declarations,
    reportCommandIssue: vi.fn(),
    ...extra,
  }
  return { host: createPluginHost(deps), deps }
}

describe('the commands a plugin declares in its manifest', () => {
  it('are listed with the title, icon, scope and shortcut of the declaration once the handler is registered', () => {
    const { host } = createHost([
      declare({
        id: 'tree.revealNote',
        title: 'Show note in file tree',
        scope: 'note',
      }),
    ])

    host.createContext('md.tree').registerCommand({ id: 'tree.revealNote', handler: vi.fn() })

    expect(host.getPaletteCommands()).toStrictEqual([
      {
        id: formatCommandId('md.tree', 'tree.revealNote'),
        title: 'Show note in file tree',
        icon: 'folder-tree',
        shortcut: 'mod+shift+e',
        scope: 'note',
      },
    ])
  })

  it('are not listed for a handler registered under an id the manifest does not declare, and the mismatch is reported once', () => {
    const { host, deps } = createHost([declare()])
    const ctx = host.createContext('md.tree')

    ctx.registerCommand({ id: 'tree.open', handler: vi.fn() })
    ctx.registerCommand({ id: 'tree.close', handler: vi.fn() })
    host.getPaletteCommands()
    host.getPaletteCommands()

    expect(host.getPaletteCommands().map((one) => one.id)).toStrictEqual(['md.tree:tree.open'])
    expect(host.hasCommand('md.tree:tree.close')).toBe(false)
    expect(deps.reportCommandIssue).toHaveBeenCalledTimes(1)
    expect(deps.reportCommandIssue).toHaveBeenCalledWith(
      'md.tree',
      expect.stringContaining('tree.close'),
    )
  })

  it('are not listed while the plugin has no backend and registered no handler once its frontend has activated', () => {
    const { host, deps } = createHost([declare()])
    host.createContext('md.tree')

    expect(host.getPaletteCommands()).toStrictEqual([])
    expect(deps.reportCommandIssue).toHaveBeenCalledWith(
      'md.tree',
      expect.stringContaining('tree.open'),
    )
  })

  it('are listed while the frontend of their plugin is still loading, and running one waits for the plugin', async () => {
    let settle: () => void = () => undefined
    const gate = new Promise<void>((resolve) => {
      settle = resolve
    })
    const { host } = createHost([declare()], { waitForPlugin: () => gate })
    const handler = vi.fn()

    expect(host.getPaletteCommands().map((one) => one.id)).toStrictEqual(['md.tree:tree.open'])
    const running = host.executeCommand('md.tree:tree.open', note)
    host.createContext('md.tree').registerCommand({ id: 'tree.open', handler })
    expect(handler).not.toHaveBeenCalled()
    settle()

    expect(await running).toBe(true)
    expect(handler).toHaveBeenCalledWith(note)
  })

  it('are not run for a plugin that failed to activate', async () => {
    const { host } = createHost([declare()], { waitForPlugin: async () => undefined })
    host.createContext('md.tree')
    host.disposePlugin('md.tree')

    expect(await host.executeCommand('md.tree:tree.open', note)).toBe(false)
  })

  it('are left to the backend, and not listed by the host, when the plugin has a backend and no handler is registered', () => {
    const { host } = createHost([declare({ id: 'deck.create' })], {
      hasPluginFrontend: () => false,
    })

    expect(host.getPaletteCommands()).toStrictEqual([])
  })

  it('are gone with a disabled plugin, bind no shortcut, and return with the plugin', () => {
    const { host } = createHost([declare()])
    const handler = vi.fn()
    host.createContext('md.tree').registerCommand({ id: 'tree.open', handler })
    const event = new KeyboardEvent('keydown', {
      key: 'e',
      shiftKey: true,
      ctrlKey: true,
      metaKey: true,
    })

    host.disposePlugin('md.tree')

    expect(host.getPaletteCommands()).toStrictEqual([])
    expect(host.handleShortcut(event, async () => note)).toBe(false)

    host.createContext('md.tree').registerCommand({ id: 'tree.open', handler })

    expect(host.getPaletteCommands().map((one) => one.id)).toStrictEqual(['md.tree:tree.open'])
    expect(host.handleShortcut(event, async () => note)).toBe(true)
  })

  it('are run through the full id the palette lists', async () => {
    const { host } = createHost([declare()])
    const handler = vi.fn()
    host.createContext('md.tree').registerCommand({ id: 'tree.open', handler })

    expect(await host.executeCommand('md.tree:tree.open', note)).toBe(true)

    expect(handler).toHaveBeenCalledWith(note)
  })

  it('keep two plugins that register the same short id apart', async () => {
    const { host } = createHost([
      declare(),
      declare({ pluginId: 'md.other', title: 'Open other tree' }),
    ])
    const first = vi.fn()
    const second = vi.fn()

    host.createContext('md.tree').registerCommand({ id: 'tree.open', handler: first })
    host.createContext('md.other').registerCommand({ id: 'tree.open', handler: second })
    await host.executeCommand('md.other:tree.open', note)

    expect(host.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.tree:tree.open',
      'md.other:tree.open',
    ])
    expect(first).not.toHaveBeenCalled()
    expect(second).toHaveBeenCalledTimes(1)
  })
})
