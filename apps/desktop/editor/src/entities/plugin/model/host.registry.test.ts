/**
 * What the host answers about the registrations of its plugins: which menu items
 * a target is offered, which command or item an id runs, the editor a kind is
 * opened as, and a registration replaced under the same id.
 */
import { describe, expect, it, vi } from 'vitest'
import type { ContextMenuTarget } from '@numen/plugin-sdk'
import { createPluginHost } from './host'
import type { PluginHostDeps } from './types'

const createDeps = (more: Partial<PluginHostDeps> = {}): PluginHostDeps => ({
  registerKind: vi.fn(() => () => {}),
  registerExtensionReader: vi.fn(() => () => {}),
  openTab: vi.fn(async () => 'tab-1'),
  ...more,
})

const target = (more: Partial<ContextMenuTarget> = {}): ContextMenuTarget => ({
  path: 'Notes/One.md',
  isFolder: false,
  isNote: true,
  ...more,
})

const ROOT = target({ path: '', isFolder: true, isNote: false })
const FOLDER = target({ path: 'Notes', isFolder: true, isNote: false })
const NOTE = target()
const FILE = target({ path: 'Books/One.bin', isNote: false })

describe('the menu items a target is offered', () => {
  const createHost = () => {
    const host = createPluginHost(createDeps())
    const ctx = host.createContext('sample.plugin')
    const handler = vi.fn()
    const add = (id: string, more: object = {}) =>
      ctx.registerContextMenu({ id, text: id, handler, ...more })
    add('any')
    add('all', { target: 'all', order: 5 })
    add('root', { target: 'root', order: 1 })
    add('folder', { target: 'folder', order: 2 })
    add('file', { target: 'file', order: 3 })
    add('note', { target: 'note', order: 4 })
    add('either', { target: ['note', 'folder'], order: 7 })
    return { host, handler }
  }
  const ids = (host: ReturnType<typeof createHost>['host'], one: ContextMenuTarget) =>
    host.getContextMenuItems(one).map((item) => item.id)

  it('are the items for the root, in their order', () => {
    const { host } = createHost()

    expect(ids(host, ROOT)).toStrictEqual(['any', 'root', 'folder', 'all', 'either'])
  })

  it('are the items for a folder, a note and a file of another kind', () => {
    const { host } = createHost()

    expect(ids(host, FOLDER)).toStrictEqual(['any', 'folder', 'all', 'either'])
    expect(ids(host, NOTE)).toStrictEqual(['any', 'file', 'note', 'all', 'either'])
    expect(ids(host, FILE)).toStrictEqual(['any', 'file', 'all'])
  })

  it('runs the item an id names, and nothing for an id no item has', async () => {
    const { host, handler } = createHost()

    expect(await host.runContextMenu('note', NOTE)).toBe(true)
    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE)
    expect(await host.runContextMenu('missing', NOTE)).toBe(false)
    expect(host.getContextMenuIcon('missing')).toBeUndefined()
  })
})

describe('a command run by its id', () => {
  it('is found under its own id and under the plugin prefix, and nothing else is', async () => {
    const host = createPluginHost(createDeps())
    const handler = vi.fn()
    host.createContext('sample.plugin').registerCommand({ id: 'count', title: 'Count', handler })
    const note = { path: 'Notes/One.md', title: 'One', type: 'note' as const }

    expect(host.hasCommand('count')).toBe(true)
    expect(host.hasCommand('plugin:count')).toBe(true)
    expect(host.hasCommand('other')).toBe(false)
    expect(await host.executeCommand('plugin:count', note)).toBe(true)
    expect(await host.executeCommand('other', note)).toBe(false)
    expect(handler).toHaveBeenCalledOnce()
  })
})

describe('a registration replaced under the same id', () => {
  it('stays when the registration it replaced is disposed', () => {
    const host = createPluginHost(createDeps())
    const ctx = host.createContext('sample.plugin')
    const handler = vi.fn()
    const firstCommand = ctx.registerCommand({ id: 'count', title: 'First', handler })
    ctx.registerCommand({ id: 'count', title: 'Second', handler })
    const firstMenu = ctx.registerContextMenu({ id: 'menu', text: 'First', handler })
    ctx.registerContextMenu({ id: 'menu', text: 'Second', handler })
    const firstAgent = ctx.registerAgentProvider({
      id: 'agent',
      name: 'First',
      createAgent: vi.fn(),
    })
    const secondAgent = ctx.registerAgentProvider({
      id: 'agent',
      name: 'Second',
      createAgent: vi.fn(),
    })

    firstCommand.dispose()
    firstMenu.dispose()
    firstAgent.dispose()

    expect(host.getRegisteredCommands().map((one) => one.title)).toStrictEqual(['Second'])
    expect(host.getRegisteredContextMenus().map((one) => one.text)).toStrictEqual(['Second'])
    secondAgent.dispose()
  })

  it('leaves the registrations of other plugins when one plugin is disposed', () => {
    const host = createPluginHost(createDeps())
    const handler = vi.fn()
    for (const pluginId of ['first.plugin', 'second.plugin']) {
      const ctx = host.createContext(pluginId)
      ctx.registerCommand({ id: `${pluginId}.count`, title: pluginId, handler })
      ctx.registerContextMenu({ id: `${pluginId}.menu`, text: pluginId, handler })
      ctx.registerAgentProvider({ id: `${pluginId}.agent`, name: pluginId, createAgent: vi.fn() })
      ctx.registerTabKind({ kind: `${pluginId}.kind`, component: {} })
    }

    host.disposePlugin('first.plugin')

    expect(host.getRegisteredCommands().map((one) => one.title)).toStrictEqual(['second.plugin'])
    expect(host.getRegisteredContextMenus().map((one) => one.text)).toStrictEqual(['second.plugin'])
    expect(host.getRegisteredTabKinds().map((one) => one.kind)).toStrictEqual([
      'second.plugin.kind',
    ])
  })
})

describe('a tab kind opened as the editor of a path', () => {
  const registerKind = (deps: PluginHostDeps) => {
    let open: (path: string, title?: string, options?: unknown) => void = () => {}
    const host = createPluginHost({
      ...deps,
      registerEditor: vi.fn((_kind, opener) => {
        open = opener
        return () => {}
      }),
    })
    const sub = host
      .createContext('sample.plugin')
      .registerTabKind({ kind: 'sample', component: {} })
    return { open, sub }
  }

  it('calls openTabAdjacent for the where option as a string or an object', () => {
    const openTabAdjacent = vi.fn(async () => 'tab-2')
    const deps = createDeps({ openTabAdjacent })
    const { open } = registerKind(deps)

    open('Samples/One.md', 'One', 'split')
    open('Samples/Two.md', 'Two', { where: 'split' })

    expect(openTabAdjacent.mock.calls).toStrictEqual([
      ['sample', 'Samples/One.md'],
      ['sample', 'Samples/Two.md'],
    ])
    expect(deps.openTab).not.toHaveBeenCalled()
  })

  it('opens the tab in the current pane for any other option', () => {
    const deps = createDeps()
    const { open, sub } = registerKind(deps)

    open('Samples/One.md')
    open('Samples/Two.md', 'Two', { where: 'current' })
    open('Samples/Three.md', 'Three', null)
    open('Samples/Four.md', 'Four', 'split')

    expect(vi.mocked(deps.openTab).mock.calls.map((call) => call[1])).toStrictEqual([
      'Samples/One.md',
      'Samples/Two.md',
      'Samples/Three.md',
      'Samples/Four.md',
    ])
    sub.dispose()
    sub.dispose()
  })
})

describe('the settings of a window that keeps none', () => {
  it('answers nothing and follows nothing', () => {
    const host = createPluginHost(createDeps())
    const ctx = host.createContext('sample.plugin')

    expect(ctx.getSetting('fontSize')).toBeUndefined()
    expect(() => ctx.onSettingChange('fontSize', vi.fn()).dispose()).not.toThrow()
  })
})

describe('a reader opened at a selector', () => {
  it('sets a new selector on the state of the opened tab and skips a tab without state', async () => {
    const readers = new Map<string, (path: string, selector: unknown) => void>()
    const states: Record<string, { selector?: unknown }> = { 'tab-1': {} }
    const openTab = vi.fn(async (_kind: string, path?: string) =>
      path === 'Books/Gone.bin' ? 'tab-2' : 'tab-1',
    )
    const icon = { name: 'Icon' }
    const registerExtensionReader = vi.fn((ext: string, reader: unknown, _icon?: unknown) => {
      readers.set(ext, reader as (path: string, selector: unknown) => void)
      return () => {}
    })
    const host = createPluginHost(
      createDeps({
        openTab,
        getTabState: (id) => states[id],
        getReaderExtensions: () => ['bin'],
        registerExtensionReader,
      }),
    )
    host.createContext('sample.plugin').registerReader({ id: 'reader:bin', component: null, icon })
    const selector = { page: 3 }

    readers.get('bin')?.('Books/One.bin', selector)
    readers.get('bin')?.('Books/Gone.bin', selector)
    await vi.waitFor(() => expect(openTab).toHaveBeenCalledTimes(2))
    await Promise.resolve()

    expect(openTab.mock.calls[0]).toStrictEqual(['reader:bin', 'Books/One.bin'])
    expect(states['tab-1']?.selector).toStrictEqual(selector)
    expect(states['tab-1']?.selector).not.toBe(selector)
    expect(registerExtensionReader.mock.calls[0]?.[2]).toBe(icon)
  })
})
