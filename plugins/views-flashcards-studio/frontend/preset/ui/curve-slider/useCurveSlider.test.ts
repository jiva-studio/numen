/**
 * What the curve control is drawn from: the plot scale, the marks on it, the
 * callout above the knob, and the keys that move the knob.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, reactive } from 'vue'

import type { Curve, Point } from '../../types'
import { WORDS as words } from '../../words'
import { useCurveSlider, type CurveSliderProps } from './useCurveSlider'

const point = (overrides: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  canLearnEveryCard: true,
  closed: [],
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
  ...overrides,
})

const curve = (overrides: Partial<Curve> = {}): Curve => ({
  goal: 'minutes',
  grid: [10, 20, 30, 40],
  days: [],
  at: [
    point({ reviews: 0 }),
    point({ reviews: 40 }),
    point({ reviews: 80 }),
    point({ reviews: 120 }),
  ],
  now: { index: 2, value: 30, day: '' },
  suggested: { index: 3, value: 40, day: '' },
  decks: 1,
  cards: 400,
  overdue: 0,
  unbegun: 0,
  isHonest: true,
  ...overrides,
})

/** The control standing at a place of a curve, and everything it emitted. */
const slider = (overrides: Partial<CurveSliderProps> = {}) => {
  const emitted: unknown[][] = []
  const props = reactive<CurveSliderProps>({
    curve: curve(),
    material: null,
    place: 2,
    valueText: '',
    isWaiting: false,
    ...overrides,
  })
  const emit = (...args: unknown[]) => void emitted.push(args)
  const state = useCurveSlider(props, emit as unknown as Parameters<typeof useCurveSlider>[1])
  return { props, state, emitted }
}

const key = (name: string) =>
  ({ key: name, preventDefault: vi.fn() }) as unknown as KeyboardEvent & {
    preventDefault: ReturnType<typeof vi.fn>
  }

describe('the scale the picture is drawn to', () => {
  it('is the first answer’s for a goal, and a later answer of that goal is drawn against it', async () => {
    const { props, state } = slider({ place: 3 })
    expect(state.plot.knob.value?.y).toBe(16)

    props.curve = curve({
      at: [point(), point({ reviews: 20 }), point({ reviews: 40 }), point({ reviews: 60 })],
    })
    await nextTick()
    expect(state.plot.knob.value?.y).toBe(95)
  })
})

describe('the marks on the picture', () => {
  it('are the knob, unnamed, and the suggestion, named', () => {
    const { state } = slider()
    expect(state.annotations.labels.marks.value).toStrictEqual([
      { key: 'knob', at: state.plot.knob.value, text: '' },
      { key: 'suggested', at: state.plot.suggested.value, text: words.markName('minutes') },
    ])
  })

  it('leave the knob out where it relies on no place of the curve', () => {
    const { state } = slider({ place: 7 })
    expect(state.annotations.labels.marks.value.map((one) => one.key)).toStrictEqual(['suggested'])
  })

  it('are none, and there is no bubble, while the curve is a sketch', () => {
    const { state } = slider({ curve: curve({ isHonest: false }) })
    expect(state.annotations.labels.marks.value).toStrictEqual([])
    expect(state.annotations.labels.callout.value).toBeNull()
    expect(
      state.annotations.labels.heights.value.map((one: { text: string }) => one.text),
    ).toStrictEqual([words.formatGoalCost('minutes', 120)])
  })

  it('leave a name off where it would print over the bubble', () => {
    const costs = [10, 10, 10, 10, 0, 20, 30, 40, 50, 100]
    const { state } = slider({
      curve: curve({
        grid: costs.map((_, index) => index * 10),
        at: costs.map((reviews) => point({ reviews })),
        now: { index: 4, value: 40, day: '' },
        suggested: { index: 5, value: 50, day: '' },
      }),
      place: 4,
    })
    expect(state.annotations.labels.marks.value.map((one) => one.text)).toContain(
      words.markName('minutes'),
    )
    expect(state.annotations.labels.callout.value).not.toBeNull()
    expect(state.annotations.labels.named.value).toStrictEqual([])
  })
})

describe('what the control says alongside the picture', () => {
  it('draws the stretch a day leaves cards short over', () => {
    const { state } = slider({
      curve: curve({
        at: [
          point(),
          point({ reviews: 40 }),
          point({ reviews: 80, canLearnEveryCard: false }),
          point({ reviews: 120, canLearnEveryCard: false }),
        ],
      }),
    })
    expect(state.plot.short.value).toMatch(/^M\S+ \S+ L\S+ \S+$/)
  })

  it('says whether the goal is a date', () => {
    expect(slider().state.annotations.isDated.value).toBe(false)
    expect(slider({ curve: curve({ goal: 'date' }) }).state.annotations.isDated.value).toBe(true)
  })

  it('says nothing of learning where the knob relies on no place', () => {
    expect(slider({ place: 7 }).state.annotations.tiles.learning.value).toStrictEqual([])
  })

  it('reads the near end of the range off the first place', () => {
    const { state } = slider()
    expect(state.annotations.bounds.min.value).toBe(10)
    expect(state.annotations.bounds.atLeast.value).toBe(words.formatGoalValue('minutes', 10))
  })
})

describe('the keys that walk the knob', () => {
  it('take the arrow a key walks, and move the knob a place', () => {
    const { state, emitted } = slider()
    const right = key('ArrowRight')
    state.interactions.onKeyDown(right)
    expect(right.preventDefault).toHaveBeenCalled()
    expect(emitted).toStrictEqual([['move', 3]])
  })

  it('write once a walking key is release, and not for a key that walks nothing', () => {
    const { state, emitted } = slider()
    state.interactions.onKeyUp(key('a'))
    expect(emitted).toStrictEqual([])
    state.interactions.onKeyUp(key('ArrowRight'))
    expect(emitted).toStrictEqual([['commit']])
  })
})

describe('the picture the pointer is read against', () => {
  it('is the element the view passes over', () => {
    const { state } = slider()
    const element = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
    state.interactions.setChart(element)
    expect(state.interactions.picture.value).toBe(element)
  })
})

/** A chart the pointer is read against, scaled and shifted as the test specifies. */
const createChart = (a: number, e: number, isCaptured = false) => {
  const isDone: string[] = []
  const element = Object.assign(document.createElementNS('http://www.w3.org/2000/svg', 'svg'), {
    getScreenCTM: () => ({ a, e }),
    focus: () => void isDone.push('focus'),
    setPointerCapture: (id: number) => void isDone.push(`capture ${id}`),
    hasPointerCapture: () => isCaptured,
    releasePointerCapture: (id: number) => void isDone.push(`release ${id}`),
  })
  return { element, recorded: isDone }
}

const pointer = (clientX: number, button = 0) =>
  ({ clientX, button, pointerId: 7, preventDefault: vi.fn() }) as unknown as PointerEvent & {
    preventDefault: ReturnType<typeof vi.fn>
  }

describe('the pointer on the picture', () => {
  it('takes the knob to the place under it, and holds the pointer', () => {
    const { state, emitted } = slider()
    const { element, recorded } = createChart(2, 100)
    state.interactions.setChart(element)
    const down = pointer(481)
    state.interactions.onPointerDown(down)
    expect(down.preventDefault).toHaveBeenCalled()
    expect(recorded).toStrictEqual(['focus', 'capture 7'])
    expect(emitted).toStrictEqual([['move', 1]])
  })

  it('leaves a button other than the first to the window', () => {
    const { state, emitted } = slider()
    const { element, recorded } = createChart(2, 100)
    state.interactions.setChart(element)
    const down = pointer(481, 2)
    state.interactions.onPointerDown(down)
    expect(down.preventDefault).not.toHaveBeenCalled()
    expect(recorded).toStrictEqual([])
    expect(emitted).toStrictEqual([])
  })

  it('leaves the knob at its current position where the picture has no scale', () => {
    const { state, emitted } = slider()
    state.interactions.setChart(createChart(0, 100).element)
    state.interactions.onPointerDown(pointer(481))
    expect(emitted).toStrictEqual([['move', 2]])
  })

  it('leaves the knob at its current position where there is no picture to read against', () => {
    const { state, emitted } = slider()
    state.interactions.onPointerDown(pointer(481))
    const unscaled = Object.assign(document.createElement('div'), { setPointerCapture: () => {} })
    state.interactions.setChart(unscaled as unknown as SVGSVGElement)
    state.interactions.onPointerDown(pointer(481))
    expect(emitted).toStrictEqual([
      ['move', 2],
      ['move', 2],
    ])
  })

  it('walks the knob while the pointer is retained, and writes once it is let go', () => {
    const { state, emitted } = slider()
    const { element, recorded } = createChart(2, 100, true)
    state.interactions.setChart(element)
    state.interactions.onPointerMove(pointer(481))
    state.interactions.onPointerUp(pointer(481))
    expect(emitted).toStrictEqual([['move', 1], ['commit']])
    expect(recorded).toStrictEqual(['release 7'])
  })

  it('does nothing as a pointer it does not hold moves or is let go', () => {
    const { state, emitted } = slider()
    state.interactions.onPointerMove(pointer(481))
    state.interactions.onPointerUp(pointer(481))
    const { element, recorded } = createChart(2, 100, false)
    state.interactions.setChart(element)
    state.interactions.onPointerMove(pointer(481))
    state.interactions.onPointerUp(pointer(481))
    expect(emitted).toStrictEqual([])
    expect(recorded).toStrictEqual([])
  })
})
