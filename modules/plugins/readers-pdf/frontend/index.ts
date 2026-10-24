/**
 * Frontend entrypoint for the PDF Reader plugin.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import DocumentTab from './DocumentTab.vue'

export { DocumentTab }
export * from './types'
export * from './words'
export { documents } from './api/wire'
export { useDocumentReader } from './model/useDocumentReader'
export { useDocumentTab } from './model/useDocumentTab'

export function activate(context: PluginContext): void {
  context.registerReader({
    id: 'reader-pdf',
    fileExtensions: ['.pdf'],
    component: DocumentTab,
    title: 'PDF Reader',
    source: 'book',
  })
}

export function deactivate(): void {
  // Cleanup if needed
}

export default definePlugin({
  activate,
  deactivate,
})
