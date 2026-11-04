/**
 * What the host answers about the registrations of its plugins: which menu items
 * a target is offered, which command or item an id runs, the editor a kind is
 * opened as, and a registration replaced under the same id.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import type { ContextMenuTarget } from '@numen/plugin-sdk'
import { createPluginHost } from './host'
import { disposePluginEntry } from './registry'
import type { PluginHostDeps } from './types'

const createDeps = (more: Partial<PluginHostDeps> = {}): PluginHostDeps => ({
  registerKind: vi.fn(() => vi.fn()),
  registerExtensionReader: vi.fn(() => vi.fn()),
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
    add('note', { target: 'note', order: 4, icon: 'note-icon' })
    add('either', { target: ['note', 'folder'], order: 7 })
    add('arrayAll', { target: ['all'], order: 8 })
    add('unknownTarget', { target: ['nonexistent' as never], order: 9 })
    return { host, handler }
  }
  const ids = (host: ReturnType<typeof createHost>['host'], one: ContextMenuTarget) =>
    host.getContextMenuItems(one).map((item) => item.id)

  it('are the items for the root, in their order', () => {
    const { host } = createHost()

    expect(ids(host, ROOT)).toStrictEqual(['any', 'root', 'folder', 'all', 'either', 'arrayAll'])
    expect(
      ids(host, target({ path: null as unknown as string, isFolder: true, isNote: false })),
    ).toStrictEqual(['any', 'root', 'folder', 'all', 'either', 'arrayAll'])
  })

  it('are the items for a folder, a note and a file of another kind', () => {
    const { host } = createHost()

    expect(ids(host, FOLDER)).toStrictEqual(['any', 'folder', 'all', 'either', 'arrayAll'])
    expect(ids(host, NOTE)).toStrictEqual(['any', 'file', 'note', 'all', 'either', 'arrayAll'])
    expect(ids(host, FILE)).toStrictEqual(['any', 'file', 'all', 'arrayAll'])
  })

  it('runs the item an id names, and nothing for an id no item has', async () => {
    const { host, handler } = createHost()

    expect(await host.runContextMenu('note', NOTE)).toBe(true)
    expect(handler).toHaveBeenCalledExactlyOnceWith(NOTE)
    expect(host.getContextMenuIcon('note')).toBe('note-icon')
    expect(await host.runContextMenu('missing', NOTE)).toBe(false)
    expect(host.getContextMenuIcon('missing')).toBeUndefined()
  })
})

describe('a command run by its id', () => {
  it('is found under its full id, and nothing else is', async () => {
    const host = createPluginHost(
      createDeps({
        getCommandDeclarations: () => [
          {
            pluginId: 'sample.plugin',
            id: 'count',
            title: 'Count',
            hasBackend: false,
            hasFrontend: true,
          },
        ],
      }),
    )
    const handler = vi.fn()
    host.createContext('sample.plugin').registerCommand({ id: 'count', handler })
    const note = { path: 'Notes/One.md', title: 'One', type: 'note' as const }

    expect(host.hasCommand('sample.plugin:count')).toBe(true)
    expect(host.hasCommand('count')).toBe(false)
    expect(host.hasCommand('xxxxxxxcount')).toBe(false)
    expect(host.hasCommand('other')).toBe(false)
    expect(await host.executeCommand('sample.plugin:count', note)).toBe(true)
    expect(await host.executeCommand('other', note)).toBe(false)
    expect(handler).toHaveBeenCalledOnce()
  })
})

describe('a registration replaced under the same id', () => {
  it('stays when the registration it replaced is disposed', () => {
    const host = createPluginHost(createDeps())
    const ctx = host.createContext('sample.plugin')
    const handler = vi.fn()
    const firstMenu = ctx.registerContextMenu({ id: 'menu', text: 'First', handler })
    ctx.registerContextMenu({ id: 'menu', text: 'Second', handler })
    const firstReader = ctx.registerReader({ id: 'pdf', title: 'First', component: null })
    ctx.registerReader({ id: 'pdf', title: 'Second', component: null })
    const firstTab = ctx.registerTabKind({ kind: 'custom', title: 'First', component: null })
    ctx.registerTabKind({ kind: 'custom', title: 'Second', component: null })
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

    firstMenu.dispose()
    firstReader.dispose()
    firstTab.dispose()
    firstAgent.dispose()

    expect(host.getRegisteredContextMenus().map((one) => one.text)).toStrictEqual(['Second'])
    expect(host.getRegisteredReaders().map((one) => one.title)).toStrictEqual(['Second'])
    expect(host.getRegisteredTabKinds().map((one) => one.title)).toStrictEqual(['Second'])
    secondAgent.dispose()
  })

  it('leaves the registrations of other plugins when one plugin is disposed', () => {
    const host = createPluginHost(createDeps())
    const handler = vi.fn()
    for (const pluginId of ['first.plugin', 'second.plugin']) {
      host.registerManifest(pluginId, {
        id: pluginId,
        name: pluginId,
        version: '1',
        permissions: ['vault:read'],
      } as never)
      const ctx = host.createContext(pluginId)
      ctx.registerCommand({ id: `${pluginId}.count`, handler })
      ctx.registerContextMenu({ id: `${pluginId}.menu`, text: pluginId, handler })
      ctx.registerAgentProvider({ id: `${pluginId}.agent`, name: pluginId, createAgent: vi.fn() })
      ctx.registerReader({ id: `${pluginId}.reader`, title: pluginId, component: null })
      ctx.registerTabKind({ kind: `${pluginId}.kind`, component: {} })
    }

    host.disposePlugin('first.plugin')
    expect(host.createContext('first.plugin').hasPermission?.('vault:read')).toBe(false)

    expect(host.getRegisteredCommands().map((one) => one.id)).toStrictEqual(['second.plugin.count'])
    expect(host.getRegisteredContextMenus().map((one) => one.text)).toStrictEqual(['second.plugin'])
    expect(host.getRegisteredReaders().map((one) => one.title)).toStrictEqual(['second.plugin'])
    expect(host.getRegisteredTabKinds().map((one) => one.kind)).toStrictEqual([
      'second.plugin.kind',
    ])
  })

  it('disposes orphan entries across all registries on disposeAll', () => {
    const host = createPluginHost(createDeps())
    host.contextHost.registerReader('orphan.reader', { id: 'r1', component: null })
    host.contextHost.registerTabKind('orphan.tab', { kind: 't1', component: null })
    host.contextHost.registerCommand('orphan.cmd', { id: 'c1', handler: vi.fn() })
    host.contextHost.registerContextMenu('orphan.menu', {
      id: 'm1',
      text: 'M1',
      handler: vi.fn(),
    })
    host.contextHost.registerSettingsView('orphan.settings', { id: 's1', component: {} })
    host.contextHost.registerStatusView('orphan.status', { id: 'st1', component: {} })

    expect(host.getRegisteredReaders()).toHaveLength(1)
    expect(host.getRegisteredTabKinds()).toHaveLength(1)
    expect(host.getRegisteredCommands()).toHaveLength(1)
    expect(host.getRegisteredContextMenus()).toHaveLength(1)
    expect(host.getRegisteredSettingsViews('orphan.settings').size).toBe(1)
    expect(host.statusViews.value).toHaveLength(1)

    host.disposeAll()

    expect(host.getRegisteredReaders()).toHaveLength(0)
    expect(host.getRegisteredTabKinds()).toHaveLength(0)
    expect(host.getRegisteredCommands()).toHaveLength(0)
    expect(host.getRegisteredContextMenus()).toHaveLength(0)
    expect(host.getRegisteredSettingsViews('orphan.settings').size).toBe(0)
    expect(host.statusViews.value).toHaveLength(0)

    expect(() =>
      disposePluginEntry('none', {
        registry: {
          vaultListeners: new Set(),
          fileListeners: new Set(),
          readers: new Map(),
          tabKinds: new Map(),
          commands: { value: new Map() },
          contextMenus: { value: new Map() },
          settingsViews: { value: new Map() },
          agentProviders: new Map(),
          contexts: new Map(),
          manifests: new Map(),
          activated: { value: new Set() },
          disposed: new Set(),
          reportedIssues: new Set(),
        },
        statusViews: {
          views: { value: [] } as never,
          registerStatusView: vi.fn(),
          disposePlugin: vi.fn(),
        },
      }),
    ).not.toThrow()
  })
})

describe('the settings views of a plugin', () => {
  it('are the registrations of that plugin, by view id', () => {
    const host = createPluginHost(createDeps())
    host.createContext('first.plugin').registerSettingsView({ id: 'floor', component: 'A' })
    host.createContext('second.plugin').registerSettingsView({ id: 'floor', component: 'B' })

    expect([...host.getRegisteredSettingsViews('first.plugin').keys()]).toStrictEqual(['floor'])
    expect(host.getRegisteredSettingsViews('first.plugin').get('floor')?.component).toBe('A')
    expect(host.getRegisteredSettingsViews('second.plugin').get('floor')?.component).toBe('B')
    expect(host.getRegisteredSettingsViews('missing.plugin').size).toBe(0)
  })

  it('re-evaluate a computed when a view is registered or disposed', () => {
    const host = createPluginHost(createDeps())
    const ctx = host.createContext('sample.plugin')
    const count = computed(() => host.getRegisteredSettingsViews('sample.plugin').size)

    expect(count.value).toBe(0)
    const disposable = ctx.registerSettingsView({ id: 'floor', component: {} })
    expect(count.value).toBe(1)
    disposable.dispose()
    expect(count.value).toBe(0)
  })

  it('stay when the registration they replaced is disposed', () => {
    const host = createPluginHost(createDeps())
    const ctx = host.createContext('sample.plugin')
    const first = ctx.registerSettingsView({ id: 'floor', component: 'First' })
    ctx.registerSettingsView({ id: 'floor', component: 'Second' })

    first.dispose()

    expect(host.getRegisteredSettingsViews('sample.plugin').get('floor')?.component).toBe('Second')
  })

  it('are removed with the plugin and with the host', () => {
    const host = createPluginHost(createDeps())
    host.createContext('first.plugin').registerSettingsView({ id: 'floor', component: {} })
    host.createContext('second.plugin').registerSettingsView({ id: 'floor', component: {} })

    host.disposePlugin('first.plugin')
    expect(host.getRegisteredSettingsViews('first.plugin').size).toBe(0)
    expect(host.getRegisteredSettingsViews('second.plugin').size).toBe(1)

    host.disposeAll()
    expect(host.getRegisteredSettingsViews('second.plugin').size).toBe(0)
  })
})

describe('the status views of the plugins', () => {
  it('list the registrations of every plugin in order, reactively', () => {
    const host = createPluginHost(createDeps())
    const count = computed(() => host.statusViews.value.length)

    expect(count.value).toBe(0)
    host.createContext('first.plugin').registerStatusView({ id: 'update', component: 'A' })
    host.createContext('second.plugin').registerStatusView({ id: 'update', component: 'B' })

    expect(count.value).toBe(2)
    expect(host.statusViews.value.map((one) => [one.pluginId, one.registration.component])).toEqual(
      [
        ['first.plugin', 'A'],
        ['second.plugin', 'B'],
      ],
    )
  })

  it('are removed with the plugin and with the host', () => {
    const host = createPluginHost(createDeps())
    host.createContext('first.plugin').registerStatusView({ id: 'update', component: {} })
    host.createContext('second.plugin').registerStatusView({ id: 'update', component: {} })

    host.disposePlugin('first.plugin')
    expect(host.statusViews.value.map((one) => one.pluginId)).toEqual(['second.plugin'])

    host.disposeAll()
    expect(host.statusViews.value).toHaveLength(0)
  })
})

describe('a tab kind opened as the editor of a path', () => {
  const registerKind = (deps: PluginHostDeps, unregReturn: (() => void) | undefined = vi.fn()) => {
    let open: (path: string, title?: string, options?: unknown) => void = vi.fn()
    const host = createPluginHost({
      ...deps,
      registerEditor: vi.fn((_kind, opener) => {
        open = opener
        return unregReturn as () => void
      }),
    })
    const sub = host
      .createContext('sample.plugin')
      .registerTabKind({ kind: 'sample', component: {} })
    return { open, sub }
  }

  it('calls openTabAdjacent for the where option as a string or an object and falls back to openTab otherwise', async () => {
    const onError = vi.fn()
    const unregSpy = vi.fn()
    const openTabAdjacent = vi.fn(async (_kind: string, path?: string) => {
      if (path === 'Samples/FailAdj.md') throw new Error('adj fail')
      return 'tab-2'
    })
    const openTab = vi.fn(async (_kind: string, path?: string) => {
      if (path === 'Samples/FailOpen.md') throw new Error('open fail')
      return 'tab-1'
    })
    const deps = createDeps({ openTabAdjacent, openTab, onError })
    const { open, sub } = registerKind(deps, unregSpy)

    open('Samples/One.md', 'One', 'split')
    open('Samples/Two.md', 'Two', { where: 'split' })
    open('Samples/Three.md', 'Three', { where: 'current' })
    open('Samples/FailAdj.md', 'FailAdj', 'split')
    open('Samples/FailOpen.md', 'FailOpen')

    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(openTabAdjacent.mock.calls).toStrictEqual([
      ['sample', 'Samples/One.md'],
      ['sample', 'Samples/Two.md'],
      ['sample', 'Samples/FailAdj.md'],
    ])
    expect(openTab.mock.calls).toStrictEqual([
      ['sample', 'Samples/Three.md'],
      ['sample', 'Samples/FailOpen.md'],
    ])
    expect(onError).toHaveBeenCalledTimes(2)

    sub.dispose()
    sub.dispose()
    expect(unregSpy).toHaveBeenCalledTimes(1)
  })

  it('opens the tab in the current pane for any other option and tolerates non-function unreg', () => {
    const deps = createDeps()
    const { open, sub } = registerKind(deps, undefined)

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
  it('sets a new selector on the state of the opened tab and skips a tab without state or selector', async () => {
    const readers = new Map<string, (path: string, selector: unknown) => void>()
    const states: Record<string, { selector?: unknown }> = { 'tab-1': {} }
    const onError = vi.fn()
    const unregExt = vi.fn()
    const getTabState = vi.fn((id: string) => states[id])
    const openTab = vi.fn(async (_kind: string, path?: string) => {
      if (path === 'Books/Fail.bin') throw new Error('reader fail')
      return path === 'Books/Gone.bin' ? 'tab-2' : 'tab-1'
    })
    const icon = { name: 'Icon' }
    const registerExtensionReader = vi.fn((ext: string, reader: unknown, _icon?: unknown) => {
      readers.set(ext, reader as (path: string, selector: unknown) => void)
      return unregExt
    })
    const host = createPluginHost(
      createDeps({
        openTab,
        onError,
        getTabState,
        getReaderExtensions: () => ['bin'],
        registerExtensionReader,
      }),
    )
    const sub1 = host
      .createContext('sample.plugin')
      .registerReader({ id: 'reader:bin', component: null, icon })
    const sub2 = host
      .createContext('sample.plugin')
      .registerReader({ id: 'unprefixed', component: null })
    const selector = { page: 3 }

    readers.get('bin')?.('Books/Unprefixed.bin', undefined)
    await vi.waitFor(() => expect(openTab).toHaveBeenCalledTimes(1))
    expect(openTab.mock.calls[0]).toStrictEqual(['reader:unprefixed', 'Books/Unprefixed.bin'])
    expect(getTabState).not.toHaveBeenCalled()

    sub2.dispose()
    sub2.dispose()
    expect(unregExt).toHaveBeenCalledTimes(1)

    // Re-register reader:bin to test selector setting and error handling
    host.createContext('sample.plugin').registerReader({ id: 'reader:bin', component: null, icon })
    readers.get('bin')?.('Books/One.bin', selector)
    readers.get('bin')?.('Books/Gone.bin', selector)
    readers.get('bin')?.('Books/Fail.bin', selector)
    await vi.waitFor(() => expect(openTab).toHaveBeenCalledTimes(4))
    await Promise.resolve()

    expect(openTab.mock.calls[1]).toStrictEqual(['reader:bin', 'Books/One.bin'])
    expect(states['tab-1']?.selector).toStrictEqual(selector)
    expect(states['tab-1']?.selector).not.toBe(selector)
    expect(registerExtensionReader.mock.calls[0]?.[2]).toBe(icon)
    expect(onError).toHaveBeenCalledWith('sample.plugin', expect.any(Error))
    sub1.dispose()

    // Reader without getTabState or getReaderExtensions
    let noStateReader: ((path: string, selector: unknown) => void) | undefined
    const hostNoState = createPluginHost(
      createDeps({
        openTab,
        getReaderExtensions: () => ['raw'],
        registerExtensionReader: (_ext, r) => {
          noStateReader = r as (path: string, selector: unknown) => void
          return vi.fn()
        },
      }),
    )
    hostNoState.createContext('p').registerReader({ id: 'raw', component: null })
    noStateReader?.('Books/Raw.raw', { page: 1 })
    await Promise.resolve()

    const hostNoExts = createPluginHost(createDeps())
    expect(() =>
      hostNoExts.createContext('p').registerReader({ id: 'none', component: null }),
    ).not.toThrow()
  })
})
