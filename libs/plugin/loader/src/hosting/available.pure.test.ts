import { describe, expect, it, vi } from 'vitest'
import { createPaletteCommand, getIssueKey, listAvailableCommands } from './available'
import type { HostRegistry, RegisteredCommand } from './registry'
import type { CommandDeclaration } from './types'
import {
  createTestHostDeps,
  createTestPluginHost,
  fillHostDeps,
  type TestHostDeps,
} from '../testing/hostDeps'

const declaration: CommandDeclaration = { pluginId: 'md.tree', id: 'tree.open', title: 'Open tree' }

function createRegistry(
  activated: string[],
  handlers: [string, RegisteredCommand][],
  starting: string[] = [],
): HostRegistry {
  return {
    commands: { value: new Map(handlers) },
    handlers: { value: new Map() },
    plugins: {
      activated: { value: new Set(activated) },
      starting: { value: new Set(starting) },
      disposed: new Set<string>(),
    },
    reportedIssues: new Set<string>(),
  } as unknown as HostRegistry
}

describe('listing the listed commands', () => {
  it('returns the commands and the issues, and writes and reports nothing', () => {
    const reportCommandIssue = vi.fn()
    const deps: TestHostDeps = {
      getCommandDeclarations: () => [declaration],
      reportCommandIssue,
    } as unknown as TestHostDeps
    const registry = createRegistry(['md.tree'], [])

    const first = listAvailableCommands(createTestHostDeps(deps), registry)
    const second = listAvailableCommands(createTestHostDeps(deps), registry)

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
    } as unknown as TestHostDeps

    const { commands, issues } = listAvailableCommands(
      createTestHostDeps(deps),
      createRegistry(['md.tree'], []),
    )

    expect(commands).toStrictEqual([])
    expect(issues).toStrictEqual([])
  })

  it('returns the unbound issue for a command with no handler of a plugin that has no backend', () => {
    const deps = {
      getCommandDeclarations: () => [declaration],
      hasPluginBackend: () => false,
    } as unknown as TestHostDeps

    const { commands, issues } = listAvailableCommands(
      createTestHostDeps(deps),
      createRegistry(['md.tree'], []),
    )

    expect(commands).toStrictEqual([])
    expect(issues).toStrictEqual([{ pluginId: 'md.tree', id: 'tree.open', kind: 'unbound' }])
  })

  it('returns no issue for a plugin whose frontend has not activated yet', () => {
    const deps: TestHostDeps = {
      getCommandDeclarations: () => [declaration],
    } as unknown as TestHostDeps

    const { commands, issues } = listAvailableCommands(
      createTestHostDeps(deps),
      createRegistry([], []),
    )

    expect(commands.map((one) => one.id)).toStrictEqual(['md.tree:tree.open'])
    expect(issues).toStrictEqual([])
  })

  it('returns empty lists before command declarations are available and keys issues by plugin, id, and kind', () => {
    expect(getIssueKey({ pluginId: 'md.tree', id: 'tree.open', kind: 'unbound' })).toBe(
      'md.tree:tree.open:unbound',
    )
    expect(listAvailableCommands(fillHostDeps(), createRegistry([], []))).toStrictEqual({
      commands: [],
      issues: [],
    })
    expect(
      listAvailableCommands(fillHostDeps({ commands: {} }), createRegistry([], [])),
    ).toStrictEqual({
      commands: [],
      issues: [],
    })

    const multi = listAvailableCommands(
      fillHostDeps({
        commands: {
          getCommandDeclarations: () => [
            declaration,
            { pluginId: 'md.tree', id: 'tree.close', title: 'Close tree' },
            { pluginId: 'md.other', id: 'tree.open', title: 'Open other' },
          ],
        },
      }),
      createRegistry(['md.tree', 'md.other'], []),
    )
    expect(multi.issues).toStrictEqual([
      { pluginId: 'md.tree', id: 'tree.open', kind: 'unbound' },
      { pluginId: 'md.tree', id: 'tree.close', kind: 'unbound' },
      { pluginId: 'md.other', id: 'tree.open', kind: 'unbound' },
    ])
  })

  it('leaves out commands of backend-only or failed plugins and keeps starting plugins available', () => {
    expect(
      listAvailableCommands(
        fillHostDeps({
          commands: { getCommandDeclarations: () => [declaration] },
          plugins: { hasPluginFrontend: () => false },
        }),
        createRegistry([], []),
      ),
    ).toStrictEqual({ commands: [], issues: [] })

    expect(
      listAvailableCommands(
        fillHostDeps({
          commands: { getCommandDeclarations: () => [declaration] },
          plugins: { hasPluginFrontend: () => true, isPluginFailed: () => true },
        }),
        createRegistry([], []),
      ),
    ).toStrictEqual({ commands: [], issues: [] })

    const starting = listAvailableCommands(
      fillHostDeps({
        commands: { getCommandDeclarations: () => [declaration] },
        plugins: { hasPluginFrontend: () => true, isPluginFailed: () => false },
      }),
      createRegistry(['md.tree'], [], ['md.tree']),
    )
    expect(starting.commands.map((one) => one.id)).toStrictEqual(['md.tree:tree.open'])
    expect(starting.issues).toStrictEqual([])
  })
})

describe('the host reading the listed commands', () => {
  it('reports one issue once however it is read: palette, shortcut and execute', async () => {
    const reportCommandIssue = vi.fn()
    const host = createTestPluginHost({
      target: 'desktop.editor',
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

describe('the palette entry of a declared command', () => {
  const answers = {
    keeps: 'Keep',
    retainedNotice: 'Nothing changes',
    action: 'Delete',
    detail: 'From',
  }
  const warns = { action: 'Erase', detail: 'For good', back: 'Type the name' }

  it('carries the confirmation its declaration holds', () => {
    const entry = createPaletteCommand({
      id: 'md.tree:tree.open',
      declaration: { ...declaration, answers, warns },
    })

    expect([entry.answers, entry.warns]).toStrictEqual([answers, warns])
  })

  it('has none where its declaration holds none', () => {
    const entry = createPaletteCommand({ id: 'md.tree:tree.open', declaration })

    expect(entry).not.toHaveProperty('answers')
    expect(entry).not.toHaveProperty('warns')
  })
})
