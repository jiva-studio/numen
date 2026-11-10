/**
 * A kind of tab that shows a note, registered in a mounted window the way a plugin registers it:
 * the vault reports its notes as that kind, and the window opens them in it.
 */
import { defineComponent, h, reactive } from 'vue'
import type { createWindow } from '@/app/window/createWindow'
import type { TabKind } from '@/entities/tab'
import { mockVaultState } from '../answers'

const NOTE_TAB = 'sample'

interface NoteTabState {
  path: string
}

const pane = defineComponent({
  props: { state: { type: Object, required: true } },
  setup: () => () => h('div'),
})

const noteTabKind: TabKind<NoteTabState> = {
  kind: NOTE_TAB,
  open: (targetPath) => reactive({ path: targetPath }),
  pane,
  getTitle: (state) => state.path,
  address: {
    identity: (targetPath) => targetPath,
    navigate: (state, path) => {
      state.path = path
    },
    getTarget: (state) => ({ path: state.path, file: state.path, title: state.path }),
    getOpenTab: (state) => ({ path: state.path }),
  },
}

type WindowParts = ReturnType<typeof createWindow>

/** Makes the window open the notes of the vault in tabs of the sample kind. */
export const registerNoteTabs = ({ tabs }: Pick<WindowParts, 'tabs'>): void => {
  const { windowTabs, tabOpeners } = tabs
  mockVaultState.tabKind = NOTE_TAB
  windowTabs.groups.registerKind(noteTabKind)
  tabOpeners.registerEditor(NOTE_TAB, (path) => {
    windowTabs.tabs.openTabOfKind(NOTE_TAB, path).catch(() => undefined)
  })
}

/** Opens a note in a tab of the sample kind and returns the id of the tab. */
export const openNoteTab = ({ tabs }: Pick<WindowParts, 'tabs'>, path: string): Promise<string> =>
  tabs.windowTabs.tabs.openTabOfKind(NOTE_TAB, path)
