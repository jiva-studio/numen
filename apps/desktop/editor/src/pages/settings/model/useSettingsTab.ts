/**
 * The state of the settings tab: which plugin page is open, the plugin of that page, and the page
 * and search query the tab hands back to be remembered.
 */
import { computed, ref, watch } from 'vue'
import type { PluginInfo } from '@/entities/settings'
import { createManagedFromEntries } from '../lib/pluginListing'
import type { SettingsTabState } from '../types'

export function useSettingsTab(state: SettingsTabState) {
  const { initialPluginId, initialSearchQuery, installation } = state
  const selectedPluginId = ref<string | null>(initialPluginId.value)

  const plugins = computed<readonly PluginInfo[]>(() => installation.plugin.plugins.value)
  const uninstalled = computed(() =>
    createManagedFromEntries([], installation.plugin.catalog.value).find(
      (one) => one.id === selectedPluginId.value,
    ),
  )

  /**
   * The plugin of the open page, as the last read that matched it gave it. A read that does not
   * list it, or lists nothing, does not close the page.
   */
  let lastSelected: PluginInfo | undefined
  const selectedPlugin = computed<PluginInfo | undefined>(() => {
    const match = plugins.value.find((one) => one.id === selectedPluginId.value)
    if (match) lastSelected = match
    return match ?? (lastSelected?.id === selectedPluginId.value ? lastSelected : undefined)
  })

  watch(
    () => initialPluginId.value,
    (newId) => {
      selectedPluginId.value = newId
    },
  )

  function selectPlugin(pluginId: string): void {
    selectedPluginId.value = pluginId
    initialPluginId.value = pluginId
  }

  function updateQuery(query: string): void {
    initialSearchQuery.value = query
  }

  function goBack(): void {
    selectedPluginId.value = null
    initialPluginId.value = null
  }

  return {
    installation,
    searchQuery: initialSearchQuery,
    selectedPluginId,
    plugins,
    uninstalled,
    selectedPlugin,
    actions: { selectPlugin, updateQuery, goBack },
  }
}
