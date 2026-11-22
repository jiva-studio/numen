/**
 * Frontend entrypoint of the notes plugin.
 */
import { createNamedIcon } from '@numen/ui'
import { definePlugin, type HostServices, type PluginContext } from '@numen/plugin-sdk'
import MarkdownNoteEditor from './ui/MarkdownNoteEditor.vue'
import { createNoteEditorView } from './ui/noteEditorView'
import { createWikilinkExtension, type WikilinkOption, type WikilinkOptions } from './lib/wikilinks'
import { parseFrontmatter, type FrontmatterResult } from './lib/frontmatter'
import { editorPreferences } from './model/preferences'
import { followPluginExtensions } from './model/pluginExtensions'

export { MarkdownNoteEditor, createWikilinkExtension, parseFrontmatter }
export type { WikilinkOption, WikilinkOptions, FrontmatterResult }

export function activate(plugin: PluginContext, host: HostServices): void {
  editorPreferences.value = plugin.settings.getEditor()
  plugin.settings.onEditorChange((settings) => {
    editorPreferences.value = settings
  })
  if (host.editor) followPluginExtensions(host.editor)
  const icon = createNamedIcon('file-text')

  plugin.register.reader({
    id: 'markdown',
    component: MarkdownNoteEditor,
    title: 'Markdown Note Editor',
    icon,
  })

  plugin.register.tabKind({
    kind: 'note',
    component: MarkdownNoteEditor,
    title: 'Note',
    icon,
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
