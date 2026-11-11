/**
 * A plugin that is disabled leaves its tabs as placeholders that read the catalog; enabling it
 * from the placeholder returns the tab with the state the plugin saved.
 */
import { describe, expect, it, vi } from 'vitest'
import { createSettingsStore, SETTINGS_WORDS } from '@/entities/settings'
import { defineComponent, h } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { CATALOG_STATE, type CatalogEntry, type Manifest } from '@numen/core-client'
import type { JsonValue } from '@numen/plugin-loader'
import { createMockCatalogEntry, createMockManifest } from '@numen/core-client/testing'
import { answerTabKinds } from '@/testing/tabCatalog'
import { createTestTabs } from '@/testing/tabs'
import { createPluginSupport } from './pluginSupport'

const PLUGIN_ID = 'sample.decks'

const manifest: Manifest = createMockManifest(PLUGIN_ID, {
  name: 'Decks',
  permissions: ['vault:read'],
  tabKinds: ['deck'],
})

const DeckPane = defineComponent({ render: () => h('div', { class: 'deck' }, 'the deck') })

const entryOf = (state: typeof CATALOG_STATE.INSTALLED | typeof CATALOG_STATE.DISABLED) =>
  createMockCatalogEntry(PLUGIN_ID, {
    name: 'Decks',
    feature: 'flashcard decks',
    tabKinds: [{ kind: 'deck', title: 'Deck' }],
    state,
  })

const createWindow = () => {
  const entries = [entryOf(CATALOG_STATE.INSTALLED)]
  const lookups = answerTabKinds(entries)
  const catalog = {
    setEntries: (next: CatalogEntry[]) => entries.splice(0, entries.length, ...next),
    getAskedKinds: () => lookups,
  }
  const enabled: string[] = []
  const holder: { activate?: () => void } = {}
  const windowTabs = createTestTabs({
    enablePlugin: (pluginId) => {
      enabled.push(pluginId)
      catalog.setEntries([entryOf(CATALOG_STATE.INSTALLED)])
      holder.activate?.()
      return Promise.resolve()
    },
  })
  const support = createPluginSupport({
    windowTabs,
    tabLinks: {} as never,
    tabOpeners: {
      registerExtensionReader: () => () => undefined,
      registerEditor: () => () => undefined,
    } as never,
    rest: createSettingsStore(SETTINGS_WORDS, vi.fn()),
  })
  holder.activate = () => {
    support.pluginHost.lifecycle
      .createContext(PLUGIN_ID, manifest)
      .register.tabKind<{ targetPath: string; saved?: JsonValue | undefined }>({
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
    windowTabs.restored.restoreTabs(
      [{ id: 'deck:a', kind: 'deck', path: 'a.deck', saved: { page: 7 } }],
      'deck:a',
    )
    catalog.setEntries([entryOf(CATALOG_STATE.DISABLED)])
    support.pluginHost.lifecycle.disposePlugin(PLUGIN_ID)
    const placeholder = windowTabs.tabs.getTab('deck:a')
    expect(windowTabs.groups.hasKind('deck')).toBe(false)
    expect(placeholder?.kind.pane).not.toBe(DeckPane)

    const pane = mount(placeholder?.kind.pane ?? DeckPane, { props: { state: placeholder?.state } })
    await vi.waitFor(() => expect(pane.text()).toContain('Plugin disabled'))

    expect(pane.text()).toContain('Plugin disabled')
    expect(pane.text()).toContain('The plugin for flashcard decks is disabled.')
    expect(catalog.getAskedKinds()).toContain('deck')

    await pane.find('button').trigger('click')
    await flushPromises()

    expect(enabled).toStrictEqual([PLUGIN_ID])
    const returned = windowTabs.tabs.getTab('deck:a')
    expect(windowTabs.groups.hasKind('deck')).toBe(true)
    expect(returned?.kind).not.toBe(placeholder?.kind)
    expect(returned?.state).toStrictEqual({ targetPath: 'a.deck', saved: { page: 7 } })
  })

  it('keeps the tab title while the plugin is off', async () => {
    const { catalog, windowTabs, support } = createWindow()
    const id = await windowTabs.tabs.createTab('deck', 'a.deck')

    catalog.setEntries([entryOf(CATALOG_STATE.DISABLED)])
    support.pluginHost.lifecycle.disposePlugin(PLUGIN_ID)
    await flushPromises()

    expect(windowTabs.tabs.tabs.value).toEqual([{ id, title: 'Deck' }])
  })
})
