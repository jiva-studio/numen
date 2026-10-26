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
  kind: 'note',
  ...more,
})

const ROOT = target({ path: '', isFolder: true, kind: 'other' })
const FOLDER = target({ path: 'Notes', isFolder: true, kind: 'other' })
const NOTE = target()
const BOOK = target({ path: 'Books/One.pdf', kind: 'book' })

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
    add('book', { target: 'book', order: 6 })
    add('either', { target: ['book', 'folder'], order: 7 })
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
    expect(ids(host, NOTE)).toStrictEqual(['any', 'file', 'note', 'all'])
    expect(ids(host, BOOK)).toStrictEqual(['any', 'file', 'all', 'book', 'either'])
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
    const sub = host.createContext('sample.plugin').registerTabKind({ kind: 'deck', component: {} })
    return { open, sub }
  }

  it('calls openTabBeside for the where option as a string or an object', () => {
    const openTabBeside = vi.fn(async () => 'tab-2')
    const deps = createDeps({ openTabBeside })
    const { open } = registerKind(deps)

    open('Decks/One.md', 'One', 'beside')
    open('Decks/Two.md', 'Two', { where: 'beside' })

    expect(openTabBeside.mock.calls).toStrictEqual([
      ['deck', 'Decks/One.md'],
      ['deck', 'Decks/Two.md'],
    ])
    expect(deps.openTab).not.toHaveBeenCalled()
  })

  it('opens the tab in the current pane for any other option', () => {
    const deps = createDeps()
    const { open, sub } = registerKind(deps)

    open('Decks/One.md')
    open('Decks/Two.md', 'Two', { where: 'here' })
    open('Decks/Three.md', 'Three', null)
    open('Decks/Four.md', 'Four', 'beside')

    expect(vi.mocked(deps.openTab).mock.calls.map((call) => call[1])).toStrictEqual([
      'Decks/One.md',
      'Decks/Two.md',
      'Decks/Three.md',
      'Decks/Four.md',
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

describe('a reader opened on spans of a file', () => {
  it('passes the spans to the state of the opened tab and skips a tab without state', async () => {
    const readers = new Map<string, (path: string, spans?: readonly unknown[]) => void>()
    const focusSpans = vi.fn(async () => {})
    const states: Record<string, unknown> = { 'tab-1': { focusSpans } }
    const openTab = vi.fn(async (_kind: string, path?: string) =>
      path === 'Books/Gone.pdf' ? 'tab-2' : 'tab-1',
    )
    const host = createPluginHost(
      createDeps({
        openTab,
        getTabState: (id) => states[id],
        registerExtensionReader: vi.fn((ext, reader) => {
          readers.set(ext, reader as (path: string, spans?: readonly unknown[]) => void)
          return () => {}
        }),
      }),
    )
    host.createContext('sample.plugin').registerReader({
      id: 'reader:pdf',
      fileExtensions: ['pdf'],
      component: null,
      source: 'book',
    })
    const span = { from: 1, to: 4 }

    readers.get('pdf')?.('Books/One.pdf', [span])
    readers.get('pdf')?.('Books/Gone.pdf', [span])
    readers.get('pdf')?.('Books/One.pdf', [])
    await vi.waitFor(() => expect(openTab).toHaveBeenCalledTimes(3))
    await Promise.resolve()

    expect(openTab.mock.calls[0]).toStrictEqual(['reader:pdf', 'Books/One.pdf'])
    expect(focusSpans).toHaveBeenCalledExactlyOnceWith(span)
    expect((states['tab-1'] as { spans?: unknown }).spans).toStrictEqual([span])
  })
})
