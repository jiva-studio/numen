/**
 * Frontend entrypoint for the Media Downloader plugin.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import MediaUrlTab from './MediaUrlTab.vue'

export { MediaUrlTab }
export * from './words'
export * from './client'

export function activate(context: PluginContext): void {
  context.registerTabKind({
    kind: 'url',
    component: MediaUrlTab,
    title: 'Download Media',
  })
  context.registerCommand({
    id: 'downloader.open',
    title: 'Media: Download from URL',
    handler: () => {
      void context.openTab?.('url')
    },
  })
}

export function deactivate(): void {
  // Teardown hook
}

export default definePlugin({
  activate,
  deactivate,
})
