/**
 * Frontend entrypoint for the Search Relevance plugin: the search relevance settings view.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import { FloorPanel } from './floor'

export { FloorPanel, useFloor } from './floor'

export function activate(plugin: PluginContext): void {
  plugin.register.settingsView({ id: 'floor', component: FloorPanel })
}

export function deactivate(): void {
  // The registered view is released with the plugin context.
}

export default definePlugin({
  activate,
  deactivate,
})
