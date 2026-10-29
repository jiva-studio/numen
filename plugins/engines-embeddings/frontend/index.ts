/**
 * Frontend entrypoint for the Vector Embeddings plugin: the search relevance settings view.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import FloorPanel from './floor/ui/FloorPanel.vue'

export { FloorPanel }
export { useFloor } from './floor/model/useFloor'

export function activate(context: PluginContext): void {
  context.registerSettingsView({ id: 'floor', component: FloorPanel })
}

export function deactivate(): void {
  // The registered view is released with the plugin context.
}

export default definePlugin({
  activate,
  deactivate,
})
