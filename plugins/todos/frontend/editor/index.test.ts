import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { markdown } from '@codemirror/lang-markdown'
import { GFM } from '@lezer/markdown'
import { completionStatus, currentCompletions } from '@codemirror/autocomplete'
import { createTaskEditorExtension } from './index'

let view: EditorView | undefined

beforeEach(() => {
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
  view?.destroy()
  view = undefined
})

const wait = (ms: number) => vi.advanceTimersByTimeAsync(ms)

const FIELDS = {
  projects: [{ name: 'work', count: 2 }],
  contexts: [{ name: 'home', count: 1 }],
  tags: [{ name: 'river', count: 1 }],
}

function open(doc: string, late = false): EditorView {
  let fields = late ? { projects: [], contexts: [], tags: [] } : FIELDS
  const extension = createTaskEditorExtension({
    api: {} as never,
    getToday: () => '2026-11-22',
    getFields: () => fields,
    loadFields: async () => {
      await Promise.resolve()
      fields = FIELDS
    },
    hideFields: false,
    showError: () => undefined,
  })
  const parent = document.createElement('div')
  document.body.appendChild(parent)
  view = new EditorView({
    parent,
    state: EditorState.create({
      doc,
      selection: { anchor: doc.length },
      extensions: [markdown({ extensions: GFM }), extension],
    }),
  })
  return view
}

async function type(editor: EditorView, text: string): Promise<string[]> {
  const at = editor.state.doc.length
  editor.dispatch({
    changes: { from: at, insert: text },
    selection: { anchor: at + text.length },
    userEvent: 'input.type',
  })
  await wait(400)
  return currentCompletions(editor.state).map((one) => one.label)
}

describe('the task editor extension in a markdown editor', () => {
  it('offers the projects on the first plus when the fields were not loaded yet', async () => {
    const editor = open('- [ ] pay ', true)

    expect(await type(editor, '+')).toStrictEqual(['work'])
    expect(completionStatus(editor.state)).toBe('active')
  })

  it('offers the projects once a plus is typed on a task line', async () => {
    const editor = open('- [ ] pay ')

    expect(await type(editor, '+')).toStrictEqual(['work'])
    expect(completionStatus(editor.state)).toBe('active')
  })

  it('offers the contexts after an at sign and the tags after a hash', async () => {
    expect(await type(open('- [ ] pay '), '@')).toStrictEqual(['home'])
    view?.destroy()
    expect(await type(open('- [ ] pay '), '#')).toStrictEqual(['river'])
  })

  it('offers nothing on a line that is not a task', async () => {
    expect(await type(open('pay '), '+')).toStrictEqual([])
  })
})
