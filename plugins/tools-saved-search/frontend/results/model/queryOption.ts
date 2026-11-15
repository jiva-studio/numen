/**
 * The second line of a row of the query completion: the description of the entry in plain words, and its
 * example in the type of code when the example is short.
 */
import type { Completion } from '@codemirror/autocomplete'
import type { EditorState } from '@codemirror/state'
import type { EditorView } from '@codemirror/view'
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
