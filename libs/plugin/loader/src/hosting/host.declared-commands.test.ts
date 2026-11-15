import { createMockManifest } from '@numen/core-client/testing'
import { describe, expect, it, vi } from 'vitest'
import { formatCommandId } from '../commandId'
import type { CommandDeclaration } from './types'
import { createTestPluginHost, type TestHostDeps } from '../testing/hostDeps'

const note = {
  path: 'a.md',
  title: 'A',
  type: 'note' as const,
  isFolder: false,
  others: [],
  tab: '',
}

const declaring = (pluginId: string) =>
  createMockManifest(pluginId, { commands: ['tree.open', 'tree.revealNote'] })

function declare(changes: Partial<CommandDeclaration> = {}): CommandDeclaration {
  return {
    pluginId: 'md.tree',
    id: 'tree.open',
    title: 'Open file tree',
    icon: 'folder-tree',
    ...changes,
  }
}

function createHost(declarations: CommandDeclaration[], extra: Partial<TestHostDeps> = {}) {
  const deps: TestHostDeps = {
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => declarations,
    reportCommandIssue: vi.fn(),
    ...extra,
  }
  return { host: createTestPluginHost(deps), deps }
}

describe('the commands a plugin declares in its manifest', () => {
  it('are listed with the title, icon and condition of the declaration, and the key bound to them, once the handler is registered', () => {
    const { host } = createHost([
      declare({
        id: 'tree.revealNote',
        title: 'Show note in file tree',
        when: 'target.isNote',
      }),
    ])

    host.keybindings.registerKeybindings('md.tree', [
      { key: 'mod+shift+e', command: 'tree.revealNote' },
    ])
    host.lifecycle
      .createContext('md.tree', declaring('md.tree'))
      .plugin.register.command({ id: 'tree.revealNote', handler: vi.fn() })

    expect(host.commands.getPaletteCommands()).toStrictEqual([
      {
        id: formatCommandId('md.tree', 'tree.revealNote'),
        title: 'Show note in file tree',
        icon: 'folder-tree',
        when: 'target.isNote',
      },
    ])
  })

  it('are not registered under an id the manifest does not declare, and the registration fails', () => {
    const { host, deps } = createHost([declare()])
    const ctx = host.lifecycle.createContext('md.tree', declaring('md.tree'))

    ctx.plugin.register.command({ id: 'tree.open', handler: vi.fn() })
    expect(() => ctx.plugin.register.command({ id: 'tree.close', handler: vi.fn() })).toThrowError(
      'Plugin "md.tree" registers the command "tree.close", which its manifest does not declare',
    )

    expect(host.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.tree:tree.open',
    ])
    expect(host.commands.hasCommand('md.tree:tree.close')).toBe(false)
    expect(deps.reportCommandIssue).not.toHaveBeenCalled()
  })

  it('are not listed while the plugin has no backend and registered no handler once its frontend has activated', () => {
    const { host, deps } = createHost([declare()])
    host.lifecycle.createContext('md.tree', declaring('md.tree'))

    expect(host.commands.getPaletteCommands()).toStrictEqual([])
    expect(deps.reportCommandIssue).toHaveBeenCalledWith(
      'md.tree',
      expect.stringContaining('tree.open'),
    )
  })

  it('are listed while the frontend of their plugin has not started, and running one starts it and waits', async () => {
    let settle: () => void = () => undefined
    const gate = new Promise<void>((resolve) => {
      settle = resolve
    })
    const { host } = createHost([declare()], { startPlugin: () => gate })
    const handler = vi.fn()

    expect(host.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.tree:tree.open',
    ])
    const running = host.commands.executeCommand('md.tree:tree.open', note)
    host.lifecycle
      .createContext('md.tree', declaring('md.tree'))
      .plugin.register.command({ id: 'tree.open', handler })
    expect(handler).not.toHaveBeenCalled()
    settle()

    expect(await running).toBe(true)
    expect(handler).toHaveBeenCalledWith(note, undefined, {})
  })

  it('are not run for a plugin that failed to activate', async () => {
    const { host } = createHost([declare()], { startPlugin: async () => undefined })
    host.lifecycle.createContext('md.tree', declaring('md.tree'))
    host.lifecycle.disposePlugin('md.tree')

    expect(await host.commands.executeCommand('md.tree:tree.open', note)).toBe(false)
  })

  it('are left to the backend, and not listed by the host, when the plugin has a backend and no handler is registered', () => {
    const { host } = createHost([declare({ id: 'deck.create' })], {
      hasPluginFrontend: () => false,
    })

    expect(host.commands.getPaletteCommands()).toStrictEqual([])
  })

  it('are gone with a disabled plugin, bind no shortcut, and return with the plugin', () => {
    const { host } = createHost([declare()])
    const handler = vi.fn()
    const keys = [{ key: 'mod+shift+e', command: 'tree.open' }]
    host.keybindings.registerKeybindings('md.tree', keys)
    host.lifecycle
      .createContext('md.tree', declaring('md.tree'))
      .plugin.register.command({ id: 'tree.open', handler })
    const event = new KeyboardEvent('keydown', { code: 'KeyE', shiftKey: true, ctrlKey: true })

    host.lifecycle.disposePlugin('md.tree')

    expect(host.commands.getPaletteCommands()).toStrictEqual([])
    expect(host.commands.handleShortcut(event, async () => note)).toBe(false)

    host.keybindings.registerKeybindings('md.tree', keys)
    host.lifecycle
      .createContext('md.tree', declaring('md.tree'))
      .plugin.register.command({ id: 'tree.open', handler })

    expect(host.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.tree:tree.open',
    ])
    expect(host.commands.handleShortcut(event, async () => note)).toBe(true)
  })

  it('are run through the full id the palette lists', async () => {
    const { host } = createHost([declare()])
    const handler = vi.fn()
    host.lifecycle
      .createContext('md.tree', declaring('md.tree'))
      .plugin.register.command({ id: 'tree.open', handler })

    expect(await host.commands.executeCommand('md.tree:tree.open', note)).toBe(true)

    expect(handler).toHaveBeenCalledWith(note, undefined, {})
  })

  it('keep two plugins that register the same short id apart', async () => {
    const { host } = createHost([
      declare(),
      declare({ pluginId: 'md.other', title: 'Open other tree' }),
    ])
    const first = vi.fn()
    const second = vi.fn()

    host.lifecycle
      .createContext('md.tree', declaring('md.tree'))
      .plugin.register.command({ id: 'tree.open', handler: first })
    host.lifecycle
      .createContext('md.other', declaring('md.other'))
      .plugin.register.command({ id: 'tree.open', handler: second })
    await host.commands.executeCommand('md.other:tree.open', note)

    expect(host.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.tree:tree.open',
      'md.other:tree.open',
    ])
    expect(first).not.toHaveBeenCalled()
    expect(second).toHaveBeenCalledTimes(1)
  })

  it('checks the permission a command declares when another plugin runs it', async () => {
    const run = vi.fn(async () => '')
    const { host } = createHost(
      [declare({ id: 'tree.prune', permission: 'vault:manage' }), declare({ id: 'tree.open' })],
      {
        getCommands: () => ({
          getIds: () => ['md.tree:tree.prune', 'md.tree:tree.open'],
          watch: () => ({ dispose: () => undefined }),
          listItems: async () => [],
          run,
        }),
        getSlotEntries: () => [
          { pluginId: 'md.tree', slot: 'titleBar.commands', command: 'tree.prune' },
          { pluginId: 'md.tree', slot: 'titleBar.commands', command: 'tree.open' },
        ],
      },
    )

    const writerOnly = host.lifecycle.createContext(
      'md.caller.writer',
      createMockManifest('md.caller.writer', { permissions: ['vault:write'] }),
    )
    await writerOnly.host.ui.commands!.run('md.tree:tree.open', note)
    expect(run).toHaveBeenCalledWith('md.tree:tree.open', note)

    await expect(writerOnly.host.ui.commands!.run('md.tree:tree.prune', note)).rejects.toThrowError(
      'Plugin "md.caller.writer" lacks required permission "vault:manage" for commands.run',
    )
    expect(run).toHaveBeenCalledTimes(1)

    const manager = host.lifecycle.createContext(
      'md.caller.manager',
      createMockManifest('md.caller.manager', { permissions: ['vault:write', 'vault:manage'] }),
    )
    await manager.host.ui.commands!.run('md.tree:tree.prune', note)
    expect(run).toHaveBeenCalledWith('md.tree:tree.prune', note)
  })
})
