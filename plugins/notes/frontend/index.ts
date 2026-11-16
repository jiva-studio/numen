/**
 * Frontend entrypoint of the notes plugin.
 */
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import MarkdownNoteEditor from './ui/MarkdownNoteEditor.vue'
import { createNoteEditorView } from './ui/noteEditorView'
import { createWikilinkExtension, type WikilinkOption, type WikilinkOptions } from './lib/wikilinks'
import { parseFrontmatter, type FrontmatterResult } from './lib/frontmatter'
import { editorPreferences } from './model/preferences'

export { MarkdownNoteEditor, createWikilinkExtension, parseFrontmatter }
export type { WikilinkOption, WikilinkOptions, FrontmatterResult }

export function activate(plugin: PluginContext, host: HostServices): void {
  editorPreferences.value = plugin.settings.getEditor()
  plugin.settings.onEditorChange((settings) => {
    editorPreferences.value = settings
  })

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

  if (host.vault.notes) plugin.register.view(createNoteEditorView(host.vault.notes))
}

export function deactivate(): void {
  // Teardown hook
}

export default definePlugin({
  activate,
  deactivate,
})
