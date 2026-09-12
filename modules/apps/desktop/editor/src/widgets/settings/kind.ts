/**
 * The settings tab, as the window keeps it.
 */
import { ref } from 'vue'
import type { TabKind, WindowHandle } from '@/entities/tab/windowTabs'
import { SETTINGS } from '@/entities/tab/workspace'
import SettingsTab from './ui/SettingsTab.vue'
import type { Installation, SettingsTabState } from './model/useSettingsTab'
import { WORDS as words } from './words'

export function useSettingsTab(handle: WindowHandle, installation: Installation) {
  const isOpen = ref(false)
  const state: SettingsTabState = { installation }

  /** One settings tab to a window: the settings are the installation's, not a file's. */
  const kind: TabKind<SettingsTabState, typeof SETTINGS> = {
    kind: SETTINGS,
    opens: () => {
      isOpen.value = true
      return state
    },
    called: () => words.settings,
    getTitle: () => words.settings,
    draws: SettingsTab,
    identity: () => SETTINGS,
  }

  /** The settings put in front of the person. */
  const shows = (): void => {
    isOpen.value = true
    void handle.opens(SETTINGS)
  }

  return { kind, state, shows, isOpen }
}
