import { watch } from 'vue'
import { createPluginHost } from '@/entities/plugin'
import { createPluginLoader, type PluginLoader } from '@/features/plugin-loader'
import type { fileOpeners, useWindowTabs } from '@/entities/tab'
import type { useTabLinks } from '@/features/tab-linking'
import type { useSettings } from './useSettings'

export interface PluginSupportDeps {
  readonly held: ReturnType<typeof useWindowTabs>
  readonly tabLinks: ReturnType<typeof useTabLinks>
  readonly tabOpeners: ReturnType<typeof fileOpeners>
  readonly settingsHolder: { current?: ReturnType<typeof useSettings> }
}

export function createPluginSupport(deps: PluginSupportDeps) {
  const { held, tabLinks, tabOpeners, settingsHolder } = deps

  const closeTabsForPlugin = (pluginId: string) => {
    for (const tab of held.tabs.value) {
      const windowTab = held.getTab(tab.id)
      if (!windowTab) continue
      const state = windowTab.state as { pluginId?: string } | undefined
      if (state?.pluginId === pluginId) {
        tabLinks.removeClosedTab(tab.id)
        held.requestClose(tab.id)
      }
    }
  }

  const getSetting = (pluginId: string, key: string): unknown => {
    const rest = settingsHolder.current?.rest
    if (!rest) return undefined
    const value = rest.getSetting?.(['plugins', pluginId, key])
    if (value !== undefined) return value
    if (key === 'enabled') {
      const info = rest.plugins?.value?.find((p: { id: string; isActive?: boolean }) => p.id === pluginId)
      if (info !== undefined) return info.isActive
      return true
    }
    return undefined
  }

  const onSettingChange = (
    pluginId: string,
    key: string,
    handler: (value: unknown) => void | Promise<void>,
  ): (() => void) => {
    return watch(
      () => getSetting(pluginId, key),
      (newVal, oldVal) => {
        if (newVal !== oldVal) {
          void handler(newVal)
        }
      },
    )
  }

  const pluginHost = createPluginHost({
    registerKind: held.registerKind,
    registerExtensionReader: tabOpeners.registerExtensionReader,
    openTab: held.openTabOfKind,
    openTabBeside: (kind, at) => held.handle.openTabBeside(kind, at),
    registerEditor: tabOpeners.registerEditor,
    getSetting,
    onSettingChange,
    closeTabsForPlugin,
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
