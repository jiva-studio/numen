/**
 * The details of a row of the query completion, shown only under the selected row: the description of
 * the entry in plain words, and its example in the type of code when the example is short.
 */
import { completionStatus, type Completion } from '@codemirror/autocomplete'
import type { EditorState, Extension } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import type { QueryCompletion } from '../lib/queryLanguage'

const SHORT_EXAMPLE = 48

const isQueryCompletion = (completion: Completion): completion is QueryCompletion =>
  'entry' in completion

export function createQueryOptionText(
  completion: Completion,
  _state: EditorState,
  view: EditorView,
): HTMLElement | null {
  if (!isQueryCompletion(completion)) return null
  const { description, example } = completion.entry
  if (description === '' && example === '') return null
  const page = view.dom.ownerDocument
  const text = page.createElement('div')
  text.className = 'numen-query-option'
  if (description !== '') {
    const line = page.createElement('span')
    line.textContent = description
    text.append(line)
  }
  if (example !== '' && example.length <= SHORT_EXAMPLE) {
    const code = page.createElement('code')
    code.textContent = example
    text.append(code)
  }
  return text
}

/** The renderer of the details for the completion list, and the extension that shows the selected row's. */
export interface QueryOptionDetails {
  readonly render: (completion: Completion, state: EditorState, view: EditorView) => Node | null
  readonly extension: Extension
}

const isSelectedRow = (details: HTMLElement): boolean =>
  details.closest('li')?.getAttribute('aria-selected') === 'true'

/** The details of each row, hidden in every row but the selected one. */
export function createQueryOptionDetails(): QueryOptionDetails {
  const drawn = new Set<HTMLElement>()
  const render = (completion: Completion, state: EditorState, view: EditorView) => {
    const details = createQueryOptionText(completion, state, view)
    if (!details) return null
    details.hidden = true
    drawn.add(details)
    return details
  }
  const extension = EditorView.updateListener.of((update) => {
    if (completionStatus(update.state) === null) {
      drawn.clear()
      return
    }
    for (const details of drawn) details.hidden = !isSelectedRow(details)
  })
  return { render, extension }
}
