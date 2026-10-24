/**
 * Wikilink autocomplete and navigation extension for CodeMirror 6.
 *
 * Scans markdown text for [[target]] patterns and provides autocompletion
 * for vault notes and document section headings.
 */
import type { Extension } from '@codemirror/state'
import {
  autocompletion,
  type Completion,
  type CompletionContext,
  type CompletionResult,
} from '@codemirror/autocomplete'

export interface WikilinkOption {
  readonly title: string
  readonly heading?: string | undefined
  readonly detail?: string | undefined
  readonly type?: 'note' | 'heading' | 'new' | undefined
}

export interface WikilinkOptions {
  search(query: string): Promise<readonly WikilinkOption[]>
  headings?(noteTitle: string): Promise<readonly string[]>
  createNote?(title: string): Promise<string | null>
}

interface WikilinkTarget {
  readonly start: number
  readonly query: string
  readonly hasClosingBrackets: boolean
}

function parseWikilinkTarget(context: CompletionContext): WikilinkTarget | null {
  const line = context.state.doc.lineAt(context.pos)
  const lineText = line.text
  const col = context.pos - line.from

  const before = lineText.slice(0, col)
  const openIndex = before.lastIndexOf('[[')
  if (openIndex === -1) {
    return null
  }

  const query = before.slice(openIndex + 2)
  if (query.includes(']]') || query.includes('\n')) {
    return null
  }

  const after = lineText.slice(col)
  const hasClosingBrackets = after.startsWith(']]')

  return {
    start: line.from + openIndex + 2,
    query,
    hasClosingBrackets,
  }
}

/**
 * Creates a CodeMirror extension providing autocompletion for [[wikilinks]].
 */
export function createWikilinkExtension(options: WikilinkOptions): Extension {
  return autocompletion({
    override: [
      async (context: CompletionContext): Promise<CompletionResult | null> => {
        const target = parseWikilinkTarget(context)
        if (!target) {
          return null
        }

        const isHeadingQuery = target.query.includes('#')
        let suggestions: Completion[]

        if (isHeadingQuery && options.headings) {
          const parts = target.query.split('#')
          const noteTitle = parts[0]?.trim() ?? ''
          const headingPrefix = parts[1]?.trim().toLowerCase() ?? ''
          const headings = await options.headings(noteTitle)

          suggestions = headings
            .filter((h) => h.toLowerCase().includes(headingPrefix))
            .map((h) => ({
              label: noteTitle ? `${noteTitle}#${h}` : `#${h}`,
              detail: 'heading',
              type: 'property',
              apply: (view, completion, from, to) => {
                const insertText = target.hasClosingBrackets
                  ? completion.label
                  : `${completion.label}]]`
                view.dispatch({
                  changes: { from: target.start, to, insert: insertText },
                })
              },
            }))
        } else {
          const matches = await options.search(target.query)
          suggestions = matches.map((m) => {
            const detail = m.detail ?? m.type
            return {
              label: m.title,
              ...(detail ? { detail } : {}),
              type: m.type === 'heading' ? 'property' : 'class',
              apply: (view, completion, from, to) => {
                const insertText = target.hasClosingBrackets
                  ? completion.label
                  : `${completion.label}]]`
                view.dispatch({
                  changes: { from: target.start, to, insert: insertText },
                })
              },
            }
          })
        }

        return {
          from: target.start,
          options: suggestions,
          filter: false,
        }
      },
    ],
  })
}
