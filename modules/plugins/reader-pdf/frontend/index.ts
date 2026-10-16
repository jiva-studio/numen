/**
 * Frontend entrypoint for the PDF Reader plugin.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import DocumentTab from './DocumentTab.vue'
import { setPdfPluginActive } from '@/pages/document-viewer/model/pluginState'

export { DocumentTab }

export function activate(_context?: PluginContext): void {
  setPdfPluginActive(true)
}

export function deactivate(): void {
  setPdfPluginActive(false)
}

export default definePlugin({
  activate,
  deactivate,
})
