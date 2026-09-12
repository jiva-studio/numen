/**
 * The settings file's tab, as the window keeps it.
 */
import type { TabKind, WindowHandle } from '@/entities/tab'
import { SETTINGS_FILE } from '@/entities/tab'
import TextEditorTab from './ui/TextEditorTab.vue'
import { useTextEditor, type TextEditorTabDeps, type TextEditorTabState } from './model/useTextEditor'
import { WORDS as words } from './words'

/** What the tab carries beside its name, and nothing where there is nothing to say. */
const mark = (state: TextEditorTabState): string | undefined => {
  if (state.isStale.value) return 'stale'
  return state.changed.value ? '•' : undefined
}

/**
 * The settings file's tab. There is one file, so opening it again is the tab it
 * already stands in.
 */
export function createTextEditorTabKind(
  handle: WindowHandle,
  core: TextEditorTabDeps,
  readSettings: () => void,
) {
  const kind: TabKind<TextEditorTabState, typeof SETTINGS_FILE> = {
    kind: SETTINGS_FILE,
    open: () => {
      const state = useTextEditor(core, readSettings)
      void state.reload()
      return state
    },
    getTitle: () => words.called,
    getMark: mark,
    pane: TextEditorTab,
    identity: () => SETTINGS_FILE,
  }

  /** The file put in front of the person, beside what they were looking at. */
  const openSettingsFile = (): void => void handle.openTabBeside(SETTINGS_FILE)

  return { kind, openSettingsFile }
}
