/**
 * Wikilink autocomplete for CodeMirror 6.
 *
 * Typing '[[' opens note and heading suggestions. Selecting an entry replaces
 * the trigger with a complete link and ensures closing brackets.
 */
import { autocompletion, type Completion, type CompletionContext, type CompletionResult } from '@codemirror/autocomplete'
import { syntaxTree } from '@codemirror/language'
import type { Extension } from '@codemirror/state'
import type { EditorView } from '@codemirror/view'
import type { SyntaxNode } from '@lezer/common'

export interface WikilinkOption {
  readonly title: string
  readonly heading?: string | undefined
  readonly detail?: string | undefined
  readonly isNew?: boolean | undefined
  readonly type?: 'note' | 'heading' | 'new' | undefined
}

export interface WikilinkCompletionOptions {
  search(query: string): Promise<readonly WikilinkOption[]>
  headings?(noteTitle: string): Promise<readonly string[]>
  createNote?(title: string): Promise<string | null>
}

const isCode = (node: SyntaxNode): boolean => {
  for (let one: SyntaxNode | null = node; one; one = one.parent) {
    if (one.name === 'FencedCode' || one.name === 'CodeBlock' || one.name === 'InlineCode') {
      return true
    }
  }
  return false
}

export function createWikilinkSource(options: WikilinkCompletionOptions) {
  return async (context: CompletionContext): Promise<CompletionResult | null> => {
    const innermost = syntaxTree(context.state).resolveInner(context.pos, 1)
    if (isCode(innermost)) return null

    const line = context.state.doc.lineAt(context.pos)
    const lineBefore = line.text.slice(0, context.pos - line.from)
    const match = /\[\[([^\]\n]*)$/.exec(lineBefore)
    if (!match) return null

    const rawQuery = match[1] ?? ''
    const bracketStart = line.from + match.index
    const lineAfter = line.text.slice(context.pos - line.from)
    const hasClosingBrackets = lineAfter.startsWith(']]')
    const replaceTo = hasClosingBrackets ? context.pos + 2 : context.pos

    const hashIndex = rawQuery.indexOf('#')
    if (hashIndex >= 0) {
      const notePart = rawQuery.slice(0, hashIndex).trim()
      const headingQuery = rawQuery.slice(hashIndex + 1).trim().toLowerCase()
      const headingsList = options.headings ? await options.headings(notePart) : []

      const matchedHeadings = headingsList.filter((h) =>
        h.toLowerCase().includes(headingQuery),
      )

      const completions: Completion[] = matchedHeadings.map((heading) => {
        const target = notePart ? `${notePart}#${heading}` : `#${heading}`
        const insertText = `[[${target}]]`
        return {
          label: target,
          detail: 'heading',
          type: 'heading',
          apply: (view: EditorView) => {
            view.dispatch({
              changes: { from: bracketStart, to: replaceTo, insert: insertText },
              selection: { anchor: bracketStart + insertText.length },
            })
          },
        }
      })

      return {
        from: bracketStart + 2,
        options: completions,
        filter: false,
      }
    }

    const trimmedQuery = rawQuery.trim()
    const results = await options.search(trimmedQuery)

    const completions: Completion[] = results.map((item) => {
      const target = item.heading ? `${item.title}#${item.heading}` : item.title
      const insertText = `[[${target}]]`
      return {
        label: item.heading ? `${item.title}#${item.heading}` : item.title,
        detail: item.detail ?? (item.heading ? 'heading' : 'note'),
        type: item.type ?? (item.heading ? 'heading' : 'note'),
        boost: 1,
        apply: (view: EditorView) => {
          view.dispatch({
            changes: { from: bracketStart, to: replaceTo, insert: insertText },
            selection: { anchor: bracketStart + insertText.length },
          })
        },
      }
    })

    if (options.createNote && trimmedQuery) {
      const exists = results.some(
        (r) => r.title.toLowerCase() === trimmedQuery.toLowerCase() && !r.heading,
      )
      if (!exists) {
        const insertText = `[[${trimmedQuery}]]`
        completions.push({
          label: `Create "${trimmedQuery}"`,
          detail: 'new note',
          type: 'new',
          boost: -1,
          apply: (view: EditorView) => {
            view.dispatch({
              changes: { from: bracketStart, to: replaceTo, insert: insertText },
              selection: { anchor: bracketStart + insertText.length },
            })
            void options.createNote?.(trimmedQuery)
          },
        })
      }
    }

    return {
      from: bracketStart + 2,
      options: completions,
      filter: false,
    }
  }
}

export function createWikilinkCompletion(options: WikilinkCompletionOptions): Extension {
  return autocompletion({
    override: [createWikilinkSource(options)],
    icons: false,
  })
}
