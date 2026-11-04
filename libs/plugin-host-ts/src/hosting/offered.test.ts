import { describe, expect, it, vi } from 'vitest'
import { createPluginHost } from './host'
import type { CommandDeclaration, PluginHostDeps } from './types'

const declaration: CommandDeclaration = {
  pluginId: 'md.tree',
  id: 'tree.open',
  title: 'Open file tree',
  hasBackend: false,
  hasFrontend: true,
}

function createHost(extra: Partial<PluginHostDeps> = {}) {
  return createPluginHost({
    registerKind: vi.fn(() => vi.fn()),
    registerExtensionReader: vi.fn(() => vi.fn()),
    openTab: vi.fn(async () => 'tab-1'),
    getCommandDeclarations: () => [declaration],
    ...extra,
  })
}

describe('the commands of a plugin that failed to load', () => {
  it('are not offered while the plugin has not made a context', () => {
    const host = createHost({ isPluginFailed: () => true })

    expect(host.getPaletteCommands()).toStrictEqual([])
  })

  it('are offered while the plugin has not failed and has not made a context', () => {
    const host = createHost({ isPluginFailed: () => false })

    expect(host.getPaletteCommands().map((one) => one.id)).toStrictEqual(['md.tree:tree.open'])
  })
})

describe('a declared command whose handler is missing', () => {
  it('is reported once however often the palette is read', () => {
    const reportCommandIssue = vi.fn()
    const host = createHost({ reportCommandIssue })
    host.createContext('md.tree')

    host.getPaletteCommands()
    host.getPaletteCommands()
    host.hasCommand('md.tree:tree.open')

    expect(reportCommandIssue).toHaveBeenCalledTimes(1)
  })

  it('is not reported again for a plugin that is activated a second time', () => {
    const reportCommandIssue = vi.fn()
    const host = createHost({ reportCommandIssue })
    host.createContext('md.tree')
    host.getPaletteCommands()
    host.disposePlugin('md.tree')
    host.createContext('md.tree')

    host.getPaletteCommands()

    expect(reportCommandIssue).toHaveBeenCalledTimes(1)
  })
})

describe('the commands of a plugin that has a backend', () => {
  it('are left to the window while the frontend of the plugin is still loading', () => {
    const host = createHost({
      getCommandDeclarations: () => [{ ...declaration, hasBackend: true }],
    })

    expect(host.getPaletteCommands()).toStrictEqual([])
  })
})

describe('a handler registered for a command no manifest declares', () => {
  it('is reported once the first list of declarations has loaded, even for a plugin that declares nothing', () => {
    const reportCommandIssue = vi.fn()
    const host = createHost({ getCommandDeclarations: () => [], reportCommandIssue })
    host.createContext('md.silent').registerCommand({ id: 'tree.open', handler: vi.fn() })

    host.getPaletteCommands()

    expect(reportCommandIssue).toHaveBeenCalledWith(
      'md.silent',
      expect.stringContaining('tree.open'),
    )
  })

  it('is not reported before any list of declarations has loaded', () => {
    const reportCommandIssue = vi.fn()
    const host = createHost({ getCommandDeclarations: () => undefined, reportCommandIssue })
    host.createContext('md.silent').registerCommand({ id: 'tree.open', handler: vi.fn() })

    expect(host.getPaletteCommands()).toStrictEqual([])
    expect(reportCommandIssue).not.toHaveBeenCalled()
  })
})

describe('a command declared by a plugin with neither a frontend nor a backend', () => {
  it('is never offered', () => {
    const host = createHost({
      getCommandDeclarations: () => [{ ...declaration, hasFrontend: false, hasBackend: false }],
    })

    expect(host.getPaletteCommands()).toStrictEqual([])
    host.createContext('md.tree')
    expect(host.getPaletteCommands()).toStrictEqual([])
  })
})

describe('a plugin that was disposed', () => {
  it('offers none of its declared commands until it has a context again', () => {
    const host = createHost()
    host.createContext('md.tree').registerCommand({ id: 'tree.open', handler: vi.fn() })
    expect(host.getPaletteCommands().map((one) => one.id)).toStrictEqual(['md.tree:tree.open'])

    host.disposePlugin('md.tree')

    expect(host.getPaletteCommands()).toStrictEqual([])

    host.createContext('md.tree').registerCommand({ id: 'tree.open', handler: vi.fn() })

    expect(host.getPaletteCommands().map((one) => one.id)).toStrictEqual(['md.tree:tree.open'])
  })
})
