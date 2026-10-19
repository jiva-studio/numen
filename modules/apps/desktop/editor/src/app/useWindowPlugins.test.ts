import { describe, expect, it, vi } from 'vitest'
import { useWindowTabs } from '@/entities/tab'
import type { AnyTabKind } from '@/entities/tab'
import type { AgentHost, WorkspaceHost } from '@numen/plugin-sdk'
import { activate } from '@plugins/view-agent-chat/frontend'
import manifest from '@plugins/view-agent-chat/manifest.json'
import { createPluginSupport } from './useWindowPlugins'

const createSupport = () => {
  const held = useWindowTabs()
  const removeClosedTab = vi.fn()
  const support = createPluginSupport({
    held,
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
  return { held, support, removeClosedTab }
}

const coreKind = (state: object): AnyTabKind => ({
  kind: 'core',
  open: () => state,
  pane: {},
})

describe('closing the tabs of a plugin', () => {
  it('closes the tabs of the kinds the plugin registered, whatever their state holds', async () => {
    const { held, support, removeClosedTab } = createSupport()
    support.pluginHost
      .createContext('md.numen.a')
      .registerTabKind({ kind: 'a-kind', component: null, title: 'A' })
    const id = await held.createTab('a-kind', 'x')
    held.show(id)

    support.pluginHost.disposePlugin('md.numen.a')

    expect(held.getTab(id)).toBeNull()
    expect(removeClosedTab).toHaveBeenCalledWith(id)
  })

  it('keeps a tab of a core kind whose state names the plugin', async () => {
    const { held, support } = createSupport()
    held.registerKind(coreKind({ pluginId: 'md.numen.a' }))
    const id = await held.createTab('core', 'x')

    support.pluginHost.disposePlugin('md.numen.a')

    expect(held.getTab(id)).not.toBeNull()
  })

  it('keeps the tabs of another plugin', async () => {
    const { held, support } = createSupport()
    support.pluginHost
      .createContext('md.numen.b')
      .registerTabKind({ kind: 'b-kind', component: null, title: 'B' })
    const id = await held.createTab('b-kind', 'x')

    support.pluginHost.disposePlugin('md.numen.a')

    expect(held.getTab(id)).not.toBeNull()
  })
})

describe('the agent chat tab', () => {
  const activateChat = async () => {
    const { held, support } = createSupport()
    await activate(support.pluginHost.createContext(manifest.id, manifest))
    return { held, support }
  }

  it('is a kind of the window while the plugin is active', async () => {
    const { held } = await activateChat()

    expect(held.hasKind('agent')).toBe(true)
    expect(held.getTab(await held.createTab('agent', ''))).not.toBeNull()
  })

  it('goes, with its open tabs, when the plugin is disabled', async () => {
    const { held, support } = await activateChat()
    const id = await held.createTab('agent', '')

    support.pluginHost.disposePlugin(manifest.id)

    expect(held.hasKind('agent')).toBe(false)
    expect(held.getTab(id)).toBeNull()
  })
})
