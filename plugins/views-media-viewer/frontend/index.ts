import { AudioLines } from '@lucide/vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import RecordingTab from './RecordingTab.vue'
import MediaRecordingTab from './MediaRecordingTab.vue'
import { bindCommands, bindTexts, refreshRecordings, setTaskHost } from './api/wire'

export { RecordingTab, MediaRecordingTab }

/** The tab kind in which a new recording is made. */
export const RECORDER = 'recording'

export function activate(context: PluginContext): void {
  setTaskHost(context.tasks)
  bindTexts(context.texts)
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
