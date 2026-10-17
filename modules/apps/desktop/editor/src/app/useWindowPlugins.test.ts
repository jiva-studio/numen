import { describe, expect, it, vi } from 'vitest'
import { useWindowTabs } from '@/entities/tab'
import type { AnyTabKind } from '@/entities/tab'
import { createPluginSupport } from './useWindowPlugins'

function createSupport() {
  const held = useWindowTabs()
  const removeClosedTab = vi.fn()
  const support = createPluginSupport({
    held,
    tabLinks: { removeClosedTab } as never,
    tabOpeners: {
      registerExtensionReader: vi.fn(() => () => {}),
      registerEditor: vi.fn(() => () => {}),
    } as never,
    settingsHolder: {},
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
