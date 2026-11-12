/**
 * Frontend entrypoint for the PDF Reader plugin.
 */
import { FileText } from '@lucide/vue'
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import DocumentTab from './ui/DocumentTab.vue'
import { bindPDFReaderPreferences } from './model/preferences'

export { DocumentTab }
export * from './types'
export * from './words'
export { documents } from './api/wire'
export { useDocumentReader } from './model/useDocumentReader'
export { useDocumentTab } from './model/useDocumentTab'
export { readPDFReaderPreferences, type PDFReaderPreferences } from './model/preferences'

export function activate(context: PluginContext): void {
  bindPDFReaderPreferences(context)
  context.register.reader({
    id: 'reader-pdf',
    component: DocumentTab,
    title: 'PDF Reader',
    icon: FileText,
  })
}

export function deactivate(): void {
  // Cleanup if needed
}

export default definePlugin({
  activate,
  deactivate,
})
