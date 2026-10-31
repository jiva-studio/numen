/**
 * A plugin that is disabled leaves its tabs as placeholders that read the catalog; enabling it
 * from the placeholder returns the tab with the state the plugin saved.
 */
import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { CATALOG_STATE, type Manifest } from '@numen/plugin-sdk'
import {
  createMockCatalogEntry,
  createMockManifest,
  createMockPluginCatalog,
} from '@numen/plugin-sdk/testing'
import { useWindowTabs } from '@/entities/tab'
import { createPluginSupport } from './useWindowPlugins'

const PLUGIN_ID = 'sample.decks'

const manifest: Manifest = createMockManifest(PLUGIN_ID, { name: 'Decks' })

const DeckPane = defineComponent({ render: () => h('div', { class: 'deck' }, 'the deck') })

const entryOf = (state: typeof CATALOG_STATE.INSTALLED | typeof CATALOG_STATE.DISABLED) =>
  createMockCatalogEntry(PLUGIN_ID, {
    name: 'Decks',
    feature: 'flashcard decks',
    tabKinds: [{ kind: 'deck', title: 'Deck' }],
    state,
  })

/** A window with the plugin active and one of its tabs open. */
const createWindow = () => {
  const catalog = createMockPluginCatalog([entryOf(CATALOG_STATE.INSTALLED)])
  const enabled: string[] = []
  const holder: { activate?: () => void } = {}
  const windowTabs = useWindowTabs({
    catalog,
    enablePlugin: (pluginId) => {
      enabled.push(pluginId)
      catalog.setEntries([entryOf(CATALOG_STATE.INSTALLED)])
      holder.activate?.()
      return Promise.resolve()
    },
  })
  const support = createPluginSupport({
    windowTabs,
    tabLinks: { removeClosedTab: () => undefined } as never,
    tabOpeners: {
      registerExtensionReader: () => () => undefined,
      registerEditor: () => () => undefined,
    } as never,
    settingsHolder: {},
  })
  holder.activate = () => {
    support.pluginHost
      .createContext(PLUGIN_ID, manifest)
      .registerTabKind<{ targetPath: string; saved?: unknown }>({
        kind: 'deck',
        component: DeckPane,
        title: 'Deck',
        open: (targetPath, _id, saved) => ({ targetPath, saved }),
        getOpenTab: (state) => ({ path: state.targetPath }),
        getState: (state) => state.saved ?? { page: 1 },
      })
  }
  holder.activate()
  return { catalog, enabled, windowTabs, support }
}

describe('a disabled plugin', () => {
  it('leaves a placeholder that names the plugin from the catalog, and enabling returns the tab', async () => {
    const { catalog, enabled, windowTabs, support } = createWindow()
    windowTabs.restoreTabs(
      [{ id: 'deck:a', kind: 'deck', path: 'a.deck', saved: { page: 7 } }],
      'deck:a',
    )
    catalog.setEntries([entryOf(CATALOG_STATE.DISABLED)])
    support.pluginHost.disposePlugin(PLUGIN_ID)
    const placeholder = windowTabs.getTab('deck:a')
    expect(windowTabs.hasKind('deck')).toBe(false)
    expect(placeholder?.kind.pane).not.toBe(DeckPane)

    const pane = mount(placeholder?.kind.pane ?? DeckPane, { props: { state: placeholder?.state } })
    await flushPromises()

    expect(pane.text()).toContain('Plugin disabled')
    expect(pane.text()).toContain('The plugin for flashcard decks is disabled.')
    expect(catalog.getAskedKinds()).toContain('deck')

    await pane.find('button').trigger('click')
    await flushPromises()

    expect(enabled).toStrictEqual([PLUGIN_ID])
    const returned = windowTabs.getTab('deck:a')
    expect(windowTabs.hasKind('deck')).toBe(true)
    expect(returned?.kind).not.toBe(placeholder?.kind)
    expect(returned?.state).toStrictEqual({ targetPath: 'a.deck', saved: { page: 7 } })
  })

  it('keeps the tab title while the plugin is off', async () => {
    const { catalog, windowTabs, support } = createWindow()
    const id = await windowTabs.createTab('deck', 'a.deck')

    catalog.setEntries([entryOf(CATALOG_STATE.DISABLED)])
    support.pluginHost.disposePlugin(PLUGIN_ID)
    await flushPromises()

    expect(windowTabs.tabs.value).toEqual([{ id, title: 'Deck' }])
  })
})
