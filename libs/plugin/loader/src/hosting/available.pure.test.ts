import { describe, expect, it, vi } from 'vitest'
import { createPluginHost } from './host'
import { listAvailableCommands } from './available'
import type { HostRegistry, RegisteredCommand } from './registry'
import type { CommandDeclaration, PluginHostDeps } from './types'

const declaration: CommandDeclaration = { pluginId: 'md.tree', id: 'tree.open', title: 'Open tree' }

function createRegistry(
  activated: string[],
  handlers: [string, RegisteredCommand][],
): HostRegistry {
  return {
    commands: { value: new Map(handlers) },
    handlers: { value: new Map() },
    activated: { value: new Set(activated) },
    disposed: new Set<string>(),
    reportedIssues: new Set<string>(),
  } as unknown as HostRegistry
}

describe('listing the listed commands', () => {
  it('returns the commands and the issues, and writes and reports nothing', () => {
    const reportCommandIssue = vi.fn()
    const deps: PluginHostDeps = {
      getCommandDeclarations: () => [declaration],
      reportCommandIssue,
    } as unknown as PluginHostDeps
    const registry = createRegistry(['md.tree'], [])

    const first = listAvailableCommands(deps, registry)
    const second = listAvailableCommands(deps, registry)

    expect(first.commands).toStrictEqual([])
    expect(first.issues).toStrictEqual([{ pluginId: 'md.tree', id: 'tree.open', kind: 'unbound' }])
    expect(second).toStrictEqual(first)
    expect(registry.reportedIssues.size).toBe(0)
    expect(reportCommandIssue).not.toHaveBeenCalled()
  })

  it('returns no issue for a command with no handler of a plugin that has a backend', () => {
    const deps = {
      getCommandDeclarations: () => [declaration],
      hasPluginBackend: () => true,
    } as unknown as PluginHostDeps

    const { commands, issues } = listAvailableCommands(deps, createRegistry(['md.tree'], []))

    expect(commands).toStrictEqual([])
    expect(issues).toStrictEqual([])
  })

  it('returns the unbound issue for a command with no handler of a plugin that has no backend', () => {
    const deps = {
      getCommandDeclarations: () => [declaration],
      hasPluginBackend: () => false,
    } as unknown as PluginHostDeps

    const { commands, issues } = listAvailableCommands(deps, createRegistry(['md.tree'], []))

    expect(commands).toStrictEqual([])
    expect(issues).toStrictEqual([{ pluginId: 'md.tree', id: 'tree.open', kind: 'unbound' }])
  })

  it('returns no issue for a plugin whose frontend has not activated yet', () => {
    const deps: PluginHostDeps = {
      getCommandDeclarations: () => [declaration],
    } as unknown as PluginHostDeps

    const { commands, issues } = listAvailableCommands(deps, createRegistry([], []))

    expect(commands.map((one) => one.id)).toStrictEqual(['md.tree:tree.open'])
    expect(issues).toStrictEqual([])
  })
})

describe('the host reading the listed commands', () => {
  it('reports one issue once however it is read: palette, shortcut and execute', async () => {
    const reportCommandIssue = vi.fn()
    const host = createPluginHost({
      registerKind: vi.fn(() => vi.fn()),
      registerExtensionReader: vi.fn(() => vi.fn()),
      openTab: vi.fn(async () => 'tab-1'),
      getCommandDeclarations: () => [declaration],
      reportCommandIssue,
    })
    host.keybindings.registerKeybindings('md.tree', [{ key: 'e', command: 'tree.open' }])
    host.lifecycle.createContext('md.tree')
    const event = new KeyboardEvent('keydown', { key: 'e' })
    const note = {
      path: 'a.md',
      title: 'A',
      type: 'note' as const,
      isFolder: false,
      others: [],
      tab: '',
    }

    host.commands.getPaletteCommands()
    host.commands.handleShortcut(event, async () => note)
    await host.commands.executeCommand('md.tree:tree.open', note)
    host.commands.hasCommand('md.tree:tree.open')
    host.commands.getPaletteCommands()

    expect(reportCommandIssue).toHaveBeenCalledTimes(1)
    expect(reportCommandIssue).toHaveBeenCalledWith('md.tree', expect.stringContaining('tree.open'))
  })
})
