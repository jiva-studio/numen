/**
 * Frontend entrypoint of the notes plugin.
 */
import { definePlugin, type PluginContext } from '@numen/plugin-sdk'
import MarkdownNoteEditor from './ui/MarkdownNoteEditor.vue'
import { createNoteEditorView } from './ui/noteEditorView'
import { createWikilinkExtension, type WikilinkOption, type WikilinkOptions } from './lib/wikilinks'
import { parseFrontmatter, type FrontmatterResult } from './lib/frontmatter'
import {
  bindMarkdownEditorPreferences,
  readMarkdownEditorPreferences,
  type MarkdownEditorPreferences,
} from './model/preferences'

export {
  MarkdownNoteEditor,
  createWikilinkExtension,
  parseFrontmatter,
  readMarkdownEditorPreferences,
}
export type { WikilinkOption, WikilinkOptions, FrontmatterResult, MarkdownEditorPreferences }

export function activate(context: PluginContext): void {
  const preferences = bindMarkdownEditorPreferences(context)

  context.register.reader({
    id: 'markdown',
    component: MarkdownNoteEditor,
    title: 'Markdown Note Editor',
  })

  context.register.tabKind({
    kind: 'note',
    component: MarkdownNoteEditor,
    title: 'Note',
  })

  if (context.vault.notes)
    context.register.view(createNoteEditorView(context.vault.notes, preferences))
}

export function deactivate(): void {
  // Teardown hook
}

export default definePlugin({
  activate,
  deactivate,
})
