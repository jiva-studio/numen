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

const TIMED: Timed = { times: ['0:01', '0:03', '0:06'], current: -1, following: false }

const editor = (goes: (line: number) => void = () => {}) => {
  const times = timing(goes)
  return { times, view: drawing(times) }
}

/** One more editor these times are shown in. */
const drawing = (times: { extension: Extension }) => {
  const view = new EditorView({
    parent: document.body,
    state: EditorState.create({ doc: TEXT, extensions: [times.extension] }),
  })
  drawn.push(view)
  return view
}

/** Everything queued as an editor attached has run. */
const attached = () => new Promise((done) => setTimeout(done, 0))

/** What stands in the gutter, line by line. */
const gutter = (view: EditorView): string[] =>
  [...view.dom.querySelectorAll('.cm-times .cm-gutterElement')].map((one) => one.textContent ?? '')

/** The lines drawn as being said. */
const current = (view: EditorView): string[] =>
  [...view.dom.querySelectorAll('.cm-content .cm-current')].map((one) => one.textContent ?? '')

afterEach(() => {
  for (const view of drawn.splice(0)) view.destroy()
})

describe('the times of an editor', () => {
  it('stand in the gutter, one to a line', () => {
    const { times, view } = editor()

    times.show(TIMED)

    expect(gutter(view)).toStrictEqual(['0:01', '0:03', '0:06'])
  })

  it('stand nowhere until they are given', () => {
    const { view } = editor()

    expect(gutter(view)).toStrictEqual([])
  })

  it('are replaced by the ones given next', () => {
    const { times, view } = editor()
    times.show(TIMED)

    times.show({ ...TIMED, times: ['1:00:01', '1:00:03', '1:00:06'] })

    expect(gutter(view)).toStrictEqual(['1:00:01', '1:00:03', '1:00:06'])
  })

  it('say which line was clicked', () => {
    const asked: number[] = []
    const { times, view } = editor((line) => asked.push(line))
    times.show(TIMED)

    const marks = view.dom.querySelectorAll('.cm-times .cm-time')
    marks[1]?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))

    expect(asked).toStrictEqual([1])
  })
})

describe('the line being read', () => {
  it('is drawn as the one being read, and no other is', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 1 })

    expect(current(view)).toStrictEqual(['Rain on the awning.'])
  })

  it('is no line where none is being read', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 1 })

    times.show({ ...TIMED, current: -1 })

    expect(current(view)).toStrictEqual([])
  })

  it('is no line where the document is shorter than the one asked for', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 9 })

    expect(current(view)).toStrictEqual([])
  })

  it('is drawn whether or not the view follows it', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 2, following: true })
    expect(current(view)).toStrictEqual(['Someone counting change.'])

    times.show({ ...TIMED, current: 2, following: false })
    expect(current(view)).toStrictEqual(['Someone counting change.'])
  })

  it('is marked in the gutter as well as in the words', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 0 })

    const marks = [...view.dom.querySelectorAll('.cm-times .cm-gutterElement.cm-current .cm-time')]
    expect(marks.map((one) => one.textContent)).toStrictEqual(['0:01'])
  })
})

describe('an editor these are shown in', () => {
  it('is still the text it holds, and is still typed into', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 1 })

    view.dispatch({ changes: { from: 0, to: 0, insert: 'Then. ' } })

    expect(view.state.doc.toString().startsWith('Then. A bell')).toBe(true)
    expect(gutter(view)).toStrictEqual(['0:01', '0:03', '0:06'])
  })
})

describe('an editor drawn a second time', () => {
  it('is shown what the one before it was shown', async () => {
    const times = timing(() => {})
    const first = drawing(times)
    times.show({ ...TIMED, current: 1 })
    first.destroy()

    const again = drawing(times)
    await attached()

    expect(gutter(again)).toStrictEqual(['0:01', '0:03', '0:06'])
    expect(current(again)).toStrictEqual(['Rain on the awning.'])
  })

  it('is shown them where nothing was drawn when they were given', async () => {
    const times = timing(() => {})
    times.show(TIMED)

    const view = drawing(times)
    await attached()

    expect(gutter(view)).toStrictEqual(['0:01', '0:03', '0:06'])
  })

  it('is the one shown what comes next, and the one before it is left alone', async () => {
    const times = timing(() => {})
    const first = drawing(times)
    times.show(TIMED)
    const again = drawing(times)
    await attached()

    times.show({ ...TIMED, times: ['9:01', '9:03', '9:06'] })

    expect(gutter(again)).toStrictEqual(['9:01', '9:03', '9:06'])
    expect(gutter(first)).toStrictEqual(['0:01', '0:03', '0:06'])
  })

  it('goes on being shown them once the one before it is destroyed', () => {
    const times = timing(() => {})
    const first = drawing(times)
    const again = drawing(times)

    first.destroy()
    times.show(TIMED)

    expect(gutter(again)).toStrictEqual(['0:01', '0:03', '0:06'])
  })
})

describe('the times in the gutter', () => {
  it('are not walked over by the tab key on the way to the words', () => {
    const { times, view } = editor()

    times.show(TIMED)

    const marks = [...view.dom.querySelectorAll<HTMLElement>('.cm-times .cm-time')]
    expect(marks.length).toBe(3)
    expect(marks.every((one) => one.tabIndex === -1)).toBe(true)
  })
})
