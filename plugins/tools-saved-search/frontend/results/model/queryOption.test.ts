import { afterEach, describe, expect, it, vi } from 'vitest'
import { autocompletion, moveCompletionSelection, startCompletion } from '@codemirror/autocomplete'
import { EditorView } from '@codemirror/view'
import {
  createQueryCompletion,
  type QueryCompletion,
  type QueryFunction,
} from '../lib/queryLanguage'
import { queryTheme } from '../lib/queryTheme'
import { createQueryOptionDetails } from './queryOption'
import { createQueryOptionText } from './queryOptionText'

const view = new EditorView({ doc: '' })

const ENTRY: QueryCompletion['entry'] = {
  name: 'rerank',
  kind: 'stage',
  summary: 'reorder the best results with a reranking model',
  description: 'Reorders the best results with a reranking model.',
  example: 'meaning("heat").rerank()',
}

const CATALOG: readonly QueryFunction[] = [
  {
    name: 'words',
    kind: 'function',
    summary: 'passages with these words',
    description: 'Finds passages that contain the words you give.',
    example: 'words("seed tray")',
  },
  {
    name: 'hasTag',
    kind: 'function',
    summary: 'notes with this tag',
    description: 'Finds notes that carry the tag you give.',
    example: 'hasTag("garden")',
  },
]

const openEditors: EditorView[] = []

afterEach(() => {
  openEditors.splice(0).forEach((editor) => {
    editor.destroy()
    editor.dom.remove()
  })
})

const openCompletion = async (): Promise<{ editor: EditorView; rows: HTMLElement[] }> => {
  const parent = document.createElement('div')
  document.body.append(parent)
  const details = createQueryOptionDetails()
  const editor = new EditorView({
    parent,
    doc: '',
    extensions: [
      queryTheme,
      autocompletion({
        override: [createQueryCompletion(() => CATALOG)],
        addToOptions: [{ render: details.render, position: 90 }],
        icons: false,
      }),
      details.extension,
    ],
  })
  openEditors.push(editor)
  editor.focus()
  startCompletion(editor)
  const rows = await vi.waitFor(() => {
    const found = [...editor.dom.querySelectorAll<HTMLElement>('.cm-tooltip-autocomplete li')]
    if (found.length !== CATALOG.length) throw new Error('the completion list is not open')
    return found
  })
  return { editor, rows }
}

const isShown = (element: HTMLElement | null): boolean => element !== null && !element.hidden

describe('the completion list', () => {
  it('draws each row as the name and its summary', async () => {
    const { rows } = await openCompletion()

    expect(
      rows.map((row) => [
        row.querySelector('.cm-completionLabel')?.textContent,
        row.querySelector('.cm-completionDetail')?.textContent,
      ]),
    ).toStrictEqual([
      ['hasTag', 'notes with this tag'],
      ['words', 'passages with these words'],
    ])
  })

  it('shows the description and example under the selected row only', async () => {
    const { editor, rows } = await openCompletion()

    expect(
      rows.map((row) => isShown(row.querySelector<HTMLElement>('.numen-query-option'))),
    ).toStrictEqual([true, false])
    expect(rows[0]!.querySelector('.numen-query-option')?.textContent).toContain(
      'Finds notes that carry the tag you give.',
    )

    await vi.waitFor(() => expect(moveCompletionSelection(true)(editor)).toBe(true))

    await vi.waitFor(() =>
      expect(
        rows.map((row) => isShown(row.querySelector<HTMLElement>('.numen-query-option'))),
      ).toStrictEqual([false, true]),
    )
  })
})

describe('createQueryOptionText', () => {
  it('writes the description and then a short example', () => {
    const option: QueryCompletion = { label: 'rerank', entry: ENTRY }
    const text = createQueryOptionText(option, view.state, view)

    expect(text?.querySelector('span')?.textContent).toBe(
      'Reorders the best results with a reranking model.',
    )
    expect(text?.querySelector('code')?.textContent).toBe('meaning("heat").rerank()')
  })

  it('leaves out an example too long for the row', () => {
    const example = `meaning("${'heat '.repeat(20)}").rerank()`
    const option: QueryCompletion = { label: 'rerank', entry: { ...ENTRY, example } }
    const text = createQueryOptionText(option, view.state, view)

    expect(text?.querySelector('code')).toBeNull()
  })

  it('draws nothing for an option of another source', () => {
    expect(createQueryOptionText({ label: 'other' }, view.state, view)).toBeNull()
  })
})
