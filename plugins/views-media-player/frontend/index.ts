import { Mic } from '@lucide/vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import RecordingTab from './RecordingTab.vue'
import MediaRecordingTab from './MediaRecordingTab.vue'
import { WORDS } from './words'
import { setTaskHost } from './api/wire'

export { RecordingTab, MediaRecordingTab }

/** The kind of the tab a new recording is made in. */
export const RECORDER = 'recording'

export function activate(context: PluginContext): void {
  setTaskHost(context.tasks)
  context.registerReader({
    id: 'media-player',
    fileExtensions: ['.mp3', '.m4a', '.wav', '.ogg', '.aac', '.flac'],
    component: RecordingTab,
    title: 'Audio Player',
    source: 'recording',
    capabilities: ['recording', 'media', 'audio'],
  })

  context.registerTabKind({
    kind: RECORDER,
    component: MediaRecordingTab,
    title: 'Record Audio',
  })

  context.registerCommand({
    id: 'record',
    title: WORDS.record,
    icon: Mic,
    handler: async () => {
      await context.openTab?.(RECORDER)
    },
  })
}

export default definePlugin({
  activate,
})
