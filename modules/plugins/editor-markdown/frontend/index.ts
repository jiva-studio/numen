/**
 * Frontend entrypoint for the Markdown Editor plugin.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import MarkdownNoteEditor from './MarkdownNoteEditor.vue'
import {
  createWikilinkExtension,
  type WikilinkOption,
  type WikilinkOptions,
} from './extensions/wikilinks'
import { parseFrontmatter, type FrontmatterResult } from './frontmatter'

export {
  MarkdownNoteEditor,
  MarkdownNoteEditor as MarkdownEditor,
  createWikilinkExtension,
  parseFrontmatter,
}
export type { WikilinkOption, WikilinkOptions, FrontmatterResult }

export function activate(context: PluginContext): void {
  context.registerReader({
    id: 'editor-markdown',
    fileExtensions: ['.md', '.markdown', '.txt', '.note'],
    component: MarkdownNoteEditor,
    title: 'Markdown Note Editor',
    source: 'note',
  })

  context.registerTabKind({
    kind: 'note',
    component: MarkdownNoteEditor,
    title: 'Note',
  })
}

export function deactivate(): void {
  // Teardown hook
}

export default definePlugin({
  activate,
  deactivate,
})
