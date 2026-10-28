/**
 * Frontend entrypoint of the notes plugin.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import MarkdownNoteEditor from './MarkdownNoteEditor.vue'
import { createNoteEditorView } from './view/noteEditorView'
import {
  createWikilinkExtension,
  type WikilinkOption,
  type WikilinkOptions,
} from './extensions/wikilinks'
import { parseFrontmatter, type FrontmatterResult } from './frontmatter'
import {
  bindMarkdownEditorPreferences,
  readMarkdownEditorPreferences,
  type MarkdownEditorPreferences,
} from './preferences'

export {
  MarkdownNoteEditor,
  createWikilinkExtension,
  parseFrontmatter,
  readMarkdownEditorPreferences,
}
export type { WikilinkOption, WikilinkOptions, FrontmatterResult, MarkdownEditorPreferences }

export function activate(context: PluginContext): void {
  const preferences = bindMarkdownEditorPreferences(context)

  context.registerReader({
    id: 'markdown',
    component: MarkdownNoteEditor,
    title: 'Markdown Note Editor',
  })

  context.registerTabKind({
    kind: 'note',
    component: MarkdownNoteEditor,
    title: 'Note',
  })

  if (context.notes) context.registerView(createNoteEditorView(context.notes, preferences))
}

export function deactivate(): void {
  // Teardown hook
}

export default definePlugin({
  activate,
  deactivate,
})
