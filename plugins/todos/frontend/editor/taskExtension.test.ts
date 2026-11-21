import { afterEach, describe, expect, it, vi } from 'vitest'
import { EditorState } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { editorNotePath } from '@numen/plugin-sdk'
import { createTaskExtension, type TickRequest } from './taskExtension'

let view: EditorView | undefined

afterEach(() => {
  view?.destroy()
  view = undefined
})

function open(doc: string, path = 'Projects/Home.md') {
  const tick = vi.fn<(request: TickRequest) => Promise<void>>(async () => undefined)
  view = new EditorView({
    state: EditorState.create({
      doc,
      extensions: [editorNotePath.of(path), createTaskExtension({ tick })],
    }),
    parent: document.body,
  })
  return { view, tick }
}

const boxes = (root: EditorView) => [...root.dom.querySelectorAll<HTMLInputElement>('.cm-todo-box')]

describe('the task extension', () => {
  it('draws a checkbox in place of the marker and the box of a task line, ticked when it is done', () => {
    const { view: editor } = open('- [ ] write\n- [x] send\nplain')

    const drawn = boxes(editor)
    expect(drawn.map((one) => one.checked)).toEqual([false, true])
    expect(editor.dom.querySelector('.cm-content')?.textContent).toBe('writesendplain')
  })

  it('draws nothing in a fenced code block', () => {
    const { view: editor } = open('```\n- [ ] x\n```\n- [ ] y')

    expect(boxes(editor)).toHaveLength(1)
  })

  it('marks the fields and dims a completed task', () => {
    const { view: editor } = open(
      '- [x] (A) pay due:2026-11-20 +home @bank rec:1m pri:A done:2026-11-19',
    )

    const has = (name: string) => editor.dom.querySelector(`.${name}`)?.textContent
    expect(has('cm-todo-priority')).toBe('(A)')
    expect(has('cm-todo-due')).toBe('due:2026-11-20')
    expect(has('cm-todo-project')).toBe('+home')
    expect(has('cm-todo-context')).toBe('@bank')
    expect(has('cm-todo-recurrence')).toBe('rec:1m')
    expect(has('cm-todo-done-on')).toBe('done:2026-11-19')
    expect(editor.dom.querySelector('.cm-todo-line-done')).not.toBeNull()
  })

  it('does not dim an open task', () => {
    const { view: editor } = open('- [ ] pay')

    expect(editor.dom.querySelector('.cm-todo-line-done')).toBeNull()
  })

  it('follows the text as it changes', () => {
    const { view: editor } = open('plain')

    editor.dispatch({ changes: { from: 0, insert: '- [ ] ' } })

    expect(boxes(editor)).toHaveLength(1)
  })

  it('ticks the task of the line with the line number and its text when the box is clicked', () => {
    const { view: editor, tick } = open('intro\n- [ ] write due:2026-11-20')

    boxes(editor)[0]?.click()

    expect(tick).toHaveBeenCalledWith({
      path: 'Projects/Home.md',
      line: 2,
      raw: '- [ ] write due:2026-11-20',
      isDone: true,
    })
  })

  it('unticks a done task when its box is clicked', () => {
    const { view: editor, tick } = open('- [x] write')

    boxes(editor)[0]?.click()

    expect(tick).toHaveBeenCalledWith(expect.objectContaining({ line: 1, isDone: false }))
  })

  it('leaves the box off where the editor holds no note of the vault', () => {
    const { view: editor } = open('- [ ] write', '')

    expect(boxes(editor)[0]?.disabled).toBe(true)
  })
})
