/**
 * A kind of tab that shows a note, registered in a mounted window the way a plugin registers it:
 * the vault reports its notes as that kind, and the window opens them in it.
 */
import { defineComponent, h, reactive } from 'vue'
import { createMockManifest } from '@numen/core-client/testing'
import type { PluginHost } from '@numen/plugin-loader'
import type { WindowParts } from '@/app/window/createWindow'
import { mockVaultState } from '../answers'

const NOTE_TAB = 'sample'

const NOTE_PLUGIN = 'org.example.notes'

interface NoteTabState {
  path: string
}

const pane = defineComponent({
  props: { state: { type: Object, required: true } },
  setup: () => () => h('div'),
})

type TabKindRegistration = Parameters<
  ReturnType<PluginHost['lifecycle']['createContext']>['plugin']['register']['tabKind']
>[0]

const isNoteTab = (state: object): state is NoteTabState => 'path' in state

const getPath = (state: object): string => (isNoteTab(state) ? state.path : '')

const noteTabKind: TabKindRegistration = {
  kind: NOTE_TAB,
  component: pane,
  open: (targetPath) => reactive<NoteTabState>({ path: targetPath }),
  getTitle: getPath,
  identity: (targetPath) => targetPath,
  navigate: (state, path) => {
    if (isNoteTab(state)) state.path = path
  },
  getTarget: (state) => ({ path: getPath(state), file: getPath(state), title: getPath(state) }),
  getOpenTab: (state) => ({ path: getPath(state) }),
}

type NoteTabsWindow = {
  readonly pluginHost: PluginHost
  readonly tabs: Pick<WindowParts['tabs'], 'windowTabs'>
}

/** Makes the window open the notes of the vault in tabs of the sample kind. */
export const registerNoteTabs = ({
  pluginHost: host,
}: Pick<NoteTabsWindow, 'pluginHost'>): void => {
  mockVaultState.tabKind = NOTE_TAB
  host.lifecycle
    .createContext(
      NOTE_PLUGIN,
      createMockManifest(NOTE_PLUGIN, {
        name: 'Notes',
        permissions: ['vault:read'],
        tabKinds: [NOTE_TAB],
      }),
    )
    .plugin.register.tabKind(noteTabKind)
}

/** Opens a note in a tab of the sample kind and returns the id of the tab. */
export const openNoteTab = (
  { tabs }: Pick<NoteTabsWindow, 'tabs'>,
  path: string,
): Promise<string> => tabs.windowTabs.tabs.openTabOfKind(NOTE_TAB, path)
