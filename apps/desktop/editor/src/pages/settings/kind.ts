/**
 * The settings tab, as the window keeps it.
 */
import { ref } from 'vue'
import { runInBackground } from '@numen/ui'
import type { TabKind, WindowHandle } from '@/entities/tab'
import { SETTINGS } from '@/entities/tab'
import SettingsTab from './ui/SettingsTab.vue'
import type { Installation, SettingsTabState } from './types'
import { WORDS as words } from './words'

export function useSettingsTab(
  handle: WindowHandle,
  installation: Installation,
  reportError: (error: unknown) => void,
) {
  const isOpen = ref(false)
  const initialPluginId = ref<string | null>(null)
  const initialSearchQuery = ref('')
  const state: SettingsTabState = { installation, initialPluginId, initialSearchQuery }

  /** One settings tab to a window: the settings are the installation's, not a file's. */
  const kind: TabKind<SettingsTabState, typeof SETTINGS> = {
    kind: SETTINGS,
    open: (pluginId?: string) => {
      isOpen.value = true
      if (pluginId) {
        initialPluginId.value = pluginId
      }
      return state
    },
    getTitle: () => words.settings,
    pane: SettingsTab,
    identity: () => SETTINGS,
    isTool: true,
  }

  /** Opens the settings tab, optionally at one plugin or with a search filter. */
  const openSettings = (pluginId?: string, filter?: string): void => {
    isOpen.value = true
    if (filter !== undefined) {
      initialPluginId.value = null
      initialSearchQuery.value = filter
    } else if (pluginId !== undefined) {
      initialPluginId.value = pluginId
    }
    runInBackground(handle.openTab(SETTINGS, pluginId), reportError)
  }

  return { kind, state, openSettings, isOpen }
}
