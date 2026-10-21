/**
 * The layout of a book: when the columns are set again, which offset is kept in
 * front, and where a link that led here lands.
 *
 * The columns and the turns are stood in for by recorders, so what is read is
 * the order and the arguments the layout joins them with.
 */
import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, reactive, ref, shallowRef } from 'vue'

import { useBookLayout } from './layout'
import type { BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

const fake = vi.hoisted(() => ({
  frames: [] as Array<() => void>,
  columns: {} as Record<string, unknown>,
  turns: {} as Record<string, unknown>,
}))

vi.mock('@/shared/lib/clock', () => ({
  onNextFrame: (run: () => void) => {
    fake.frames.push(run)
  },
}))
vi.mock('./columns', () => ({ useBookColumns: () => fake.columns }))
vi.mock('./turns', () => ({ useBookTurns: () => fake.turns }))

const KEPT = 77
const SPREAD_OF = (at: number) => at + 1000

const setUp = () => {
  fake.frames.length = 0
  fake.columns = {
    textSize: ref(1),
    viewport: ref({ width: 300, height: 200 }),
    isMeasured: ref(true),
    isStood: ref(false),
    flow: ref({}),
    spreadCount: ref(5),
    setting: ref({}),
    marks: shallowRef([]),
    measure: vi.fn(),
    readColumns: vi.fn(),
    findFragmentOffset: vi.fn(),
    markRuns: vi.fn(),
  }
  fake.turns = {
    front: ref({}),
    leftInChapter: ref(0),
    stand: vi.fn(),
    getKeptOffset: vi.fn(() => KEPT),
    getSpreadOf: vi.fn(SPREAD_OF),
    turn: vi.fn(),
    standing: ref(0),
  }
}

const columns = () =>
  fake.columns as {
    viewport: { value: { width: number; height: number } }
    textSize: { value: number }
    isMeasured: { value: boolean }
    isStood: { value: boolean }
    measure: ReturnType<typeof vi.fn>
    readColumns: ReturnType<typeof vi.fn>
    findFragmentOffset: ReturnType<typeof vi.fn>
    markRuns: ReturnType<typeof vi.fn>
  }

const turns = () =>
  fake.turns as {
    stand: ReturnType<typeof vi.fn>
    getKeptOffset: ReturnType<typeof vi.fn>
    getSpreadOf: ReturnType<typeof vi.fn>
    standing: { value: number }
  }

const flushFrames = () => {
  for (const run of fake.frames.splice(0)) run()
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
    span: { from: 100, to: 200 },
    book: { from: 0, to: 500 },
    at: 120,
    highlights: [],
    otherHighlights: [],
    textSize: 1,
    chapter: '',
    words: {},
  }) as unknown as SettledBookProps & { markup: string; at: number }
  const area = shallowRef<HTMLElement | null>(options.hasArea === false ? null : box)
  const paper = shallowRef<HTMLElement | null>(options.hasPaper === false ? null : text)
  const onMove = vi.fn()
  const takeLed = vi.fn(() => options.led)
  let layout!: ReturnType<typeof useBookLayout>
  const wrapper = mount(
    defineComponent({
      setup() {
        layout = useBookLayout({
          area,
          paper,
          props,
          onMove,
          takeLed,
          getEdge: () => 0,
        })
        return () => null
      },
    }),
  )
  return { wrapper, props, box, text, onMove, takeLed, layout: () => layout }
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

    layout().settle(40)

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

    layout().settle(40)

    expect(columns().measure).toHaveBeenCalledTimes(1)
    expect(fake.frames).toHaveLength(0)
    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('does nothing on the frame where the area is gone', () => {
    const { layout } = room({ hasArea: false })
    fake.frames.length = 0

    layout().settle(40)
    flushFrames()

    expect(columns().readColumns).not.toHaveBeenCalled()
    expect(turns().stand).not.toHaveBeenCalled()
    expect(columns().markRuns).not.toHaveBeenCalled()
  })

  it('does nothing on the frame where the text is gone', () => {
    const { layout } = room({ hasPaper: false })
    fake.frames.length = 0

    layout().settle(40)
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

    layout().settle(40, { path: 'ch1.xhtml', fragment: 'here' })
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

    layout().settle(40, { path: '', fragment: 'top' })
    flushFrames()

    expect(columns().findFragmentOffset).toHaveBeenCalledWith('top')
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(0), 'auto')
    expect(onMove).toHaveBeenCalledWith(0)
  })

  it('is not looked for when the link names another document', () => {
    const { layout, onMove } = room()
    columns().findFragmentOffset.mockReturnValue(150)
    fake.frames.length = 0

    layout().settle(40, { path: 'ch2.xhtml', fragment: 'here' })
    flushFrames()

    expect(columns().findFragmentOffset).not.toHaveBeenCalled()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(40), 'auto')
    expect(onMove).not.toHaveBeenCalled()
  })

  it('keeps the offset when the link names a place the document does not hold', () => {
    const { layout, onMove } = room()
    columns().findFragmentOffset.mockReturnValue(undefined)
    fake.frames.length = 0

    layout().settle(40, { path: '', fragment: 'missing' })
    flushFrames()

    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(40), 'auto')
    expect(onMove).not.toHaveBeenCalled()
  })

  it('is not looked for when no link led here', () => {
    const { layout, onMove } = room()
    fake.frames.length = 0

    layout().settle(40)
    flushFrames()

    expect(columns().findFragmentOffset).not.toHaveBeenCalled()
    expect(onMove).not.toHaveBeenCalled()
  })
})

describe('the room or the type changing size', () => {
  it('sets the columns again with the offset in front kept, when the width changes', async () => {
    const { layout } = room()
    layout()
    columns().measure.mockClear()
    fake.frames.length = 0

    columns().viewport.value = { ...columns().viewport.value, width: 640 }
    await nextTick()

    expect(columns().measure).toHaveBeenCalledTimes(1)
    flushFrames()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(KEPT), 'auto')
  })

  it('sets the columns again with the offset in front kept, when the height changes', async () => {
    room()
    columns().measure.mockClear()
    fake.frames.length = 0

    columns().viewport.value = { ...columns().viewport.value, height: 480 }
    await nextTick()

    expect(columns().measure).toHaveBeenCalledTimes(1)
    flushFrames()
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
    const { props, takeLed } = room()
    turns().standing.value = 3
    columns().measure.mockClear()
    fake.frames.length = 0

    props.at = 90
    props.markup = '<p>two</p>'
    await nextTick()

    expect(takeLed).toHaveBeenCalledTimes(1)
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

describe('an offset requested for from outside', () => {
  const asked = async (at: number, standing = 0) => {
    const { props } = room()
    turns().standing.value = standing
    turns().stand.mockClear()
    turns().getSpreadOf.mockClear()
    props.at = at
    await nextTick()
  }

  it('is turned to, smoothly, when it falls inside the document', async () => {
    await asked(150)

    expect(turns().stand).toHaveBeenCalledTimes(1)
    expect(turns().stand).toHaveBeenCalledWith(SPREAD_OF(150), 'smooth')
  })

  it('is turned to when it is the first byte of the document', async () => {
    await asked(100)

    expect(turns().stand).toHaveBeenCalledWith(SPREAD_OF(100), 'smooth')
  })

  it('is turned to when it is the last byte of the document', async () => {
    await asked(199)

    expect(turns().stand).toHaveBeenCalledWith(SPREAD_OF(199), 'smooth')
  })

  it('is not turned to when it falls before the document', async () => {
    await asked(99)

    expect(turns().getSpreadOf).not.toHaveBeenCalled()
    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('is not turned to when it falls where the document ends', async () => {
    await asked(200)

    expect(turns().getSpreadOf).not.toHaveBeenCalled()
    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('is not turned to when it falls after the document', async () => {
    await asked(350)

    expect(turns().stand).not.toHaveBeenCalled()
  })

  it('is not turned to when its spread is the one in front', async () => {
    await asked(150, SPREAD_OF(150))

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

    expect(columns().measure).toHaveBeenCalledTimes(1)
    flushFrames()
    expect(turns().stand).toHaveBeenLastCalledWith(SPREAD_OF(KEPT), 'auto')
  })
})
