import {
  definePlugin,
  type FileSelectEvent,
  type PluginContext,
  type VaultChangeEvent,
} from '@numen/plugin-sdk'
import PlexGraphView from './PlexGraphView.vue'

export { PlexGraphView, PlexGraphView as PlexTab }
export { usePlexTab } from './model/usePlexTab'
export { usePlexView } from './model/usePlexView'
export { ITEMS, NONE, OFFERED, NEW_NOTE } from './lib/menu'
export * from './types'
export * from './words'

export function activate(context: PluginContext): void {
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
  // Cleanup if needed
}

export default definePlugin({
  activate,
  deactivate,
})
