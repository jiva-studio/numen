import { describe, expect, it, vi } from 'vitest'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost } from './host'
import type { CommandDeclaration, PluginHostDeps } from './types'

const declaration: CommandDeclaration = {
  pluginId: 'md.tree',
  id: 'tree.open',
  title: 'Open file tree',
}

const TREE_MANIFEST = createMockManifest('md.tree', { commands: ['tree.open'] })

function createHost(extra: Partial<PluginHostDeps> = {}) {
  return createPluginHost({
    target: 'desktop.editor',
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => [declaration],
    ...extra,
  })
}

describe('the commands of a plugin that failed to load', () => {
  it('are not listed while the plugin has not made a context', () => {
    const host = createHost({ isPluginFailed: () => true })

    expect(host.commands.getPaletteCommands()).toStrictEqual([])
  })

  it('are listed while the plugin has not failed and has not made a context', () => {
    const host = createHost({ isPluginFailed: () => false })

    expect(host.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.tree:tree.open',
    ])
  })
})

describe('a declared command whose handler is missing', () => {
  it('is reported once however often the palette is read', () => {
    const reportCommandIssue = vi.fn()
    const host = createHost({ reportCommandIssue })
    host.lifecycle.createContext('md.tree')

    host.commands.getPaletteCommands()
    host.commands.getPaletteCommands()
    host.commands.hasCommand('md.tree:tree.open')

    expect(reportCommandIssue).toHaveBeenCalledTimes(1)
  })

  it('is not reported again for a plugin that is activated a second time', () => {
    const reportCommandIssue = vi.fn()
    const host = createHost({ reportCommandIssue })
    host.lifecycle.createContext('md.tree')
    host.commands.getPaletteCommands()
    host.lifecycle.disposePlugin('md.tree')
    host.lifecycle.createContext('md.tree')

    host.commands.getPaletteCommands()

    expect(reportCommandIssue).toHaveBeenCalledTimes(1)
  })
})

describe('the commands of a plugin that has no frontend', () => {
  it('are left to the window, which runs them in the backend of the plugin', () => {
    const host = createHost({ hasPluginFrontend: () => false })

    expect(host.commands.getPaletteCommands()).toStrictEqual([])
  })
})

describe('a command declared by a plugin with no frontend to load', () => {
  it('is not listed by the host, before or after the plugin has a context', () => {
    const host = createHost({ hasPluginFrontend: () => false })

    expect(host.commands.getPaletteCommands()).toStrictEqual([])
    host.lifecycle.createContext('md.tree')
    expect(host.commands.getPaletteCommands()).toStrictEqual([])
  })
})

describe('a plugin that was disposed', () => {
  it('lists none of its declared commands until it has a context again', () => {
    const host = createHost()
    host.lifecycle
      .createContext('md.tree', TREE_MANIFEST)
      .plugin.register.command({ id: 'tree.open', handler: vi.fn() })
    expect(host.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.tree:tree.open',
    ])

    host.lifecycle.disposePlugin('md.tree')

    expect(host.commands.getPaletteCommands()).toStrictEqual([])

    host.lifecycle
      .createContext('md.tree', TREE_MANIFEST)
      .plugin.register.command({ id: 'tree.open', handler: vi.fn() })

    expect(host.commands.getPaletteCommands().map((one) => one.id)).toStrictEqual([
      'md.tree:tree.open',
    ])
  })
})
