import { describe, expect, it, vi } from 'vitest'
import { useWindowTabs } from '@/entities/tab'
import type { AnyTabKind } from '@/entities/tab'
import type { AgentHost, WorkspaceHost } from '@numen/plugin-sdk'
import { activate } from '@plugins/views.agent.chat/frontend'
import manifest from '@plugins/views.agent.chat/manifest.json'
import { createPluginSupport } from './useWindowPlugins'

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
  it('closes the tabs of the kinds the plugin registered, whatever their state holds', async () => {
    const { windowTabs, support, removeClosedTab } = createSupport()
    support.pluginHost
      .createContext('md.numen.a')
      .registerTabKind({ kind: 'a-kind', component: null, title: 'A' })
    const id = await windowTabs.createTab('a-kind', 'x')
    windowTabs.show(id)

    support.pluginHost.disposePlugin('md.numen.a')

    expect(windowTabs.getTab(id)).toBeNull()
    expect(removeClosedTab).toHaveBeenCalledWith(id)
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

describe('the agent chat tab', () => {
  const activateChat = async () => {
    const { windowTabs, support } = createSupport()
    await activate(support.pluginHost.createContext(manifest.id, manifest))
    return { windowTabs, support }
  }

  it('is a kind of the window while the plugin is active', async () => {
    const { windowTabs } = await activateChat()

    expect(windowTabs.hasKind('agent')).toBe(true)
    expect(windowTabs.getTab(await windowTabs.createTab('agent', ''))).not.toBeNull()
  })

  it('goes, with its open tabs, when the plugin is disabled', async () => {
    const { windowTabs, support } = await activateChat()
    const id = await windowTabs.createTab('agent', '')

    support.pluginHost.disposePlugin(manifest.id)

    expect(windowTabs.hasKind('agent')).toBe(false)
    expect(windowTabs.getTab(id)).toBeNull()
  })
})
