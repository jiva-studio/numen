/**
 * The id a command is filed under: the plugin that provides it and the id the plugin gave it.
 */
import { createMockManifest } from '@numen/core-client/testing'
import { describe, expect, it, vi } from 'vitest'
import { formatCommandId } from '../commandId'
import type { CommandDeclaration } from './types'
import { createTestPluginHost, type TestHostDeps } from '../testing/hostDeps'

const declare = (pluginId: string, id: string): CommandDeclaration => ({
  pluginId,
  id,
  title: id,
})

const createDeps = (...declarations: CommandDeclaration[]): TestHostDeps => ({
  target: 'desktop.editor',
  registerKind: vi.fn(() => vi.fn()),
  registerExtensionReader: vi.fn(() => vi.fn()),
  openTab: vi.fn(async () => 'tab-1'),
  getCommandDeclarations: () => declarations,
})

const declaring = (pluginId: string) =>
  createMockManifest(pluginId, { commands: ['tree.open', 'note.open'] })

const NOTE = {
  path: 'Notes/One.md',
  title: 'One',
  type: 'note' as const,
  isFolder: false,
  others: [],
  tab: '',
}

describe('the full id of a command', () => {
  it('is the owner, a colon and the short id', () => {
    expect(formatCommandId('md.example.notes', 'note.create')).toBe('md.example.notes:note.create')
    expect(formatCommandId('app', 'tab.close')).toBe('app:tab.close')
  })
})

describe('a command a plugin registers', () => {
  it('is filed under the id of the plugin and its own id', async () => {
    const host = createTestPluginHost(createDeps(declare('md.example.files', 'tree.open')))
    const handler = vi.fn()
    host.lifecycle
      .createContext('md.example.files', declaring('md.example.files'))
      .plugin.register.command({ id: 'tree.open', handler })

    expect(host.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.example.files:tree.open',
    ])
    expect(host.commands.hasCommand('md.example.files:tree.open')).toBe(true)
    expect(await host.commands.executeCommand('md.example.files:tree.open', NOTE)).toBe(true)
    expect(handler).toHaveBeenCalledOnce()
  })

  it('is not found by its short id or under the old plugin prefix', async () => {
    const host = createTestPluginHost(createDeps(declare('md.example.files', 'tree.open')))
    host.lifecycle
      .createContext('md.example.files', declaring('md.example.files'))
      .plugin.register.command({ id: 'tree.open', handler: vi.fn() })

    expect(host.commands.hasCommand('tree.open')).toBe(false)
    expect(host.commands.hasCommand('plugin:tree.open')).toBe(false)
    expect(await host.commands.executeCommand('plugin:tree.open', NOTE)).toBe(false)
  })

  it('is listed once for each of two plugins that give the same short id', async () => {
    const host = createTestPluginHost(
      createDeps(declare('md.example.one', 'note.open'), declare('md.example.two', 'note.open')),
    )
    const first = vi.fn()
    const second = vi.fn()
    host.lifecycle
      .createContext('md.example.one', declaring('md.example.one'))
      .plugin.register.command({ id: 'note.open', handler: first })
    host.lifecycle
      .createContext('md.example.two', declaring('md.example.two'))
      .plugin.register.command({ id: 'note.open', handler: second })

    expect(host.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.example.one:note.open',
      'md.example.two:note.open',
    ])
    await host.commands.executeCommand('md.example.two:note.open', NOTE)
    expect(second).toHaveBeenCalledOnce()
    expect(first).not.toHaveBeenCalled()
  })

  it('is rejected when its full id is already registered, naming the id', () => {
    const host = createTestPluginHost(createDeps())
    const context = host.lifecycle.createContext('md.example.one', declaring('md.example.one'))
    const first = vi.fn()
    context.plugin.register.command({ id: 'note.open', handler: first })

    expect(() => context.plugin.register.command({ id: 'note.open', handler: vi.fn() })).toThrow(
      'md.example.one:note.open',
    )
    expect(host.registered.getRegisteredCommands().map((one) => one.handler)).toStrictEqual([first])
  })

  it('is accepted again after its plugin is disposed', () => {
    const host = createTestPluginHost(createDeps())
    host.lifecycle
      .createContext('md.example.one', declaring('md.example.one'))
      .plugin.register.command({ id: 'note.open', handler: vi.fn() })
    host.lifecycle.disposePlugin('md.example.one')
    const again = vi.fn()

    host.lifecycle
      .createContext('md.example.one', declaring('md.example.one'))
      .plugin.register.command({ id: 'note.open', handler: again })

    expect(host.registered.getRegisteredCommands().map((one) => one.handler)).toStrictEqual([again])
  })
})
