import { AudioLines } from '@lucide/vue'
import { definePlugin, getFileStem, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import RecordingTab from './ui/RecordingTab.vue'
import MediaRecordingTab from './ui/MediaRecordingTab.vue'
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

export function activate(plugin: PluginContext, host: HostServices): void {
  setTaskHost(host.work.tasks)
  bindTexts(host.vault.texts)
  bindNotes(host.vault.notes)
  bindCommands(host.ui.commands)
  host.events.onVaultChange?.((event) => refreshRecordings(event.paths ?? []))
  plugin.register.reader({
    id: 'media-viewer',
    component: RecordingTab,
    title: 'Audio Player',
    icon: AudioLines,
    capabilities: ['recording', 'media', 'audio'],
  })

  plugin.register.tabKind({
    kind: RECORDER,
    component: MediaRecordingTab,
    title: 'Record Audio',
  })

  plugin.register.tabKind<MediaTabState>({
    kind: MEDIA,
    component: RecordingTab,
    open: (path) => ({ path }),
    getTitle: (state) => getFileStem(state.path),
    identity: (path) => path,
    getTarget: (state) => ({ path: state.path, file: state.path, title: getFileStem(state.path) }),
    getOpenTab: (state) => ({ path: state.path }),
  })

  plugin.register.command({
    id: 'recording.create',
    handler: async () => {
      await host.tabs.open?.(RECORDER)
    },
  })
}

export default definePlugin({
  activate,
})
