import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import RecordingTab from './RecordingTab.vue'
import MediaRecordingTab from './MediaRecordingTab.vue'

export { RecordingTab, MediaRecordingTab }

export function activate(context: PluginContext): void {
  context.registerReader({
    id: 'media-player',
    fileExtensions: ['.mp3', '.m4a', '.wav', '.ogg', '.aac', '.flac'],
    component: RecordingTab,
    title: 'Audio Player',
    source: 'recording',
    capabilities: ['recording', 'media', 'audio'],
  })

  context.registerTabKind({
    kind: 'recording',
    component: MediaRecordingTab,
    title: 'Record Audio',
  })
}

export default definePlugin({
  activate,
})
