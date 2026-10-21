/**
 * A time against every line, and the line being read now.
 *
 * The gutter and the decoration are read out of the view the extension was put
 * in, which is what a person sees.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { EditorState, StateEffect, StateField, type Extension } from '@codemirror/state'
import { Decoration, EditorView, type DecorationSet } from '@codemirror/view'
import { createTimeGutter, type TimeGutterState } from './timeGutter'

const drawn: EditorView[] = []

const TEXT = 'A bell over the door.\nRain on the awning.\nSomeone counting change.'

const TIMED: TimeGutterState = { times: ['0:01', '0:03', '0:06'], current: -1, isFollowing: false }

const editor = (goToLine: (line: number) => void = () => {}) => {
  const times = createTimeGutter(goToLine)
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
const waitTick = () => new Promise((done) => setTimeout(done, 0))

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

  it('does not dispatch when the state is unchanged', () => {
    const { times, view } = editor()
    times.show(TIMED)

    let dispatches = 0
    const was = view.dispatch.bind(view)
    view.dispatch = ((...specs: Parameters<EditorView['dispatch']>) => {
      dispatches++
      return was(...specs)
    }) as EditorView['dispatch']

    times.show({ ...TIMED })

    expect(dispatches).toBe(0)
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

  it('is no line where the document is shorter than the one requested for', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 9 })

    expect(current(view)).toStrictEqual([])
  })

  it('is drawn whether or not the view follows it', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 2, isFollowing: true })
    expect(current(view)).toStrictEqual(['Someone counting change.'])

    times.show({ ...TIMED, current: 2, isFollowing: false })
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
    const times = createTimeGutter(() => {})
    const first = drawing(times)
    times.show({ ...TIMED, current: 1 })
    first.destroy()

    const again = drawing(times)
    await waitTick()

    expect(gutter(again)).toStrictEqual(['0:01', '0:03', '0:06'])
    expect(current(again)).toStrictEqual(['Rain on the awning.'])
  })

  it('is shown them where nothing was drawn when they were given', async () => {
    const times = createTimeGutter(() => {})
    times.show(TIMED)

    const view = drawing(times)
    await waitTick()

    expect(gutter(view)).toStrictEqual(['0:01', '0:03', '0:06'])
  })

  it('is the one shown what arrives next, and the one before it is left alone', async () => {
    const times = createTimeGutter(() => {})
    const first = drawing(times)
    times.show(TIMED)
    const again = drawing(times)
    await waitTick()

    times.show({ ...TIMED, times: ['9:01', '9:03', '9:06'] })

    expect(gutter(again)).toStrictEqual(['9:01', '9:03', '9:06'])
    expect(gutter(first)).toStrictEqual(['0:01', '0:03', '0:06'])
  })

  it('goes on being shown them once the one before it is destroyed', () => {
    const times = createTimeGutter(() => {})
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

describe('the view going after the line being returned', () => {
  /**
   * Every scroll the extension asked for, counted. It shows what it shows in
   * one effect and asks to be moved in a second, so a dispatch carrying two is
   * a dispatch that moves the view.
   */
  const watchView = (view: EditorView) => {
    let scrolls = 0
    const was = view.dispatch.bind(view)
    view.dispatch = ((...specs: Parameters<EditorView['dispatch']>) => {
      for (const one of specs) {
        const effects = one && typeof one === 'object' && 'effects' in one ? one.effects : null
        if (Array.isArray(effects) && effects.length > 1) scrolls++
      }
      return was(...specs)
    }) as EditorView['dispatch']
    return () => scrolls
  }

  it('does not move for words arriving while nothing is being returned', async () => {
    const { times, view } = editor()
    await waitTick()
    times.show({ ...TIMED, current: 1, isFollowing: true })

    const scrolls = watchView(view)
    // The transcript grows: another line arrives, and the line being said is
    // the one it was.
    times.show({ times: [...TIMED.times, '0:09'], current: 1, isFollowing: true })

    expect(scrolls()).toBe(0)
  })

  it('moves where the line being returned becomes another', async () => {
    const { times, view } = editor()
    await waitTick()
    times.show({ ...TIMED, current: 0, isFollowing: true })

    const scrolls = watchView(view)
    times.show({ ...TIMED, current: 2, isFollowing: true })

    expect(scrolls()).toBe(1)
  })
})

describe('an editor drawn a second time', () => {
  it('is shown the times without the view being moved', async () => {
    const { times, view } = editor()
    await waitTick()
    times.show({ ...TIMED, current: 2, isFollowing: true })
    view.destroy()

    // A tab moved between panes is drawn again while the recording stands
    // where it stood.
    const again = drawing(times)
    let effects = 0
    const was = again.dispatch.bind(again)
    again.dispatch = ((...specs: Parameters<EditorView['dispatch']>) => {
      for (const one of specs) {
        const put = one && typeof one === 'object' && 'effects' in one ? one.effects : null
        if (Array.isArray(put) && put.length > 1) effects++
      }
      return was(...specs)
    }) as EditorView['dispatch']
    await waitTick()

    expect(gutter(again)).toStrictEqual(TIMED.times)
    expect(effects).toBe(0)
  })
})

describe('an editor drawn again while the recording remains still', () => {
  it('is not moved by the words being requested for again', async () => {
    const times = createTimeGutter(() => {})
    const first = drawing(times)
    times.show({ ...TIMED, current: 2, isFollowing: true })
    first.destroy()

    const again = drawing(times)
    await waitTick()

    let effects = 0
    const was = again.dispatch.bind(again)
    again.dispatch = ((...specs: Parameters<EditorView['dispatch']>) => {
      for (const one of specs) {
        const put = one && typeof one === 'object' && 'effects' in one ? one.effects : null
        if (Array.isArray(put) && put.length > 1) effects++
      }
      return was(...specs)
    }) as EditorView['dispatch']

    // The transcript is asked for again while nothing is being said: the words
    // grow and the line being said is the one it was.
    times.show({ times: [...TIMED.times, '0:09'], current: 2, isFollowing: true })

    expect(effects).toBe(0)
  })
})

/** Every list of effects the view was dispatched, in order. */
const recordEffects = (view: EditorView) => {
  const seen: StateEffect<unknown>[][] = []
  const was = view.dispatch.bind(view)
  view.dispatch = ((...specs: Parameters<EditorView['dispatch']>) => {
    for (const one of specs) {
      const effects = one && typeof one === 'object' && 'effects' in one ? one.effects : null
      if (Array.isArray(effects)) seen.push(effects)
    }
    return was(...specs)
  }) as EditorView['dispatch']
  return seen
}

/** The dispatches that asked the view to move. */
const moves = (seen: StateEffect<unknown>[][]) => seen.filter((effects) => effects.length > 1)

describe('an effect the times did not send', () => {
  it('leaves what is shown as it was', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 1 })
    const other = StateEffect.define<number>()

    view.dispatch({ effects: other.of(5) })

    expect(gutter(view)).toStrictEqual(['0:01', '0:03', '0:06'])
    expect(current(view)).toStrictEqual(['Rain on the awning.'])
  })
})

describe('the line being read, at the edges of the document', () => {
  it('is drawn where it is the first line', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 0 })

    expect(current(view)).toStrictEqual(['A bell over the door.'])
  })

  it('is drawn where it is the last line', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 2 })

    expect(current(view)).toStrictEqual(['Someone counting change.'])
  })

  it('is no line where it is the count of the lines', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 3 })

    expect(current(view)).toStrictEqual([])
  })
})

describe('the gutter elements', () => {
  const elements = (view: EditorView) =>
    [...view.dom.querySelectorAll('.cm-times .cm-gutterElement')].map((one) => one.className)

  it('carry the class of the line being read and no other', () => {
    const { times, view } = editor()

    times.show({ ...TIMED, current: 1 })

    expect(elements(view)).toStrictEqual([
      'cm-gutterElement',
      'cm-gutterElement cm-current',
      'cm-gutterElement',
    ])
  })

  it('give the class up when the line is no longer the one being read', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 0 })

    times.show({ ...TIMED, current: 1 })

    expect(elements(view)).toStrictEqual([
      'cm-gutterElement',
      'cm-gutterElement cm-current',
      'cm-gutterElement',
    ])
  })

  it('keep the time of a line that did not change', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 0 })
    const marks = () => [...view.dom.querySelectorAll('.cm-times .cm-time')]
    const before = marks()

    times.show({ ...TIMED, current: 1 })

    const after = marks()
    expect(after[2]).toBe(before[2])
    expect(after[0]).not.toBe(before[0])
    expect(after[1]).not.toBe(before[1])
  })

  it('keep every time while the text is typed into', () => {
    const { times, view } = editor()
    times.show(TIMED)
    const before = [...view.dom.querySelectorAll('.cm-times .cm-time')]

    view.dispatch({ changes: { from: 0, to: 0, insert: 'Then. ' } })

    const after = [...view.dom.querySelectorAll('.cm-times .cm-time')]
    expect(after).toHaveLength(3)
    expect(after.every((one, at) => one === before[at])).toBe(true)
  })
})

describe('a time in the gutter', () => {
  it('is a button that submits nothing', () => {
    const { times, view } = editor()
    times.show(TIMED)

    const mark = view.dom.querySelector<HTMLButtonElement>('.cm-times .cm-time')

    expect(mark?.type).toBe('button')
  })

  it('keeps the editor from taking the click for its own', () => {
    const { times, view } = editor()
    times.show(TIMED)

    const click = new MouseEvent('mousedown', { bubbles: true, cancelable: true })
    view.dom.querySelector('.cm-times .cm-time')?.dispatchEvent(click)

    expect(click.defaultPrevented).toBe(true)
  })
})

describe('the same times given again', () => {
  it('are not shown again where they are a copy', () => {
    const { times, view } = editor()
    times.show(TIMED)
    const seen = recordEffects(view)

    times.show({ ...TIMED, times: [...TIMED.times] })

    expect(seen).toHaveLength(0)
  })

  it('are shown again where one of them is another', () => {
    const { times, view } = editor()
    times.show(TIMED)

    times.show({ ...TIMED, times: ['0:01', '9:99', '0:06'] })

    expect(gutter(view)).toStrictEqual(['0:01', '9:99', '0:06'])
  })

  it('are shown again where only the following of the line changed', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 1 })
    const seen = recordEffects(view)

    times.show({ ...TIMED, current: 1, isFollowing: true })

    expect(seen).toHaveLength(1)
  })
})

describe('the view following the line being returned', () => {
  it('moves to it where following is turned on', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 1 })
    const seen = recordEffects(view)

    times.show({ ...TIMED, current: 1, isFollowing: true })

    expect(moves(seen)).toHaveLength(1)
  })

  it('does not move where following is turned off', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 1, isFollowing: true })
    const seen = recordEffects(view)

    times.show({ ...TIMED, current: 1, isFollowing: false })

    expect(seen).toHaveLength(1)
    expect(moves(seen)).toHaveLength(0)
  })

  it('moves to the nearest edge of the line', () => {
    const { times, view } = editor()
    const seen = recordEffects(view)

    times.show({ ...TIMED, current: 1, isFollowing: true })

    const target = moves(seen)[0]?.[1] as unknown as { value: { y: string } }
    expect(target.value.y).toBe('nearest')
  })

  it('moves to the first line', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 2, isFollowing: true })
    const seen = recordEffects(view)

    times.show({ ...TIMED, current: 0, isFollowing: true })

    expect(moves(seen)).toHaveLength(1)
  })

  it('does not move where no line is being returned', () => {
    const { times, view } = editor()
    times.show({ ...TIMED, current: 2, isFollowing: true })
    const seen = recordEffects(view)

    times.show({ ...TIMED, current: -1, isFollowing: true })

    expect(seen).toHaveLength(1)
    expect(moves(seen)).toHaveLength(0)
  })

  it('does not move to the line the count of the lines names', () => {
    const { times, view } = editor()
    const seen = recordEffects(view)

    times.show({ ...TIMED, current: 3, isFollowing: true })

    expect(seen).toHaveLength(1)
    expect(moves(seen)).toHaveLength(0)
  })

  it('does not move to a line the document does not have', () => {
    const { times, view } = editor()
    const seen = recordEffects(view)

    times.show({ ...TIMED, current: 9, isFollowing: true })

    expect(seen).toHaveLength(1)
    expect(moves(seen)).toHaveLength(0)
  })
})

describe('an editor destroyed', () => {
  it('is shown nothing more', () => {
    const { times, view } = editor()
    times.show(TIMED)
    view.destroy()
    const seen = recordEffects(view)

    times.show({ ...TIMED, current: 1 })

    expect(seen).toHaveLength(0)
  })

  it('before it was shown what the one before it was, leaves the later times to the next editor', async () => {
    const times = createTimeGutter(() => {})
    const first = drawing(times)
    times.show(TIMED)
    first.destroy()
    const second = drawing(times)
    times.show({ ...TIMED, times: ['9:01', '9:03', '9:06'] })
    second.destroy()
    await waitTick()

    const third = drawing(times)
    await waitTick()

    expect(gutter(third)).toStrictEqual(['9:01', '9:03', '9:06'])
  })
})

describe('a time that changes in place', () => {
  it('is drawn anew where its text changes, and again where it becomes the line being read', () => {
    const { times, view } = editor()
    times.show(TIMED)
    const marked = () =>
      [...view.dom.querySelectorAll('.cm-times .cm-gutterElement')].map((one) =>
        one.classList.contains('cm-current'),
      )

    times.show({ ...TIMED, times: ['0:01', '0:04', '0:06'] })
    expect(gutter(view)).toStrictEqual(['0:01', '0:04', '0:06'])
    expect(marked()).toStrictEqual([false, false, false])

    times.show({ ...TIMED, times: ['0:01', '0:04', '0:06'], current: 1 })
    expect(gutter(view)).toStrictEqual(['0:01', '0:04', '0:06'])
    expect(marked()).toStrictEqual([false, true, false])
  })

  it('is drawn anew where only the times change and the text stays', () => {
    const { times, view } = editor()
    times.show(TIMED)
    const text = view.state.doc

    times.show({ ...TIMED, times: ['0:02', '0:03', '0:06'] })

    expect(view.state.doc).toBe(text)
    expect(gutter(view)).toStrictEqual(['0:02', '0:03', '0:06'])
  })
})

describe('a time whose slot in the gutter is taken by another line', () => {
  it('goes to the line it now remains against', () => {
    const fold = StateEffect.define<null>()
    const folded = StateField.define<DecorationSet>({
      create: () => Decoration.none,
      update: (set, transaction) =>
        transaction.effects.some((effect) => effect.is(fold))
          ? Decoration.set([Decoration.replace({}).range(0, transaction.state.doc.line(2).from)])
          : set,
      provide: (field) => EditorView.decorations.from(field),
    })
    const asked: number[] = []
    const times = createTimeGutter((line) => asked.push(line))
    const view = new EditorView({
      parent: document.body,
      state: EditorState.create({ doc: TEXT, extensions: [times.extension, folded] }),
    })
    drawn.push(view)
    times.show({ ...TIMED, times: ['0:01', '0:05', '0:05'] })

    view.dispatch({ effects: fold.of(null) })
    view.dom
      .querySelectorAll('.cm-times .cm-time')[1]
      ?.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))

    expect(gutter(view)).toStrictEqual(['0:01', '0:05'])
    expect(asked).toStrictEqual([2])
  })
})
