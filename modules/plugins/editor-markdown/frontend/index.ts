/**
 * Frontend entrypoint for the Markdown Editor plugin.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import MarkdownEditor from './MarkdownEditor.vue'
import {
  createWikilinkExtension,
  type WikilinkOption,
  type WikilinkOptions,
} from './extensions/wikilinks'

export { MarkdownEditor, createWikilinkExtension }
export type { WikilinkOption, WikilinkOptions }

export default definePlugin({
  activate(context: PluginContext) {
    context.registerReader({
      id: 'editor-markdown',
      fileExtensions: ['.md', '.txt', '.markdown'],
      component: MarkdownEditor,
      title: 'Markdown Editor',
    })
  },
})
