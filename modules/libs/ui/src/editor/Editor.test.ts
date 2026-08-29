/**
 * What the component does with what it is handed, and what it hands back.
 *
 * The view is found through the DOM. The tests hold the component to its
 * props, its model, its events and what it exposes.
 */
import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { undo } from '@codemirror/commands'
import { EditorSelection } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import Editor from './Editor.vue'
import { marked, type EditorChange } from './change'
import { opening, resolving } from './outside'

type Props = InstanceType<typeof Editor>['$props']
type Exposed = { focus: () => void; measure: () => void; reveal: (line: number) => boolean }

// Nothing here has a size, and the editor measures anyway.
Range.prototype.getClientRects = () =>
  Object.assign([], { item: () => null }) as unknown as DOMRectList
Range.prototype.getBoundingClientRect = () => new DOMRect()

const drawn: { unmount: () => void }[] = []

const editor = (props: Partial<Props> = {}) => {
  const wrapper = mount(Editor, { attachTo: document.body, props })
  drawn.push(wrapper)
  const view = EditorView.findFromDOM(wrapper.element as HTMLElement)
  if (!view) throw new Error('the editor drew no view')
  return { wrapper, view, exposed: wrapper.vm as unknown as Exposed }
}

/** Where the caret is, as a person would say it. */
const caretOf = (view: EditorView) => {
  const line = view.state.doc.lineAt(view.state.selection.main.head)
  return { line: line.number, column: view.state.selection.main.head - line.from }
}

/** What is drawn over a change, and nothing where nothing is. */
const over = (view: EditorView) => {
  const found: { from: number; to: number; mark: string | null }[] = []
  view.state.field(marked).decorations.between(0, view.state.doc.length, (from, to, deco) => {
    found.push({ from, to, mark: (deco.spec as { class?: string }).class ?? null })
  })
  return found
}

const caretAt = (view: EditorView, line: number, column: number) => {
  const at = view.state.doc.line(line).from + column
  view.dispatch({ selection: EditorSelection.single(at) })
}

/** A key pressed wherever the caret is, and the event as the page left it. */
const chord = (on: EventTarget, held: KeyboardEventInit = { ctrlKey: true }) => {
  const key = new KeyboardEvent('keydown', {
    key: 's',
    code: 'KeyS',
    keyCode: 83,
    bubbles: true,
    cancelable: true,
    ...held,
  })
  on.dispatchEvent(key)
  return key
}

afterEach(() => {
  while (drawn.length) drawn.pop()?.unmount()
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

describe('the text', () => {
  it('is what the component was given', () => {
    expect(editor({ modelValue: '# Entropy\n' }).view.state.doc.toString()).toBe('# Entropy\n')
  })

  it('is replaced by text put in from outside', async () => {
    const { wrapper, view } = editor({ modelValue: 'first' })
    await wrapper.setProps({ modelValue: 'second' })
    expect(view.state.doc.toString()).toBe('second')
  })

  it('is given back as it is typed', () => {
    const { wrapper, view } = editor({ modelValue: 'one' })
    view.dispatch({ changes: { from: 3, insert: ' two' }, userEvent: 'input.type' })
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['one two'])
  })

  it('is given back once for one thing typed', () => {
    const { wrapper, view } = editor({ modelValue: 'one' })
    view.dispatch({ changes: { from: 3, insert: ' two' }, userEvent: 'input.type' })
    expect(wrapper.emitted('update:modelValue')).toHaveLength(1)
  })
})

describe('a document put in from outside', () => {
  it('is no step to undo', async () => {
    const { wrapper, view } = editor({ modelValue: 'what was read' })
    await wrapper.setProps({ modelValue: 'what is on disk' })

    undo(view)
    expect(view.state.doc.toString()).toBe('what is on disk')
  })

  it('leaves what was typed before it undoable', () => {
    const { view } = editor({ modelValue: 'one' })
    view.dispatch({ changes: { from: 3, insert: ' two' }, userEvent: 'input.type' })

    undo(view)
    expect(view.state.doc.toString()).toBe('one')
  })

  it('keeps the caret on its line when the lines above it move', async () => {
    const { wrapper, view } = editor({ modelValue: 'one\ntwo\nthree' })
    caretAt(view, 3, 2)

    await wrapper.setProps({ modelValue: 'one, and a good deal more of it\ntwo\nthree' })
    expect(caretOf(view)).toEqual({ line: 3, column: 2 })
  })

  it('puts the caret at the end of a line that no longer reaches it', async () => {
    const { wrapper, view } = editor({ modelValue: 'one\ntwo and more\nthree' })
    caretAt(view, 2, 11)

    await wrapper.setProps({ modelValue: 'one\ntwo\nthree' })
    expect(caretOf(view)).toEqual({ line: 2, column: 3 })
  })

  it('keeps both ends of a selection', async () => {
    const { wrapper, view } = editor({ modelValue: 'one\ntwo\nthree' })
    const anchor = view.state.doc.line(1).from + 1
    const head = view.state.doc.line(3).from + 4
    view.dispatch({ selection: EditorSelection.single(anchor, head) })

    await wrapper.setProps({ modelValue: 'one, and a good deal more of it\ntwo\nthree' })

    const range = view.state.selection.main
    expect(view.state.doc.lineAt(range.anchor).number).toBe(1)
    expect(range.anchor - view.state.doc.line(1).from).toBe(1)
    expect(view.state.doc.lineAt(range.head).number).toBe(3)
    expect(range.head - view.state.doc.line(3).from).toBe(4)
  })
})

describe('what it will not be typed into', () => {
  it('is editable as it stands', () => {
    const { view } = editor({ modelValue: 'one' })
    expect(view.state.readOnly).toBe(false)
    expect(view.state.facet(EditorView.editable)).toBe(true)
  })

  it('is neither editable nor writable when it is read only', () => {
    const { view } = editor({ modelValue: 'one', readonly: true })
    expect(view.state.readOnly).toBe(true)
    expect(view.state.facet(EditorView.editable)).toBe(false)
  })

  it('is closed and opened again while it stands', async () => {
    const { wrapper, view } = editor({ modelValue: 'one' })

    await wrapper.setProps({ readonly: true })
    expect(view.state.readOnly).toBe(true)
    expect(view.state.facet(EditorView.editable)).toBe(false)

    await wrapper.setProps({ readonly: false })
    expect(view.state.readOnly).toBe(false)
    expect(view.state.facet(EditorView.editable)).toBe(true)
  })

  it('holds the text it was given while it is read only', async () => {
    const { wrapper, view } = editor({ modelValue: 'first', readonly: true })
    await wrapper.setProps({ modelValue: 'second' })
    expect(view.state.doc.toString()).toBe('second')
  })
})

describe('a change something other than the reader is making', () => {
  const DOC = 'the cat sat on the mat'
  const CHANGE: EditorChange = { id: 'one', from: 4, to: 7, text: 'dog' }

  it('is drawn for nobody while the component is handed none', () => {
    expect(over(editor({ modelValue: DOC }).view)).toEqual([])
  })

  it('marks the stretch that is about to change', () => {
    const { view } = editor({ modelValue: DOC, change: CHANGE })
    expect(over(view)).toEqual([{ from: 4, to: 7, mark: 'cm-changing' }])
  })

  it('is shown once the text arrives by the ordinary route', async () => {
    const { wrapper, view } = editor({ modelValue: DOC, change: CHANGE })
    await wrapper.setProps({ modelValue: 'the dog sat on the mat' })
    expect(over(view)).toEqual([{ from: 4, to: 7, mark: null }])
  })

  it('is dropped whole when there is no longer a change', async () => {
    const { wrapper, view } = editor({ modelValue: DOC, change: CHANGE })
    await wrapper.setProps({ change: null })
    expect(over(view)).toEqual([])
  })

  it('is drawn where a second change stands instead of where the first did', async () => {
    const { wrapper, view } = editor({ modelValue: DOC, change: CHANGE })
    await wrapper.setProps({ change: { id: 'two', from: 12, to: 14, text: 'under' } })
    expect(over(view)).toEqual([{ from: 12, to: 14, mark: 'cm-changing' }])
  })

  it('writes none of the text itself', async () => {
    const { wrapper, view } = editor({ modelValue: DOC, change: CHANGE })
    await wrapper.setProps({ change: { id: 'two', from: 0, to: 22, text: 'something else' } })
    await wrapper.setProps({ change: null })

    expect(view.state.doc.toString()).toBe(DOC)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('leaves nothing behind when it is never made', async () => {
    const { wrapper, view } = editor({ modelValue: DOC, change: CHANGE })
    await wrapper.setProps({ change: null })

    expect(view.state.doc.toString()).toBe(DOC)
    expect(view.contentDOM.querySelector('.cm-changing')).toBe(null)
  })
})

describe('an address in the text', () => {
  it('is emitted when something follows it', () => {
    const { wrapper, view } = editor({ modelValue: '<https://example.invalid>' })
    view.state.facet(opening)('https://example.invalid')
    expect(wrapper.emitted('open')).toEqual([['https://example.invalid']])
  })

  it('is emitted for nobody until something follows one', () => {
    expect(editor({ modelValue: 'go [there](note.md)' }).wrapper.emitted('open')).toBeUndefined()
  })

  it('is what the caller makes of it', () => {
    const { view } = editor({
      modelValue: 'go [there](note.md)',
      resolve: (address: string) => `vault://${address}`,
    })
    expect(view.state.facet(resolving)('note.md')).toBe('vault://note.md')
  })

  it('stands as it was written where the caller makes nothing of it', () => {
    expect(editor({ modelValue: 'go' }).view.state.facet(resolving)('note.md')).toBe('note.md')
  })
})

describe('what a parent can ask for', () => {
  it('is the caret', () => {
    const { view, exposed } = editor({ modelValue: 'one' })
    exposed.focus()
    expect(document.activeElement).toBe(view.contentDOM)
  })

  it('is a line, with the caret put on it', () => {
    const { view, exposed } = editor({ modelValue: 'one\ntwo\nthree' })

    expect(exposed.reveal(2)).toBe(true)
    expect(view.state.selection.main.head).toBe(view.state.doc.line(3).from)
  })

  it('is refused by an editor holding no text, which has no line to give', () => {
    const { exposed } = editor({ modelValue: '' })
    expect(exposed.reveal(2)).toBe(false)
  })

  it('is a measurement, for an editor that was drawn out of sight', () => {
    const measure = vi.spyOn(EditorView.prototype, 'requestMeasure')
    const { exposed } = editor({ modelValue: 'one' })

    const before = measure.mock.calls.length
    exposed.measure()
    expect(measure.mock.calls.length).toBe(before + 1)
  })
})

describe('the chord that keeps the text', () => {
  it('reaches the outside', () => {
    const { wrapper, view } = editor({ modelValue: 'one' })
    chord(view.contentDOM)
    expect(wrapper.emitted('save')).toEqual([[]])
  })

  it('is answered by the editor, so nothing behind it hears the key', () => {
    const { view } = editor({ modelValue: 'one' })
    expect(chord(view.contentDOM).defaultPrevented).toBe(true)
  })

  it('leaves the text as it was', () => {
    const { wrapper, view } = editor({ modelValue: 'one' })
    chord(view.contentDOM)
    expect(view.state.doc.toString()).toBe('one')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('is the chord and not the letter in it', () => {
    const { wrapper, view } = editor({ modelValue: 'one' })
    chord(view.contentDOM, {})
    expect(wrapper.emitted('save')).toBeUndefined()
  })

  it('is heard by the editor the caret is in and by no other', () => {
    const one = editor({ modelValue: 'one' })
    const two = editor({ modelValue: 'two' })

    chord(one.view.contentDOM)
    expect(one.wrapper.emitted('save')).toEqual([[]])
    expect(two.wrapper.emitted('save')).toBeUndefined()
  })

  it('is heard while the text will not be typed into', () => {
    const { wrapper, view } = editor({ modelValue: 'one', readonly: true })
    chord(view.contentDOM)
    expect(wrapper.emitted('save')).toEqual([[]])
  })

  it('is heard by no editor while the caret is in none of them', () => {
    const { wrapper } = editor({ modelValue: 'one' })
    chord(document.body)
    expect(wrapper.emitted('save')).toBeUndefined()
  })
})
