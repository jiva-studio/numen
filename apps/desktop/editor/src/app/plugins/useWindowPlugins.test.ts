import { describe, expect, it, vi } from 'vitest'
import { nextTick, ref } from 'vue'
import { DEFAULT_PLUGIN_ENABLED } from '@numen/plugin-host'
import { createNoticeQueue } from '@numen/shell'
import { useWindowTabs } from '@/entities/tab'
import type { AnyTabKind } from '@/entities/tab'
import type { AgentHost, PluginContext, PluginManifest, WorkspaceHost } from '@numen/plugin-sdk'
import { createPluginSupport, getFrontendPluginIds } from './useWindowPlugins'

const createSupport = () => {
  const windowTabs = useWindowTabs()
  const removeClosedTab = vi.fn()
  const support = createPluginSupport({
    windowTabs,
    tabLinks: { removeClosedTab } as never,
    tabOpeners: {
      registerExtensionReader: vi.fn(() => () => undefined),
      registerEditor: vi.fn(() => () => undefined),
    } as never,
    settingsHolder: {},
    services: {
      workspace: { getNoteInFocus: () => ({ path: '', title: '' }) } as unknown as WorkspaceHost,
      agent: { finish: () => Promise.resolve(), getModelsAt: () => [] } as unknown as AgentHost,
    },
  })
  return { windowTabs, support, removeClosedTab }
}

const coreKind = (state: object): AnyTabKind => ({
  kind: 'core',
  open: () => state,
  pane: {},
})

describe('closing the tabs of a plugin', () => {
  it('replaces the tabs of the kinds the plugin registered with placeholders', async () => {
    const { windowTabs, support } = createSupport()
    support.pluginHost
      .createContext('md.numen.a')
      .registerTabKind({ kind: 'a-kind', component: null, title: 'A' })
    const id = await windowTabs.createTab('a-kind', 'x')
    windowTabs.show(id)

    support.pluginHost.disposePlugin('md.numen.a')

    expect(windowTabs.getTab(id)?.state).toEqual({
      kind: 'a-kind',
      targetPath: 'x',
      saved: undefined,
      title: 'x',
    })
  })

  it('keeps a tab of a core kind whose state names the plugin', async () => {
    const { windowTabs, support } = createSupport()
    windowTabs.registerKind(coreKind({ pluginId: 'md.numen.a' }))
    const id = await windowTabs.createTab('core', 'x')

    support.pluginHost.disposePlugin('md.numen.a')

    expect(windowTabs.getTab(id)).not.toBeNull()
  })

  it('keeps the tabs of another plugin', async () => {
    const { windowTabs, support } = createSupport()
    support.pluginHost
      .createContext('md.numen.b')
      .registerTabKind({ kind: 'b-kind', component: null, title: 'B' })
    const id = await windowTabs.createTab('b-kind', 'x')

    support.pluginHost.disposePlugin('md.numen.a')

    expect(windowTabs.getTab(id)).not.toBeNull()
  })
})

/** A plugin that adds one tab kind to the window. */
const manifest: PluginManifest = {
  id: 'md.numen.sample.chat',
  name: 'Chat',
  version: '1.0.0',
}

const activate = (context: PluginContext) => {
  context.registerTabKind({ kind: 'chat', component: null, title: 'Chat' })
}

describe('the tab kind a plugin adds', () => {
  const activateChat = async () => {
    const { windowTabs, support } = createSupport()
    activate(support.pluginHost.createContext(manifest.id, manifest))
    return { windowTabs, support }
  }

  it('is a kind of the window while the plugin is active', async () => {
    const { windowTabs } = await activateChat()

    expect(windowTabs.hasKind('chat')).toBe(true)
    expect(windowTabs.getTab(await windowTabs.createTab('chat', ''))).not.toBeNull()
  })

  it('goes, with its open tabs turned into placeholders, when the plugin is disabled', async () => {
    const { windowTabs, support } = await activateChat()
    const id = await windowTabs.createTab('chat', '')

    support.pluginHost.disposePlugin(manifest.id)

    expect(windowTabs.hasKind('chat')).toBe(false)
    expect(windowTabs.getTab(id)?.state).toEqual({
      kind: 'chat',
      targetPath: '',
      saved: undefined,
      title: 'Chat',
    })
  })
})

describe('the settings and notices a plugin is given', () => {
  const createWith = (more: Partial<Parameters<typeof createPluginSupport>[0]>) =>
    createPluginSupport({
      windowTabs: useWindowTabs(),
      tabLinks: { removeClosedTab: vi.fn() } as never,
      tabOpeners: {
        registerExtensionReader: vi.fn(() => () => undefined),
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      settingsHolder: {},
      ...more,
    })

  it('reads a setting of the plugin, and the default of its switch where none is written', () => {
    const settings = ref<Record<string, unknown>>({ fontSize: 14 })
    const rest = { getSetting: (path: readonly string[]) => settings.value[path[2] ?? ''] }
    const { pluginHost } = createWith({ settingsHolder: { current: { rest } as never } })
    const ctx = pluginHost.createContext('md.numen.a')

    expect(ctx.getSetting('fontSize')).toBe(14)
    expect(ctx.getSetting('enabled')).toBe(DEFAULT_PLUGIN_ENABLED)
    expect(ctx.getSetting('missing')).toBeUndefined()
  })

  it('prefers the committed setting over an unconfirmed pending write', () => {
    const rest = {
      getCommittedSetting: (path: readonly string[]) => (path[2] === 'enabled' ? false : undefined),
      getSetting: () => true,
    }
    const { pluginHost } = createWith({ settingsHolder: { current: { rest } as never } })
    const ctx = pluginHost.createContext('md.numen.a')

    expect(ctx.getSetting('enabled')).toBe(false)
  })

  it('registers the extensions declared for a reader in the plugin settings', () => {
    const registerExtensionReader = vi.fn(() => () => undefined)
    const rest = {
      plugins: ref([
        {
          id: 'md.numen.other',
          readers: [{ id: 'document', extensions: ['.other'] }],
        },
        {
          id: 'md.numen.pdf',
          readers: [
            { id: 'book', extensions: ['.epub'] },
            { id: 'document', extensions: ['.pdf'] },
          ],
        },
      ]),
    }
    const { pluginHost } = createWith({
      tabOpeners: {
        registerExtensionReader,
        registerEditor: vi.fn(() => () => undefined),
      } as never,
      settingsHolder: { current: { rest } as never },
    })

    pluginHost
      .createContext('md.numen.pdf')
      .registerReader({ id: 'document', title: 'Document', component: null })

    expect(registerExtensionReader).toHaveBeenCalledTimes(1)
    expect(registerExtensionReader).toHaveBeenCalledWith(
      '.pdf',
      expect.any(Function),
      undefined,
    )
  })

  it('reads no setting before the window has its settings', () => {
    const { pluginHost } = createWith({})

    expect(pluginHost.createContext('md.numen.a').getSetting('enabled')).toBeUndefined()
  })

  it('calls the handler of a followed setting once per changed value', async () => {
    const settings = ref<Record<string, unknown>>({ fontSize: 14 })
    const rest = { getSetting: (path: readonly string[]) => settings.value[path[2] ?? ''] }
    const { pluginHost } = createWith({ settingsHolder: { current: { rest } as never } })
    const handler = vi.fn()
    pluginHost.createContext('md.numen.a').onSettingChange('fontSize', handler)

    settings.value = { fontSize: 14 }
    await nextTick()
    settings.value = { fontSize: 16 }
    await nextTick()

    expect(handler).toHaveBeenCalledExactlyOnceWith(16)
  })

  it("shows a plugin's notices in the window's one list, and clears them with the plugin", () => {
    const notices = createNoticeQueue()
    const { pluginHost } = createWith({ notices })

    pluginHost.createContext('md.numen.a').notices?.show('Saved.', 'warning')
    expect(notices.notices.value.map((one) => one.text)).toStrictEqual(['Saved.'])

    pluginHost.disposePlugin('md.numen.a')
    expect(notices.notices.value).toStrictEqual([])
  })

  it('gives a plugin no notices where the window keeps no list', () => {
    const { pluginHost } = createWith({})

    expect(pluginHost.createContext('md.numen.a').notices).toBeUndefined()
  })
})

describe('the plugins the window loads', () => {
  it('are the ones that declare a frontend, so a backend alone is never fetched', () => {
    const plugins = [
      { id: 'md.numen.agents.server', hasFrontend: false },
      { id: 'md.numen.views.sketch', hasFrontend: true },
    ]

    expect(getFrontendPluginIds(plugins)).toStrictEqual(['md.numen.views.sketch'])
  })
})
