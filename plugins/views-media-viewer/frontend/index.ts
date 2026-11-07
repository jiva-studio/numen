import { AudioLines } from '@lucide/vue'
import { definePlugin, getFileStem, type PluginContext } from '@numen/plugin-sdk'
import RecordingTab from './RecordingTab.vue'
import MediaRecordingTab from './MediaRecordingTab.vue'
import { bindCommands, bindNotes, bindTexts, refreshRecordings, setTaskHost } from './api/wire'

export { RecordingTab, MediaRecordingTab }

/** The tab kind in which a new recording is made. */
export const RECORDER = 'recording'

/** The tab kind a note with media opens in, as the manifest's when condition selects it. */
export const MEDIA = 'media'

/** What a note's tab holds: the path of the note. */
interface MediaTabState {
  readonly path: string
}

export function activate(context: PluginContext): void {
  setTaskHost(context.tasks)
  bindTexts(context.texts)
  bindNotes(context.notes)
  bindCommands(context.commands)
  context.onVaultChange?.((event) => refreshRecordings(event.paths ?? []))
  context.registerReader({
    id: 'media-viewer',
    component: RecordingTab,
    title: 'Audio Player',
    icon: AudioLines,
    capabilities: ['recording', 'media', 'audio'],
  })

  context.registerTabKind({
    kind: RECORDER,
    component: MediaRecordingTab,
    title: 'Record Audio',
  })

  context.registerTabKind<MediaTabState>({
    kind: MEDIA,
    component: RecordingTab,
    open: (path) => ({ path }),
    getTitle: (state) => getFileStem(state.path),
    identity: (path) => path,
    getTarget: (state) => ({ path: state.path, file: state.path, title: getFileStem(state.path) }),
    getOpenTab: (state) => ({ path: state.path }),
  })

  context.registerCommand({
    id: 'recording.create',
    handler: async () => {
      await context.openTab?.(RECORDER)
    },
  })
}

export default definePlugin({
  activate,
})
