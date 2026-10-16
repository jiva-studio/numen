import { watch } from 'vue'
import { createPluginHost } from '@/entities/plugin'
import { createPluginLoader, type PluginLoader } from '@/features/plugin-loader'
import { cards } from '@/entities/deck'
import type { fileOpeners, useWindowTabs } from '@/entities/tab'
import type { useTabLinks } from '@/features/tab-linking'
import type { useSettings } from './useSettings'

export interface PluginSupportDeps {
  readonly held: ReturnType<typeof useWindowTabs>
  readonly tabLinks: ReturnType<typeof useTabLinks>
  readonly tabOpeners: ReturnType<typeof fileOpeners>
  readonly settingsHolder: { current?: ReturnType<typeof useSettings> }
}

function createPluginSettingsAccessor(settingsHolder: { current?: ReturnType<typeof useSettings> }) {
  const getSetting = (pluginId: string, key: string): unknown => {
    const rest = settingsHolder.current?.rest
    if (!rest) return undefined
    const value = rest.getSetting?.(['plugins', pluginId, key])
    if (value !== undefined) return value
    if (key === 'enabled') {
      const info = rest.plugins?.value?.find((p: { id: string; isActive?: boolean }) => p.id === pluginId)
      return info?.isActive ?? true
    }
    return undefined
  }

  const onSettingChange = (
    pluginId: string,
    key: string,
    handler: (value: unknown) => void | Promise<void>,
  ): (() => void) =>
    watch(
      () => getSetting(pluginId, key),
      (newVal, oldVal) => {
        if (newVal !== oldVal) void handler(newVal)
      },
    )

  return { getSetting, onSettingChange }
}

function writeCustomCard(at: string, values: readonly { field: string; text: string }[]) {
  void cards.writeDeck(
    at,
    {
      preamble: '',
      cards: [{ values }] as unknown as Parameters<typeof cards.writeDeck>[1]['cards'],
      sections: [],
      tail: '',
    },
    null,
  )
}

export function createPluginSupport(deps: PluginSupportDeps) {
  const { held, tabLinks, tabOpeners, settingsHolder } = deps
  const { getSetting, onSettingChange } = createPluginSettingsAccessor(settingsHolder)

  const closeTabsForPlugin = (pluginId: string) => {
    for (const tab of held.tabs.value) {
      const windowTab = held.getTab(tab.id)
      if ((windowTab?.state as { pluginId?: string } | undefined)?.pluginId === pluginId) {
        tabLinks.removeClosedTab(tab.id)
        held.requestClose(tab.id)
      }
    }
  }

  const pluginHost = createPluginHost({
    registerKind: held.registerKind,
    registerExtensionReader: tabOpeners.registerExtensionReader,
    registerEditor: tabOpeners.registerEditor,
    openTab: held.openTabOfKind,
    openTabBeside: held.openTabBesideOfKind,
    closeTab: held.finishClose,
    getSetting,
    onSettingChange,
    closeTabsForPlugin,
    onCustomCardAdded: writeCustomCard,
  })
  const pluginLoader = createPluginLoader({ host: pluginHost })

  return { pluginHost, pluginLoader }
}

export function watchPluginSettings(
  settings: ReturnType<typeof useSettings>,
  pluginLoader: PluginLoader,
) {
  watch(
    () => settings.rest?.plugins?.value ?? [],
    (plugins) => {
      void (async () => {
        for (const plugin of plugins) {
          if (!plugin.isActive && pluginLoader.isPluginActive(plugin.id)) {
            await pluginLoader.deactivatePlugin(plugin.id)
          } else if (plugin.isActive && !pluginLoader.isPluginActive(plugin.id)) {
            await pluginLoader.activatePlugin(plugin.id)
          }
        }
      })()
    },
  )
}
