import {
  definePlugin,
  type FileSelectEvent,
  type PluginContext,
  type VaultChangeEvent,
} from '@numen/plugin-sdk'
import PlexGraphView from './PlexGraphView.vue'
import { setPlexPluginActive } from '@/pages/plex-graph/model/pluginState'

export { PlexGraphView, PlexGraphView as PlexTab }
export * from './types'
export * from './words'

export function activate(context: PluginContext): void {
  setPlexPluginActive(true)

  context.registerTabKind({
    kind: 'plex-graph',
    component: PlexGraphView,
    title: 'Knowledge Graph',
  })

  context.onVaultChange((_event: VaultChangeEvent) => {
    // Vault change handling
  })

  context.onFileSelect((_event: FileSelectEvent) => {
    // File selection handling
  })
}

export function deactivate(): void {
  setPlexPluginActive(false)
}

export default definePlugin({
  activate,
  deactivate,
})
