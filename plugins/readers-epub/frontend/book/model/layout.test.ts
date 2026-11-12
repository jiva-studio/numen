/**
 * The layout of a book: when the columns are set again, which offset is kept in
 * front, and where a link that led here lands.
 *
 * The columns and the turns are replaced by recorders, so the assertions cover
 * the order and the arguments the layout calls them with.
 */
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, reactive, ref, shallowRef } from 'vue'

import type { useBookColumns } from './columns'
import { useBookLayout } from './layout'
import type { useBookTurns } from './turns'
import type { BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

type BookColumns = ReturnType<typeof useBookColumns>
type BookTurns = ReturnType<typeof useBookTurns>

const fake = vi.hoisted(() => ({
  frames: [] as Array<() => void>,
  columns: {} as {
    sizing: object
    runMarks: object
    laid: object
    isStood: unknown
    readColumns: unknown
  },
  turns: {} as Record<string, unknown>,
}))

vi.mock('@numen/ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@numen/ui')>()),
  onNextFrame: (run: () => void) => {
    fake.frames.push(run)
  },
}))

const KEPT = 77
const SPREAD_OF = (pos: number) => pos + 1000

const setUp = () => {
  fake.frames.length = 0
  fake.columns = {
    sizing: {
      textSize: ref(1),
      viewport: ref({ width: 300, height: 200 }),
      isMeasured: ref(true),
      setting: ref({}),
      measure: vi.fn(),
    },
    runMarks: {
      marks: shallowRef([]),
      findFragmentOffset: vi.fn(),
      markRuns: vi.fn(),
      markCopy: vi.fn(),
      unmarkCopy: vi.fn(),
    },
    laid: { flow: ref({}), spreadCount: ref(5) },
    isStood: ref(false),
    readColumns: vi.fn(),
  }
  fake.turns = {
    position: { front: ref({}), leftInChapter: ref(0), standing: ref(0) },
    moves: {
      stand: vi.fn(),
      enterSpread: vi.fn(),
      getKeptOffset: vi.fn(() => KEPT),
      keepOffset: vi.fn(),
      getSpreadOf: vi.fn(SPREAD_OF),
      turn: vi.fn(),
    },
  }
}

const flatten = (parts: unknown): Record<string, unknown> =>
  Object.assign({}, ...Object.values(parts as Record<string, object>))

const columns = () =>
  ({
    ...flatten({
      sizing: fake.columns.sizing,
      runMarks: fake.columns.runMarks,
      laid: fake.columns.laid,
    }),
    isStood: fake.columns.isStood,
    readColumns: fake.columns.readColumns,
  }) as {
    viewport: { value: { width: number; height: number } }
    textSize: { value: number }
    isMeasured: { value: boolean }
    isStood: { value: boolean }
    measure: ReturnType<typeof vi.fn>
    readColumns: ReturnType<typeof vi.fn>
    findFragmentOffset: ReturnType<typeof vi.fn>
    markRuns: ReturnType<typeof vi.fn>
    markCopy: ReturnType<typeof vi.fn>
    unmarkCopy: ReturnType<typeof vi.fn>
  }

const turns = () =>
  flatten(fake.turns) as {
    stand: ReturnType<typeof vi.fn>
    enterSpread: ReturnType<typeof vi.fn>
    turn: ReturnType<typeof vi.fn>
    getKeptOffset: ReturnType<typeof vi.fn>
    keepOffset: ReturnType<typeof vi.fn>
    getSpreadOf: ReturnType<typeof vi.fn>
    standing: { value: number }
  }

const flushFrames = () => {
  for (const run of fake.frames.splice(0)) run()
}

const flushAllFrames = () => {
  for (let frame = 0; frame < 100 && fake.frames.length > 0; frame += 1) flushFrames()
  expect(fake.frames).toHaveLength(0)
}

interface Options {
  readonly led?: BookLink | undefined
  readonly hasArea?: boolean
  readonly hasPaper?: boolean
}

const room = (options: Options = {}) => {
  const box = document.createElement('div')
  const text = document.createElement('div')
  const props = reactive({
    markup: '<p>one</p>',
    path: 'ch1.xhtml',
    span: { start: 100, end: 200 },
    book: { start: 0, end: 500 },
    at: 120,
    target: null,
    highlights: [],
    otherHighlights: [],
    textSize: 1,
    chapter: '',
    words: {},
  }) as unknown as SettledBookProps & {
    markup: string
    at: number
    target: { offset: number } | null
    span: { start: number; end: number }
  }
  const area = shallowRef<HTMLElement | null>(options.hasArea === false ? null : box)
  const paper = shallowRef<HTMLElement | null>(options.hasPaper === false ? null : text)
  const holder = document.createElement('div')
  const leaving = shallowRef<HTMLElement | null>(holder)
  const onMove = vi.fn()
  const takeLedLink = vi.fn(() => options.led)
  let layout!: ReturnType<typeof useBookLayout>
  const wrapper = mount(
    defineComponent({
      setup() {
        layout = useBookLayout({
          area,
          paper,
          leaving,
          props,
          onMove,
          takeLedLink,
          column: fake.columns as unknown as BookColumns,
          turns: fake.turns as unknown as BookTurns,
        })
        return () => null
      },
    }),
  )
  return { wrapper, props, box, text, holder, onMove, takeLedLink, layout: () => layout }
}

beforeEach(setUp)

afterEach(() => {
  Reflect.deleteProperty(document, 'fonts')
})

describe('setting the columns', () => {
  it('measures the room and reads the columns on the next frame, carrying the offset to the edge', () => {
    const { layout, box, text } = room()
    columns().measure.mockClear()
    turns().stand.mockClear()
    fake.frames.length = 0

    layout().settling.settle(() => 40)

    expect(columns().measure).toHaveBeenCalledTimes(1)
    flushFrames()
    expect(columns().readColumns).toHaveBeenLastCalledWith(box, text)
    expect(turns().getSpreadOf).toHaveBeenLastCalledWith(40)
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(40), 'auto')
    expect(columns().markRuns).toHaveBeenCalled()
  })

  it('reads nothing of an area that has no size yet', () => {
    columns().isMeasured.value = false
    const { layout } = room()
    columns().measure.mockClear()

    layout().settling.settle(() => 40)

    expect(columns().measure).toHaveBeenCalledTimes(1)
    expect(fake.frames).toHaveLength(0)
    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('does nothing on the frame where the area is gone', () => {
    const { layout } = room({ hasArea: false })
    fake.frames.length = 0

    layout().settling.settle(() => 40)
    flushFrames()

    expect(columns().readColumns).not.toHaveBeenCalled()
    expect(turns().stand).not.toHaveBeenCalled()
    expect(columns().markRuns).not.toHaveBeenCalled()
  })

  it('does nothing on the frame where the text is gone', () => {
    const { layout } = room({ hasPaper: false })
    fake.frames.length = 0

    layout().settling.settle(() => 40)
    flushFrames()

    expect(columns().readColumns).not.toHaveBeenCalled()
    expect(turns().stand).not.toHaveBeenCalled()
    expect(columns().markRuns).not.toHaveBeenCalled()
  })
})

describe('the place a link led to', () => {
  it('is found when the link names this document, and the spread of it is put in front', () => {
    const { layout, onMove } = room()
    columns().findFragmentOffset.mockReturnValue(150)
    fake.frames.length = 0

    layout().settling.settle(() => 40, { path: 'ch1.xhtml', fragment: 'here' })
    flushFrames()

    expect(columns().findFragmentOffset).toHaveBeenCalledWith('here')
    expect(turns().getSpreadOf).toHaveBeenLastCalledWith(150)
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(150), 'auto')
    expect(onMove).toHaveBeenCalledWith(150)
  })

  it('is found when the link names no document, and a place at the start of the text is reported', () => {
    const { layout, onMove } = room()
    columns().findFragmentOffset.mockReturnValue(0)
    fake.frames.length = 0

    layout().settling.settle(() => 40, { path: '', fragment: 'top' })
    flushFrames()

    expect(columns().findFragmentOffset).toHaveBeenCalledWith('top')
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(0), 'auto')
    expect(onMove).toHaveBeenCalledWith(0)
  })

  it('is not looked for when the link names another document', () => {
    const { layout, onMove } = room()
    columns().findFragmentOffset.mockReturnValue(150)
    fake.frames.length = 0

    layout().settling.settle(() => 40, { path: 'ch2.xhtml', fragment: 'here' })
    flushFrames()

    expect(columns().findFragmentOffset).not.toHaveBeenCalled()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(40), 'auto')
    expect(onMove).not.toHaveBeenCalled()
  })

  it('keeps the offset when the link names a place the document does not hold', () => {
    const { layout, onMove } = room()
    columns().findFragmentOffset.mockReturnValue(undefined)
    fake.frames.length = 0

    layout().settling.settle(() => 40, { path: '', fragment: 'missing' })
    flushFrames()

    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(40), 'auto')
    expect(onMove).not.toHaveBeenCalled()
  })

  it('is not looked for when no link led here', () => {
    const { layout, onMove } = room()
    fake.frames.length = 0

    layout().settling.settle(() => 40)
    flushFrames()

    expect(columns().findFragmentOffset).not.toHaveBeenCalled()
    expect(onMove).not.toHaveBeenCalled()
  })
})

describe('the room or the type changing size', () => {
  it('sets the columns again with the offset in front kept, once a new width has settled', async () => {
    const { layout } = room()
    layout()
    columns().measure.mockClear()
    fake.frames.length = 0

    columns().viewport.value = { ...columns().viewport.value, width: 640 }
    await nextTick()

    expect(columns().measure).not.toHaveBeenCalled()
    flushAllFrames()
    expect(columns().measure).toHaveBeenCalledTimes(1)
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(KEPT), 'auto')
  })

  it('sets the columns again with the offset in front kept, once a new height has settled', async () => {
    room()
    columns().measure.mockClear()
    fake.frames.length = 0

    columns().viewport.value = { ...columns().viewport.value, height: 480 }
    await nextTick()

    expect(columns().measure).not.toHaveBeenCalled()
    flushAllFrames()
    expect(columns().measure).toHaveBeenCalledTimes(1)
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(KEPT), 'auto')
  })

  it('sets the columns again with the offset in front kept, when the type changes size', async () => {
    room()
    columns().measure.mockClear()
    fake.frames.length = 0

    columns().textSize.value = 1.5
    await nextTick()

    expect(columns().measure).toHaveBeenCalledTimes(1)
    flushFrames()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(KEPT), 'auto')
  })

  it('keeps the offset of a page turned between the change and the frame', async () => {
    room()
    fake.frames.length = 0
    columns().viewport.value = { ...columns().viewport.value, width: 640 }
    await nextTick()

    const TURNED = KEPT + 50
    turns().getKeptOffset.mockReturnValue(TURNED)
    flushAllFrames()

    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(TURNED), 'auto')
  })

  it('reads the columns once for a size that changes on every frame, after it has settled', async () => {
    room()
    flushAllFrames()
    columns().readColumns.mockClear()

    for (const width of [620, 640, 660, 680, 700]) {
      columns().viewport.value = { ...columns().viewport.value, width }
      await nextTick()
      flushFrames()
    }
    expect(columns().readColumns).not.toHaveBeenCalled()

    flushAllFrames()
    expect(columns().readColumns).toHaveBeenCalledTimes(1)
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(KEPT), 'auto')
  })

  it('reads the columns once when the size it measured itself is reported as a change', async () => {
    const { layout } = room()
    flushAllFrames()
    columns().readColumns.mockClear()
    columns().measure.mockImplementation(() => {
      columns().viewport.value = { width: 301, height: 200 }
    })

    layout().settling.settle(() => 40)
    await nextTick()
    flushAllFrames()

    expect(columns().readColumns).toHaveBeenCalledTimes(1)
  })

  it('leaves the columns as they are while none of the three changes', async () => {
    room()
    columns().measure.mockClear()

    columns().viewport.value = { ...columns().viewport.value }
    await nextTick()

    expect(columns().measure).not.toHaveBeenCalled()
  })
})

describe('another document being opened', () => {
  it('starts at the first spread and is set at the offset requested for', async () => {
    const { props, takeLedLink } = room()
    turns().standing.value = 3
    columns().measure.mockClear()
    fake.frames.length = 0

    props.at = 90
    props.markup = '<p>two</p>'
    await nextTick()

    expect(takeLedLink).toHaveBeenCalledTimes(1)
    expect(turns().standing.value).toBe(0)
    expect(columns().measure).toHaveBeenCalledTimes(1)
    flushFrames()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(90), 'auto')
  })

  it('is not stood until the frame that sets it, and is stood after it', async () => {
    const { props } = room()
    flushFrames()
    expect(columns().isStood.value).toBe(true)

    props.markup = '<p>two</p>'
    await nextTick()
    expect(columns().isStood.value).toBe(false)

    flushFrames()
    expect(columns().isStood.value).toBe(true)
  })

  it('stays as it is while the room changes size', async () => {
    room()
    flushFrames()

    columns().viewport.value = { width: 400, height: 200 }
    await nextTick()

    expect(columns().isStood.value).toBe(true)
  })

  it('lands on the place the link led to', async () => {
    const { props, onMove } = room({ led: { path: '', fragment: 'here' } })
    columns().findFragmentOffset.mockReturnValue(160)
    fake.frames.length = 0

    props.markup = '<p>two</p>'
    await nextTick()
    flushFrames()

    expect(columns().findFragmentOffset).toHaveBeenCalledWith('here')
    expect(onMove).toHaveBeenCalledWith(160)
  })
})

describe('a page turned while another document is drawn', () => {
  it('is made at once where the area has no size to set the columns in', () => {
    const { layout } = room()
    columns().isMeasured.value = false
    columns().isStood.value = false

    layout().turn('next')

    expect(turns().turn).toHaveBeenCalledExactlyOnceWith('next')
  })

  it('is made at once where the columns stand for the markup in front', () => {
    const { layout } = room()
    flushFrames()

    layout().turn('next')

    expect(turns().turn).toHaveBeenCalledExactlyOnceWith('next')
  })

  it('waits for the columns to stand for the new markup', async () => {
    const { props, layout } = room()
    flushFrames()
    props.markup = '<p>two</p>'
    await nextTick()

    layout().turn('next')
    layout().turn('next')
    expect(turns().turn).not.toHaveBeenCalled()

    flushFrames()
    await nextTick()
    expect(turns().turn).toHaveBeenCalledTimes(2)
    expect(turns().turn).toHaveBeenNthCalledWith(1, 'next')
    expect(turns().turn).toHaveBeenNthCalledWith(2, 'next')
  })

  it('is made once, and not again by the next document drawn', async () => {
    const { props, layout } = room()
    flushFrames()
    props.markup = '<p>two</p>'
    await nextTick()
    layout().turn('back')
    flushFrames()
    await nextTick()

    props.markup = '<p>three</p>'
    await nextTick()
    flushFrames()
    await nextTick()

    expect(turns().turn).toHaveBeenCalledExactlyOnceWith('back')
  })

  it('queues eight turns at most', async () => {
    const { props, layout } = room()
    flushFrames()
    props.markup = '<p>two</p>'
    await nextTick()

    for (let press = 0; press < 20; press += 1) layout().turn('next')
    flushFrames()
    await nextTick()

    expect(turns().turn).toHaveBeenCalledTimes(8)
  })
})

describe('an offset requested for from outside', () => {
  const request = async (pos: number, standing = 0) => {
    const { props } = room()
    turns().standing.value = standing
    turns().stand.mockClear()
    turns().getSpreadOf.mockClear()
    props.target = { offset: pos }
    await nextTick()
  }

  it('is turned to, smoothly, when it falls inside the document', async () => {
    await request(150)

    expect(turns().stand).toHaveBeenCalledTimes(1)
    expect(turns().stand).toHaveBeenCalledWith(SPREAD_OF(150), 'smooth')
  })

  it('is turned to when it is the first byte of the document', async () => {
    await request(100)

    expect(turns().stand).toHaveBeenCalledWith(SPREAD_OF(100), 'smooth')
  })

  it('is turned to when it is the last byte of the document', async () => {
    await request(199)

    expect(turns().stand).toHaveBeenCalledWith(SPREAD_OF(199), 'smooth')
  })

  it('is not turned to when it falls before the document', async () => {
    await request(99)

    expect(turns().getSpreadOf).not.toHaveBeenCalled()
    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('is not turned to when it falls where the document ends', async () => {
    await request(200)

    expect(turns().getSpreadOf).not.toHaveBeenCalled()
    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('is kept in front when it is turned to', async () => {
    await request(150)

    expect(turns().keepOffset).toHaveBeenCalledWith(150)
  })

  it('is kept in front when its spread is the one in front', async () => {
    await request(150, SPREAD_OF(150))

    expect(turns().stand).not.toHaveBeenCalled()
    expect(turns().keepOffset).toHaveBeenCalledWith(150)
  })

  it('is not kept when it falls after the document', async () => {
    await request(350)

    expect(turns().keepOffset).not.toHaveBeenCalled()
  })

  it('is not turned to when it falls after the document', async () => {
    await request(350)

    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('is not turned to by a change of the offset in front', async () => {
    const { props } = room()
    turns().standing.value = 0
    turns().stand.mockClear()

    props.at = 150
    await nextTick()

    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('is turned to again when the same offset is requested twice', async () => {
    const { props } = room()
    turns().stand.mockClear()

    props.target = { offset: 150 }
    await nextTick()
    turns().standing.value = SPREAD_OF(150) + 1
    props.target = { offset: 150 }
    await nextTick()

    expect(turns().stand).toHaveBeenCalledTimes(2)
  })

  it('is not turned to when its spread is the one in front', async () => {
    await request(150, SPREAD_OF(150))

    expect(turns().getSpreadOf).toHaveBeenCalledWith(150)
    expect(turns().stand).not.toHaveBeenCalled()
  })
})

describe('the layout being mounted', () => {
  it('sets the columns at the offset requested for', () => {
    room()

    expect(columns().measure).toHaveBeenCalledTimes(1)
    flushFrames()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(120), 'auto')
  })

  it('sets them again with the offset in front kept once the type has arrived', async () => {
    Object.defineProperty(document, 'fonts', {
      configurable: true,
      value: { ready: Promise.resolve() },
    })
    room()
    flushFrames()
    columns().measure.mockClear()

    await Promise.resolve()
    await Promise.resolve()

    flushFrames()
    expect(columns().measure).toHaveBeenCalledTimes(1)
    flushFrames()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(KEPT), 'auto')
  })

  it('does not set the columns again on fonts.ready when the fonts are already loaded', async () => {
    Object.defineProperty(document, 'fonts', {
      configurable: true,
      value: { status: 'loaded', ready: Promise.resolve() },
    })
    room()
    flushFrames()
    columns().measure.mockClear()
    turns().stand.mockClear()

    await Promise.resolve()
    await Promise.resolve()

    flushAllFrames()
    expect(columns().measure).not.toHaveBeenCalled()
    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('does not set the columns a second time when fonts.ready resolves before the first layout', async () => {
    Object.defineProperty(document, 'fonts', {
      configurable: true,
      value: { status: 'loading', ready: Promise.resolve() },
    })
    room()

    await Promise.resolve()
    await Promise.resolve()

    flushAllFrames()
    expect(columns().measure).toHaveBeenCalledTimes(1)
    expect(columns().readColumns).toHaveBeenCalledTimes(1)
  })
})

describe('the page counter of a document just drawn', () => {
  const pageOf = (page: number) => ({ page, pages: 40 })
  const setFront = (page: number) => {
    ;(turns() as unknown as { front: { value: unknown } }).front.value = pageOf(page)
  }

  it('stays on the page before it until the columns stand at their spread', async () => {
    const { layout, props } = room()
    flushFrames()
    setFront(48)
    expect(layout().position.front.value).toStrictEqual(pageOf(48))

    props.markup = '<p>the document before</p>'
    await nextTick()
    setFront(1)

    expect(layout().position.front.value).toStrictEqual(pageOf(48))
    flushFrames()
    expect(layout().position.front.value).toStrictEqual(pageOf(1))
  })

  it('stays on the page before it when the span arrives before the markup', () => {
    const { layout, props } = room()
    flushFrames()
    setFront(48)

    props.span = { start: 200, end: 300 }
    setFront(1)

    expect(layout().position.front.value).toStrictEqual(pageOf(48))
  })

  it('follows a turn made once the columns stand', () => {
    const { layout } = room()
    flushFrames()

    setFront(7)

    expect(layout().position.front.value).toStrictEqual(pageOf(7))
  })
})

describe('the text set again for a picture or a face that arrives', () => {
  it('keeps the offset in front, once for the requests of one frame', () => {
    const { layout } = room()
    flushFrames()
    turns().stand.mockClear()

    layout().settling.settleKept()
    layout().settling.settleKept()
    layout().settling.settleKept()
    flushFrames()
    flushFrames()

    expect(turns().stand).toHaveBeenCalledTimes(1)
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(KEPT), 'auto')
  })

  it('is requested when a face finishes loading', () => {
    const listeners = new Map<string, () => void>()
    Object.defineProperty(document, 'fonts', {
      configurable: true,
      value: {
        ready: new Promise(() => undefined),
        addEventListener: (name: string, run: () => void) => listeners.set(name, run),
        removeEventListener: vi.fn(),
      },
    })
    room()
    flushFrames()
    turns().stand.mockClear()

    listeners.get('loadingdone')?.()
    flushFrames()
    flushFrames()

    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(KEPT), 'auto')
  })
})

describe('the columns of a document with a picture that has not arrived', () => {
  const appendImage = (text: HTMLElement) => {
    const picture = document.createElement('img')
    Object.defineProperty(picture, 'complete', { value: false })
    text.append(picture)
    return picture
  }

  it('are read once the picture has arrived', () => {
    const { layout, text } = room()
    flushFrames()
    const picture = appendImage(text)
    columns().readColumns.mockClear()
    turns().stand.mockClear()

    layout().settling.settle(() => 40)
    flushFrames()

    expect(columns().readColumns).not.toHaveBeenCalled()
    picture.dispatchEvent(new Event('load'))
    expect(columns().readColumns).toHaveBeenCalledTimes(1)
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(40), 'auto')
  })

  it('are read once the picture has failed', () => {
    const { layout, text } = room()
    flushFrames()
    const picture = appendImage(text)
    columns().readColumns.mockClear()

    layout().settling.settle(() => 40)
    flushFrames()
    picture.dispatchEvent(new Event('error'))

    expect(columns().readColumns).toHaveBeenCalledTimes(1)
  })
})

describe('a page turned into the adjacent document', () => {
  /** Turns past the end of the document at 100..200 and draws the one at 200..300. */
  const turnIntoNext = async (setup: ReturnType<typeof room>, offset: number) => {
    const { props, layout, text } = setup
    flushFrames()
    text.innerHTML = '<p data-offset="190">one</p>'
    // The place the running turn of the paper goes to when the paper is copied.
    text.style.translate = '-900px 0'
    turns().turn.mockReturnValue(200)
    layout().turn('next')
    props.at = offset
    props.span = { start: 200, end: 300 }
    props.markup = '<p>two</p>'
    await nextTick()
  }

  it('carries the columns in and carries out a copy of the spread before', async () => {
    const setup = room()
    await turnIntoNext(setup, 200)

    expect(setup.holder.textContent).toBe('one')
    flushFrames()

    expect(turns().stand).not.toHaveBeenCalledWith(SPREAD_OF(200), 'auto')
    expect(turns().enterSpread).toHaveBeenCalledExactlyOnceWith(SPREAD_OF(200), 'next', {
      copy: setup.holder.firstElementChild,
      target: -900,
      remove: expect.any(Function),
    })
  })

  it('highlights the copy, and removes it with its highlights at the end of the turn', async () => {
    const setup = room()
    await turnIntoNext(setup, 200)
    const copy = setup.holder.firstElementChild
    expect(columns().markCopy).toHaveBeenCalledExactlyOnceWith(copy)
    flushFrames()

    const leaving = turns().enterSpread.mock.calls[0]?.[2] as { remove: () => void }
    leaving.remove()

    expect(setup.holder.children).toHaveLength(0)
    expect(columns().unmarkCopy).toHaveBeenCalledExactlyOnceWith(copy)
  })

  it('removes the copy with its highlights when another document is drawn', async () => {
    const setup = room()
    await turnIntoNext(setup, 200)
    const copy = setup.holder.firstElementChild
    flushFrames()

    setup.props.at = 310
    setup.props.span = { start: 300, end: 400 }
    setup.props.markup = '<p>three</p>'
    await nextTick()

    expect(setup.holder.children).toHaveLength(0)
    expect(columns().unmarkCopy).toHaveBeenCalledExactlyOnceWith(copy)
  })

  it('lands at once where the owner moved the reader elsewhere', async () => {
    const setup = room()
    await turnIntoNext(setup, 250)
    flushFrames()

    expect(setup.holder.children).toHaveLength(0)
    expect(turns().enterSpread).not.toHaveBeenCalled()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(250), 'auto')
  })

  it('lands at once for the next document drawn without a turn', async () => {
    const setup = room()
    await turnIntoNext(setup, 200)
    flushFrames()
    turns().enterSpread.mockClear()

    setup.props.at = 310
    setup.props.span = { start: 300, end: 400 }
    setup.props.markup = '<p>three</p>'
    await nextTick()
    flushFrames()

    expect(setup.holder.children).toHaveLength(0)
    expect(turns().enterSpread).not.toHaveBeenCalled()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(310), 'auto')
  })
})
