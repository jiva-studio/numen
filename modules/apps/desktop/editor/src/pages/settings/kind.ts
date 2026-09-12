/**
 * The settings tab, as the window keeps it.
 */
import { ref } from 'vue'
import type { TabKind, WindowHandle } from '@/entities/tab'
import { SETTINGS } from '@/entities/tab'
import SettingsTab from './ui/SettingsTab.vue'
import type { Installation, SettingsTabState } from './model/useSettingsTab'
import { WORDS as words } from './words'

export function useSettingsTab(handle: WindowHandle, installation: Installation) {
  const isOpen = ref(false)
  const state: SettingsTabState = { installation }

  /** One settings tab to a window: the settings are the installation's, not a file's. */
  const kind: TabKind<SettingsTabState, typeof SETTINGS> = {
    kind: SETTINGS,
    open: () => {
      isOpen.value = true
      return state
    },
    getTitle: () => words.settings,
    pane: SettingsTab,
    identity: () => SETTINGS,
  }

  /** The settings put in front of the person. */
  const openSettings = (): void => {
    isOpen.value = true
    void handle.openTab(SETTINGS)
  }

  return { kind, state, openSettings, isOpen }
}
