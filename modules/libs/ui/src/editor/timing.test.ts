/**
 * A time against every line, and the line being read now.
 *
 * The gutter and the decoration are read out of the view the extension was put
 * in, which is what a person sees.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { EditorState, type Extension } from '@codemirror/state'
import { EditorView } from '@codemirror/view'
import { timing, type Timed } from './timing'

// Nothing here has a size, and the editor measures anyway.
Range.prototype.getClientRects = () =>
  Object.assign([], { item: () => null }) as unknown as DOMRectList
Range.prototype.getBoundingClientRect = () => new DOMRect()

const drawn: EditorView[] = []

const TEXT = 'A bell over the door.\nRain on the awning.\nSomeone counting change.'

const TIMED: Timed = { times: ['0:01', '0:03', '0:06'], now: -1, follows: false }

const editor = (goes: (line: number) => void = () => {}) => {
  const heard = timing(goes)
  return { heard, view: drawing(heard) }
}

/** One more editor these times are shown in. */
const drawing = (heard: { extension: Extension }) => {
  const view = new EditorView({
    parent: document.body,
    state: EditorState.create({ doc: TEXT, extensions: [heard.extension] }),
  })
  drawn.push(view)
  return view
}

/** Everything queued as an editor attached has run. */
const attached = () => new Promise((done) => setTimeout(done, 0))

/** What stands in the gutter, line by line. */
const gutter = (view: EditorView): string[] =>
  [...view.dom.querySelectorAll('.cm-times .cm-gutterElement')].map((one) => one.textContent ?? '')

/** The lines drawn as being read. */
const reading = (view: EditorView): string[] =>
  [...view.dom.querySelectorAll('.cm-content .cm-reading')].map((one) => one.textContent ?? '')

afterEach(() => {
  for (const view of drawn.splice(0)) view.destroy()
})

describe('the times of an editor', () => {
  it('stand in the gutter, one to a line', () => {
    const { heard, view } = editor()

    heard.show(TIMED)

    expect(gutter(view)).toStrictEqual(['0:01', '0:03', '0:06'])
  })

  it('stand nowhere until they are given', () => {
    const { view } = editor()

    expect(gutter(view)).toStrictEqual([])
  })

  it('are replaced by the ones given next', () => {
    const { heard, view } = editor()
    heard.show(TIMED)

    heard.show({ ...TIMED, times: ['1:00:01', '1:00:03', '1:00:06'] })

    expect(gutter(view)).toStrictEqual(['1:00:01', '1:00:03', '1:00:06'])
  })

  it('say which line was clicked', () => {
    const asked: number[] = []
    const { heard, view } = editor((line) => asked.push(line))
    heard.show(TIMED)

    const marks = view.dom.querySelectorAll('.cm-times .cm-time')
    marks[1]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))

    expect(asked).toStrictEqual([1])
  })
})

describe('the line being read', () => {
  it('is drawn as the one being read, and no other is', () => {
    const { heard, view } = editor()

    heard.show({ ...TIMED, now: 1 })

    expect(reading(view)).toStrictEqual(['Rain on the awning.'])
  })

  it('is no line where none is being read', () => {
    const { heard, view } = editor()
    heard.show({ ...TIMED, now: 1 })

    heard.show({ ...TIMED, now: -1 })

    expect(reading(view)).toStrictEqual([])
  })

  it('is no line where the document is shorter than the one asked for', () => {
    const { heard, view } = editor()

    heard.show({ ...TIMED, now: 9 })

    expect(reading(view)).toStrictEqual([])
  })

  it('is drawn whether or not the view follows it', () => {
    const { heard, view } = editor()

    heard.show({ ...TIMED, now: 2, follows: true })
    expect(reading(view)).toStrictEqual(['Someone counting change.'])

    heard.show({ ...TIMED, now: 2, follows: false })
    expect(reading(view)).toStrictEqual(['Someone counting change.'])
  })

  it('is marked in the gutter as well as in the words', () => {
    const { heard, view } = editor()

    heard.show({ ...TIMED, now: 0 })

    const marks = [...view.dom.querySelectorAll('.cm-times .cm-gutterElement.cm-reading .cm-time')]
    expect(marks.map((one) => one.textContent)).toStrictEqual(['0:01'])
  })
})

describe('an editor these are shown in', () => {
  it('is still the text it holds, and is still typed into', () => {
    const { heard, view } = editor()
    heard.show({ ...TIMED, now: 1 })

    view.dispatch({ changes: { from: 0, to: 0, insert: 'Then. ' } })

    expect(view.state.doc.toString().startsWith('Then. A bell')).toBe(true)
    expect(gutter(view)).toStrictEqual(['0:01', '0:03', '0:06'])
  })
})

describe('an editor drawn a second time', () => {
  it('is shown what the one before it was shown', async () => {
    const heard = timing(() => {})
    const first = drawing(heard)
    heard.show({ ...TIMED, now: 1 })
    first.destroy()

    const again = drawing(heard)
    await attached()

    expect(gutter(again)).toStrictEqual(['0:01', '0:03', '0:06'])
    expect(reading(again)).toStrictEqual(['Rain on the awning.'])
  })

  it('is shown them where nothing was drawn when they were given', async () => {
    const heard = timing(() => {})
    heard.show(TIMED)

    const view = drawing(heard)
    await attached()

    expect(gutter(view)).toStrictEqual(['0:01', '0:03', '0:06'])
  })

  it('is the one shown what comes next, and the one before it is left alone', async () => {
    const heard = timing(() => {})
    const first = drawing(heard)
    heard.show(TIMED)
    const again = drawing(heard)
    await attached()

    heard.show({ ...TIMED, times: ['9:01', '9:03', '9:06'] })

    expect(gutter(again)).toStrictEqual(['9:01', '9:03', '9:06'])
    expect(gutter(first)).toStrictEqual(['0:01', '0:03', '0:06'])
  })

  it('goes on being shown them once the one before it is destroyed', () => {
    const heard = timing(() => {})
    const first = drawing(heard)
    const again = drawing(heard)

    first.destroy()
    heard.show(TIMED)

    expect(gutter(again)).toStrictEqual(['0:01', '0:03', '0:06'])
  })
})

describe('the times in the gutter', () => {
  it('are not walked over by the tab key on the way to the words', () => {
    const { heard, view } = editor()

    heard.show(TIMED)

    const marks = [...view.dom.querySelectorAll<HTMLElement>('.cm-times .cm-time')]
    expect(marks.length).toBe(3)
    expect(marks.every((one) => one.tabIndex === -1)).toBe(true)
  })
})
