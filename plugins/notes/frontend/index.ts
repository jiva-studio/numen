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

export { MarkdownNoteEditor, createWikilinkExtension, parseFrontmatter }
export type { WikilinkOption, WikilinkOptions, FrontmatterResult }

export interface MarkdownEditorPreferences {
  softWrap?: boolean
  lineNumbers?: boolean
  tabSize?: number
}

export function readMarkdownEditorPreferences(
  getSetting?: (path: readonly string[]) => unknown,
): MarkdownEditorPreferences {
  return {
    softWrap: (getSetting?.(['soft_wrap']) as boolean | undefined) ?? true,
    lineNumbers: (getSetting?.(['line_numbers']) as boolean | undefined) ?? true,
    tabSize: (getSetting?.(['tab_size']) as number | undefined) ?? 2,
  }
}

export function activate(context: PluginContext): void {
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

  if (context.notes) context.registerView(createNoteEditorView(context.notes))
}

export function deactivate(): void {
  // Teardown hook
}

export default definePlugin({
  activate,
  deactivate,
})
