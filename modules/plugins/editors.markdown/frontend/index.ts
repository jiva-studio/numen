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
