/**
 * What the host answers about the registrations of its plugins: which command an id runs, the
 * editor a kind is opened as, and a registration replaced under the same id.
 */
import { describe, expect, it, vi } from 'vitest'
import { computed } from 'vue'
import { createMockManifest } from '@numen/core-client/testing'
import { createPluginHost } from './host'
import { disposePluginEntry } from './registry'
import type { PluginHostDeps } from './types'
import { STUB_COMPONENT } from '../testing/stubComponent'

const createDeps = (more: Partial<PluginHostDeps> = {}): PluginHostDeps => ({
  registerKind: vi.fn(() => vi.fn()),
  registerExtensionReader: vi.fn(() => vi.fn()),
  openTab: vi.fn(async () => 'tab-1'),
  ...more,
})

const declaring = (pluginId: string) =>
  createMockManifest(pluginId, {
    permissions: ['vault:read'],
    commands: ['count', `${pluginId}.count`],
    readers: ['pdf', 'reader:bin', 'unprefixed', 'raw', 'none', `${pluginId}.reader`],
    tabKinds: ['custom', 'sample', `${pluginId}.kind`],
    settingsViews: ['floor'],
  })

const COMPONENT_A = { name: 'A' }
const COMPONENT_B = { name: 'B' }

describe('a command run by its id', () => {
  it('is found under its full id, and nothing else is', async () => {
    const host = createPluginHost(
      createDeps({
        getCommandDeclarations: () => [
          {
            pluginId: 'sample.plugin',
            id: 'count',
            title: 'Count',
          },
        ],
      }),
    )
    const handler = vi.fn()
    host.lifecycle
      .createContext('sample.plugin', declaring('sample.plugin'))
      .register.command({ id: 'count', handler })
    const note = {
      path: 'Notes/One.md',
      title: 'One',
      type: 'note' as const,
      isFolder: false,
      others: [],
      tab: '',
    }

    expect(host.commands.hasCommand('sample.plugin:count')).toBe(true)
    expect(host.commands.hasCommand('count')).toBe(false)
    expect(host.commands.hasCommand('xxxxxxxcount')).toBe(false)
    expect(host.commands.hasCommand('other')).toBe(false)
    expect(await host.commands.executeCommand('sample.plugin:count', note)).toBe(true)
    expect(await host.commands.executeCommand('other', note)).toBe(false)
    expect(handler).toHaveBeenCalledOnce()
  })
})

describe('registrations of a plugin', () => {
  it('leaves the registrations of other plugins when one plugin is disposed', () => {
    const host = createPluginHost(createDeps())
    const handler = vi.fn()
    for (const pluginId of ['first.plugin', 'second.plugin']) {
      host.lifecycle.registerManifest(pluginId, declaring(pluginId))
      const ctx = host.lifecycle.createContext(pluginId)
      ctx.register.command({ id: `${pluginId}.count`, handler })
      ctx.register.reader({ id: `${pluginId}.reader`, title: pluginId, component: STUB_COMPONENT })
      ctx.register.tabKind({ kind: `${pluginId}.kind`, component: {} })
    }

    host.lifecycle.disposePlugin('first.plugin')
    expect(host.lifecycle.createContext('first.plugin').plugin.hasPermission?.('vault:read')).toBe(
      false,
    )

    expect(host.registered.getRegisteredCommands().map((one) => one.id)).toStrictEqual([
      'second.plugin.count',
    ])
    expect(host.registered.getRegisteredReaders().map((one) => one.title)).toStrictEqual([
      'second.plugin',
    ])
    expect(host.registered.getRegisteredTabKinds().map((one) => one.kind)).toStrictEqual([
      'second.plugin.kind',
    ])
  })

  it('disposes orphan entries across all registries on disposeAll', () => {
    const host = createPluginHost(createDeps())
    host.contextHost.registration.registerReader('orphan.reader', {
      id: 'r1',
      component: STUB_COMPONENT,
    })
    host.contextHost.registration.registerTabKind('orphan.tab', {
      kind: 't1',
      component: STUB_COMPONENT,
    })
    host.contextHost.registration.registerCommand('orphan.cmd', { id: 'c1', handler: vi.fn() })
    host.contextHost.registration.registerSettingsView('orphan.settings', {
      id: 's1',
      component: {},
    })

    expect(host.registered.getRegisteredReaders()).toHaveLength(1)
    expect(host.registered.getRegisteredTabKinds()).toHaveLength(1)
    expect(host.registered.getRegisteredCommands()).toHaveLength(1)
    expect(host.registered.getRegisteredSettingsViews('orphan.settings').size).toBe(1)

    host.lifecycle.disposeAll()

    expect(host.registered.getRegisteredReaders()).toHaveLength(0)
    expect(host.registered.getRegisteredTabKinds()).toHaveLength(0)
    expect(host.registered.getRegisteredCommands()).toHaveLength(0)
    expect(host.registered.getRegisteredSettingsViews('orphan.settings').size).toBe(0)

    expect(() =>
      disposePluginEntry('none', {
        registry: {
          vaultListeners: new Set(),
          fileListeners: new Set(),
          readers: new Map(),
          tabKinds: new Map(),
          commands: { value: new Map() },
          handlers: { value: new Map() },
          settingsViews: { value: new Map() },
          contexts: new Map(),
          manifests: new Map(),
          activated: { value: new Set() },
          disposed: new Set(),
          reportedIssues: new Set(),
        },
      }),
    ).not.toThrow()
  })
})

describe('the settings views of a plugin', () => {
  it('are the registrations of that plugin, by view id', () => {
    const host = createPluginHost(createDeps())
    host.lifecycle
      .createContext('first.plugin', declaring('first.plugin'))
      .register.settingsView({ id: 'floor', component: COMPONENT_A })
    host.lifecycle
      .createContext('second.plugin', declaring('second.plugin'))
      .register.settingsView({ id: 'floor', component: COMPONENT_B })

    expect([...host.registered.getRegisteredSettingsViews('first.plugin').keys()]).toStrictEqual([
      'floor',
    ])
    expect(host.registered.getRegisteredSettingsViews('first.plugin').get('floor')?.component).toBe(
      COMPONENT_A,
    )
    expect(
      host.registered.getRegisteredSettingsViews('second.plugin').get('floor')?.component,
    ).toBe(COMPONENT_B)
    expect(host.registered.getRegisteredSettingsViews('missing.plugin').size).toBe(0)
  })

  it('re-evaluate a computed when a view is registered or disposed', () => {
    const host = createPluginHost(createDeps())
    const ctx = host.lifecycle.createContext('sample.plugin', declaring('sample.plugin'))
    const count = computed(() => host.registered.getRegisteredSettingsViews('sample.plugin').size)

    expect(count.value).toBe(0)
    const disposable = ctx.register.settingsView({ id: 'floor', component: {} })
    expect(count.value).toBe(1)
    disposable.dispose()
    expect(count.value).toBe(0)
  })

  it('stay when the registration they replaced is disposed', () => {
    const host = createPluginHost(createDeps())
    const ctx = host.lifecycle.createContext('sample.plugin', declaring('sample.plugin'))
    const first = ctx.register.settingsView({ id: 'floor', component: COMPONENT_A })
    ctx.register.settingsView({ id: 'floor', component: COMPONENT_B })

    first.dispose()

    expect(
      host.registered.getRegisteredSettingsViews('sample.plugin').get('floor')?.component,
    ).toBe(COMPONENT_B)
  })

  it('are removed with the plugin and with the host', () => {
    const host = createPluginHost(createDeps())
    host.lifecycle
      .createContext('first.plugin', declaring('first.plugin'))
      .register.settingsView({ id: 'floor', component: {} })
    host.lifecycle
      .createContext('second.plugin', declaring('second.plugin'))
      .register.settingsView({ id: 'floor', component: {} })

    host.lifecycle.disposePlugin('first.plugin')
    expect(host.registered.getRegisteredSettingsViews('first.plugin').size).toBe(0)
    expect(host.registered.getRegisteredSettingsViews('second.plugin').size).toBe(1)

    host.lifecycle.disposeAll()
    expect(host.registered.getRegisteredSettingsViews('second.plugin').size).toBe(0)
  })
})

describe('the views the plugins place in a slot of a bar', () => {
  const manifest = (view: string) =>
    ({
      contributes: { slots: [{ slot: 'statusBar.right', placed: { case: 'view', value: view } }] },
    }) as unknown as Parameters<
      ReturnType<typeof createPluginHost>['lifecycle']['createContext']
    >[1]
  const entries = [
    { pluginId: 'second.plugin', slot: 'statusBar.right', view: 'update' },
    { pluginId: 'first.plugin', slot: 'statusBar.right', view: 'update' },
  ]

  it('are the entries of the slot once registered, in the order of the plugins, reactively', () => {
    const host = createPluginHost(createDeps({ getSlotEntries: () => entries }))
    const viewKeys = computed(() =>
      host.slots.getEntries('statusBar.right').map((one) => (one.kind === 'view' ? one.key : '')),
    )

    expect(viewKeys.value).toStrictEqual([])
    host.lifecycle
      .createContext('second.plugin', manifest('update'))
      .register.slotView({ id: 'update', component: COMPONENT_B })
    host.lifecycle
      .createContext('first.plugin', manifest('update'))
      .register.slotView({ id: 'update', component: COMPONENT_A })

    expect(viewKeys.value).toStrictEqual(['first.plugin/view/update', 'second.plugin/view/update'])
  })

  it('are removed with the plugin and with the host', () => {
    const host = createPluginHost(createDeps({ getSlotEntries: () => entries }))
    host.lifecycle
      .createContext('first.plugin', manifest('update'))
      .register.slotView({ id: 'update', component: {} })
    host.lifecycle
      .createContext('second.plugin', manifest('update'))
      .register.slotView({ id: 'update', component: {} })

    host.lifecycle.disposePlugin('first.plugin')
    expect(
      host.slots.getEntries('statusBar.right').map((one) => one.kind === 'view' && one.pluginId),
    ).toStrictEqual(['second.plugin'])

    host.lifecycle.disposeAll()
    expect(host.slots.getEntries('statusBar.right')).toStrictEqual([])
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
    const sub = host.lifecycle
      .createContext('sample.plugin', declaring('sample.plugin'))
      .register.tabKind({ kind: 'sample', component: {} })
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
    const openTab = vi.fn(async (_kind: string, _at?: string) => 'tab-1')
    const { open, sub } = registerKind(createDeps({ openTab }), undefined)

    open('Samples/One.md')
    open('Samples/Two.md', 'Two', { where: 'current' })
    open('Samples/Three.md', 'Three', null)
    open('Samples/Four.md', 'Four', 'split')

    expect(openTab.mock.calls.map((call) => call[1])).toStrictEqual([
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
    const ctx = host.lifecycle.createContext('sample.plugin')

    expect(ctx.settings.get('fontSize')).toBeUndefined()
    expect(() => ctx.settings.onChange('fontSize', vi.fn()).dispose()).not.toThrow()
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
    const sub1 = host.lifecycle
      .createContext('sample.plugin', declaring('sample.plugin'))
      .register.reader({ id: 'reader:bin', component: STUB_COMPONENT, icon })
    const sub2 = host.lifecycle
      .createContext('sample.plugin', declaring('sample.plugin'))
      .register.reader({ id: 'unprefixed', component: STUB_COMPONENT })
    const selector = { page: 3 }

    readers.get('bin')?.('Books/Unprefixed.bin', undefined)
    await vi.waitFor(() => expect(openTab).toHaveBeenCalledTimes(1))
    expect(openTab.mock.calls[0]).toStrictEqual(['reader:unprefixed', 'Books/Unprefixed.bin'])
    expect(getTabState).not.toHaveBeenCalled()

    sub2.dispose()
    sub2.dispose()
    expect(unregExt).toHaveBeenCalledTimes(1)

    // Re-register reader:bin to test selector setting and error handling
    sub1.dispose()
    host.lifecycle
      .createContext('sample.plugin', declaring('sample.plugin'))
      .register.reader({ id: 'reader:bin', component: STUB_COMPONENT, icon })
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
    hostNoState.lifecycle
      .createContext('p', declaring('p'))
      .register.reader({ id: 'raw', component: STUB_COMPONENT })
    noStateReader?.('Books/Raw.raw', { page: 1 })
    await Promise.resolve()

    const hostNoExts = createPluginHost(createDeps())
    expect(() =>
      hostNoExts.lifecycle
        .createContext('p', declaring('p'))
        .register.reader({ id: 'none', component: STUB_COMPONENT }),
    ).not.toThrow()
  })
})

describe('a registration that fails after its tab kind was registered', () => {
  it('leaves no tab kind of a tab kind whose editor registration throws', () => {
    const unregisterKind = vi.fn()
    const host = createPluginHost(
      createDeps({
        registerKind: vi.fn(() => unregisterKind),
        registerEditor: () => {
          throw new Error('editor refused')
        },
      }),
    )
    const context = host.lifecycle.createContext('sample.plugin', declaring('sample.plugin'))

    expect(() => context.register.tabKind({ kind: 'sample', component: {} })).toThrowError(
      'editor refused',
    )
    expect(unregisterKind).toHaveBeenCalledTimes(1)
    expect(host.registered.getRegisteredTabKinds()).toHaveLength(0)
  })

  it('leaves no tab kind of a reader whose extension registration throws', () => {
    const unregisterKind = vi.fn()
    const host = createPluginHost(
      createDeps({
        registerKind: vi.fn(() => unregisterKind),
        getReaderExtensions: () => ['bin'],
        registerExtensionReader: () => {
          throw new Error('extension refused')
        },
      }),
    )
    const context = host.lifecycle.createContext('sample.plugin', declaring('sample.plugin'))

    expect(() => context.register.reader({ id: 'raw', component: STUB_COMPONENT })).toThrowError(
      'extension refused',
    )
    expect(unregisterKind).toHaveBeenCalledTimes(1)
    expect(host.registered.getRegisteredReaders()).toHaveLength(0)
  })
})
