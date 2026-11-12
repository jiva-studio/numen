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
import type { Caught } from '@/shared/api/errors'
import type { SettingsRequest } from '@/shared/model/command'

export function useSettingsTab(
  handle: WindowHandle,
  installation: Installation,
  reportError: (error: Caught) => void,
) {
  const isOpen = ref(false)
  const initialPluginId = ref<string | null>(null)
  const initialSearchQuery = ref('')
  const markedSetting = ref('')
  const state: SettingsTabState = {
    installation,
    initialPluginId,
    initialSearchQuery,
    markedSetting,
  }

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
    pane: SettingsTab,
    getTitle: () => words.settings,
    display: { isTool: true },
    address: { identity: () => SETTINGS },
  }

  /**
   * Opens the settings tab, optionally at one plugin with one of its settings marked, or with a
   * search filter.
   */
  const openSettings = (request: SettingsRequest & { readonly filter?: string } = {}): void => {
    const { pluginId, setting, filter } = request
    isOpen.value = true
    if (filter !== undefined) {
      initialPluginId.value = null
      initialSearchQuery.value = filter
    } else if (pluginId !== undefined) {
      initialPluginId.value = pluginId
      markedSetting.value = setting ?? ''
    }
    runInBackground(handle.openTab(SETTINGS, pluginId), reportError)
  }

  return { kind, state, openSettings, isOpen }
}
