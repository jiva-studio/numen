/**
 * Frontend entrypoint of the notes plugin.
 */
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
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

export function activate(plugin: PluginContext, host: HostServices): void {
  const preferences = bindMarkdownEditorPreferences(plugin)

  plugin.register.reader({
    id: 'markdown',
    component: MarkdownNoteEditor,
    title: 'Markdown Note Editor',
  })

  plugin.register.tabKind({
    kind: 'note',
    component: MarkdownNoteEditor,
    title: 'Note',
  })

  if (host.vault.notes) plugin.register.view(createNoteEditorView(host.vault.notes, preferences))
}

export function deactivate(): void {
  // Teardown hook
}

export default definePlugin({
  activate,
  deactivate,
})
