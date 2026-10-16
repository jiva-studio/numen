/**
 * Frontend entrypoint for the PDF Reader plugin.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import DocumentTab from './DocumentTab.vue'
import { setPdfPluginActive } from '@/pages/document-viewer/model/pluginState'

export { DocumentTab }

export function activate(context?: PluginContext): void {
  setPdfPluginActive(true)
  context?.registerReader({
    id: 'reader-pdf',
    fileExtensions: ['.pdf'],
    component: DocumentTab,
    title: 'PDF Reader',
  })
}

export function deactivate(): void {
  setPdfPluginActive(false)
}

export default definePlugin({
  activate,
  deactivate,
})
