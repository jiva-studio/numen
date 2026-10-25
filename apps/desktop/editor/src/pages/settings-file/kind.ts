/**
 * The settings file tab, as the window keeps it.
 */
import type { TabKind, WindowHandle } from '@/entities/tab'
import { SETTINGS_FILE } from '@/entities/tab'
import SettingsFileTab from './ui/SettingsFileTab.vue'
import {
  useSettingsFile,
  type SettingsFileTabDeps,
  type SettingsFileTabState,
} from './model/useSettingsFile'
import { WORDS as words } from './words'

/** What the tab carries beside its name, and nothing where there is nothing to say. */
const getMark = (state: SettingsFileTabState): string | undefined => {
  if (state.isStale.value) return 'stale'
  return state.changed.value ? '•' : undefined
}

/**
 * The settings file tab. There is one file, so opening it again is the tab it
 * already stands in.
 */
export function createSettingsFileTabKind(
  handle: WindowHandle,
  core: SettingsFileTabDeps,
  readSettings: () => void,
) {
  const kind: TabKind<SettingsFileTabState, typeof SETTINGS_FILE> = {
    kind: SETTINGS_FILE,
    open: () => {
      const state = useSettingsFile(core, readSettings)
      void state.reload()
      return state
    },
    getTitle: () => words.called,
    getMark,
    pane: SettingsFileTab,
    identity: () => SETTINGS_FILE,
  }

  /** The file put in front of the person, beside what they were looking at. */
  const openSettingsFile = (): void => void handle.openTabBeside(SETTINGS_FILE)

  return { kind, openSettingsFile }
}
