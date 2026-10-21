import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import AudioPlayerTab from './AudioPlayerTab.vue'
import MediaRecordingTab from './MediaRecordingTab.vue'

export { AudioPlayerTab, MediaRecordingTab }

export function activate(context: PluginContext): void {
  // Register audio file extensions (.mp3, .wav, etc.)
  context.registerReader({
    id: 'media-player',
    fileExtensions: ['.mp3', '.m4a', '.wav', '.ogg', '.aac', '.flac'],
    component: AudioPlayerTab,
    title: 'Audio Player',
    source: 'recording',
    capabilities: ['recording', 'media', 'audio'],
  })

  // Register microphone recording tab
  context.registerTabKind({
    kind: 'recording',
    component: MediaRecordingTab,
    title: 'Record Audio',
  })
}

export default definePlugin({
  activate,
})
