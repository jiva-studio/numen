import { describe, expect, it } from 'vitest'
import type { QueryCompletion } from '../lib/queryLanguage'
import { EditorView } from '@codemirror/view'
import { createQueryOptionText } from './queryOption'

const view = new EditorView({ doc: '' })

const ENTRY: QueryCompletion['entry'] = {
  name: 'rerank',
  kind: 'stage',
  summary: 'reorder the best results with a reranking model',
  description: 'Reorders the best results with a reranking model.',
  example: 'meaning("heat").rerank()',
}

describe('createQueryOptionText', () => {
  it('writes the description and then a short example in the row', () => {
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
