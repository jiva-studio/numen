/**
 * What the one control draws itself from: the picture's scale, the marks on it,
 * the bubble over the knob, and the keys that walk it.
 */
import { describe, expect, it, vi } from 'vitest'
import { nextTick, reactive } from 'vue'

import type { Curve, Point } from '../../types'
import { WORDS as words } from '../../words'
import { useCurveSlider, type CurveSliderProps } from './useCurveSlider'

const point = (over: Partial<Point> = {}): Point => ({
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
  ...over,
})

const curve = (over: Partial<Curve> = {}): Curve => ({
  goal: 'minutes',
  grid: [10, 20, 30, 40],
  days: [],
  at: [
    point({ reviews: 0 }),
    point({ reviews: 40 }),
    point({ reviews: 80 }),
    point({ reviews: 120 }),
  ],
  now: { at: 2, value: 30, day: '' },
  suggested: { at: 3, value: 40, day: '' },
  decks: 1,
  cards: 400,
  overdue: 0,
  unbegun: 0,
  isHonest: true,
  ...over,
})

/** The control standing at a place of a curve, and everything it emitted. */
const slider = (over: Partial<CurveSliderProps> = {}) => {
  const emitted: unknown[][] = []
  const props = reactive<CurveSliderProps>({
    curve: curve(),
    material: null,
    place: 2,
    valueText: '',
    isWaiting: false,
    ...over,
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
    expect(state.knob.value?.y).toBe(16)

    props.curve = curve({
      at: [point(), point({ reviews: 20 }), point({ reviews: 40 }), point({ reviews: 60 })],
    })
    await nextTick()
    expect(state.knob.value?.y).toBe(95)
  })
})

describe('the marks on the picture', () => {
  it('are the knob, unnamed, and the suggestion, named', () => {
    const { state } = slider()
    expect(state.marks.value).toStrictEqual([
      { key: 'knob', at: state.knob.value, text: '' },
      { key: 'suggested', at: state.suggested.value, text: words.markName('minutes') },
    ])
  })

  it('leave the knob out where it relies on no place of the curve', () => {
    const { state } = slider({ place: 7 })
    expect(state.marks.value.map((one) => one.key)).toStrictEqual(['suggested'])
  })

  it('are none, and there is no bubble, while the curve is a sketch', () => {
    const { state } = slider({ curve: curve({ isHonest: false }) })
    expect(state.marks.value).toStrictEqual([])
    expect(state.callout.value).toBeNull()
    expect(state.heights.value.map((one: { text: string }) => one.text)).toStrictEqual([
      words.formatGoalCost('minutes', 120),
    ])
  })

  it('leave a name off where it would print over the bubble', () => {
    const costs = [10, 10, 10, 10, 0, 20, 30, 40, 50, 100]
    const { state } = slider({
      curve: curve({
        grid: costs.map((_, at) => at * 10),
        at: costs.map((reviews) => point({ reviews })),
        now: { at: 4, value: 40, day: '' },
        suggested: { at: 5, value: 50, day: '' },
      }),
      place: 4,
    })
    expect(state.marks.value.map((one) => one.text)).toContain(words.markName('minutes'))
    expect(state.callout.value).not.toBeNull()
    expect(state.named.value).toStrictEqual([])
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
    expect(state.short.value).toMatch(/^M\S+ \S+ L\S+ \S+$/)
  })

  it('says whether the goal is a date', () => {
    expect(slider().state.isDated.value).toBe(false)
    expect(slider({ curve: curve({ goal: 'date' }) }).state.isDated.value).toBe(true)
  })

  it('says nothing of learning where the knob relies on no place', () => {
    expect(slider({ place: 7 }).state.learning.value).toStrictEqual([])
  })

  it('reads the near end of the range off the first place', () => {
    const { state } = slider()
    expect(state.least.value).toBe(10)
    expect(state.atLeast.value).toBe(words.formatGoalValue('minutes', 10))
  })
})

describe('the keys that walk the knob', () => {
  it('take the arrow a key walks, and move the knob a place', () => {
    const { state, emitted } = slider()
    const right = key('ArrowRight')
    state.onKeyDown(right)
    expect(right.preventDefault).toHaveBeenCalled()
    expect(emitted).toStrictEqual([['move', 3]])
  })

  it('write once a walking key is release, and not for a key that walks nothing', () => {
    const { state, emitted } = slider()
    state.onKeyUp(key('a'))
    expect(emitted).toStrictEqual([])
    state.onKeyUp(key('ArrowRight'))
    expect(emitted).toStrictEqual([['commit']])
  })
})

describe('the picture the pointer is read against', () => {
  it('is the element the view passes over', () => {
    const { state } = slider()
    const element = document.createElementNS('http://www.w3.org/2000/svg', 'svg')
    state.setPicture(element)
    expect(state.picture.value).toBe(element)
  })
})

/** A picture the pointer is read against, scaled and shifted as the test says. */
const createPicture = (a: number, e: number, isCaptured = false) => {
  const isDone: string[] = []
  const element = Object.assign(document.createElementNS('http://www.w3.org/2000/svg', 'svg'), {
    getScreenCTM: () => ({ a, e }),
    focus: () => void isDone.push('focus'),
    setPointerCapture: (id: number) => void isDone.push(`capture ${id}`),
    hasPointerCapture: () => isCaptured,
    releasePointerCapture: (id: number) => void isDone.push(`release ${id}`),
  })
  return { element, done: isDone }
}

const pointer = (clientX: number, button = 0) =>
  ({ clientX, button, pointerId: 7, preventDefault: vi.fn() }) as unknown as PointerEvent & {
    preventDefault: ReturnType<typeof vi.fn>
  }

describe('the pointer on the picture', () => {
  it('takes the knob to the place under it, and holds the pointer', () => {
    const { state, emitted } = slider()
    const { element, done } = createPicture(2, 100)
    state.setPicture(element)
    const down = pointer(481)
    state.onPointerDown(down)
    expect(down.preventDefault).toHaveBeenCalled()
    expect(done).toStrictEqual(['focus', 'capture 7'])
    expect(emitted).toStrictEqual([['move', 1]])
  })

  it('leaves a button other than the first to the window', () => {
    const { state, emitted } = slider()
    const { element, done } = createPicture(2, 100)
    state.setPicture(element)
    const down = pointer(481, 2)
    state.onPointerDown(down)
    expect(down.preventDefault).not.toHaveBeenCalled()
    expect(done).toStrictEqual([])
    expect(emitted).toStrictEqual([])
  })

  it('leaves the knob at its current position where the picture has no scale', () => {
    const { state, emitted } = slider()
    state.setPicture(createPicture(0, 100).element)
    state.onPointerDown(pointer(481))
    expect(emitted).toStrictEqual([['move', 2]])
  })

  it('leaves the knob at its current position where there is no picture to read against', () => {
    const { state, emitted } = slider()
    state.onPointerDown(pointer(481))
    const unscaled = Object.assign(document.createElement('div'), { setPointerCapture: () => {} })
    state.setPicture(unscaled as unknown as SVGSVGElement)
    state.onPointerDown(pointer(481))
    expect(emitted).toStrictEqual([
      ['move', 2],
      ['move', 2],
    ])
  })

  it('walks the knob while the pointer is retained, and writes once it is let go', () => {
    const { state, emitted } = slider()
    const { element, done } = createPicture(2, 100, true)
    state.setPicture(element)
    state.onPointerMove(pointer(481))
    state.onPointerUp(pointer(481))
    expect(emitted).toStrictEqual([['move', 1], ['commit']])
    expect(done).toStrictEqual(['release 7'])
  })

  it('does nothing as a pointer it does not hold moves or is let go', () => {
    const { state, emitted } = slider()
    state.onPointerMove(pointer(481))
    state.onPointerUp(pointer(481))
    const { element, done } = createPicture(2, 100, false)
    state.setPicture(element)
    state.onPointerMove(pointer(481))
    state.onPointerUp(pointer(481))
    expect(emitted).toStrictEqual([])
    expect(done).toStrictEqual([])
  })
})
