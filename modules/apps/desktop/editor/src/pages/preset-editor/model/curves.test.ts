/**
 * The curve of a preset is sketched at once and asked of the application, one
 * question at a time; an answer already held is drawn without asking, and an
 * answer to a question nobody is waiting for is not drawn.
 */
import { describe, expect, it } from 'vitest'
import {
  DEFAULTS,
  NO_CURVE_POINT,
  type Curve,
  type Presets,
  type Settings,
  type SettingsBounds,
} from '../types'
import { estimateCurve } from '../lib/sketch'
import { getShapeKey } from '../lib/fields'
import { WORDS as words } from '../words'
import {
  applyCurveAnswer,
  createCurveState,
  getKnobPosition,
  isSameGrid,
  updateCurves as updateCurvesImpl,
  type CurveState,
} from './curves'

const updateCurves = (
  state: CurveState,
  path: string,
  getSettings: () => Settings,
  core: Presets,
  bounds: SettingsBounds,
  today: string,
  setErrorMessage: (msg: string) => void,
) => updateCurvesImpl(state, { path, getSettings, core, bounds, today, setErrorMessage })

const today = '2026-08-30'
const PATH = 'presets/Daily.md'

const BOUNDS = {
  minutesADay: { least: 0, most: 24 * 60 },
  newADay: { least: 0, most: 9999 },
  reviewsADay: { least: 0, most: 9999 },
  retention: { least: 0.7, most: 0.99 },
  backlog: { least: 0, most: 100 },
  interval: { least: 1, most: 365 },
  load: { least: 0, most: 100 },
}

const settings = (over: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...over })

const curve = (over: Partial<Curve> = {}): Curve => ({
  goal: 'minutes',
  grid: [0, 10, 20, 30],
  days: [],
  at: [],
  now: { at: 1, value: 10, day: '' },
  suggested: NO_CURVE_POINT,
  decks: 2,
  cards: 40,
  overdue: 3,
  unbegun: 7,
  isHonest: true,
  ...over,
})

/** An answer the test gives when it chooses. */
const createGate = () => {
  let open: (answer: Curve) => void = () => {}
  let fail: (error: Error) => void = () => {}
  const answered = new Promise<Curve>((resolve, reject) => {
    open = resolve
    fail = reject
  })
  return { answered, open, fail }
}

/** An application that answers each question for a curve in turn. */
const createCore = (...answers: Promise<Curve>[]) => {
  const asked: Settings[] = []
  const core = {
    curve: async (_path: string, asking: Settings) => {
      asked.push(asking)
      return (await answers.shift()) ?? curve()
    },
  } as unknown as Presets
  return { core, asked }
}

const nap = () => new Promise((wake) => setTimeout(wake, 0))

describe('a preset tab before any curve has landed', () => {
  it('is waiting for one, and holds no answer of the application', () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)

    expect(state.isWaiting.value).toBe(true)
    expect(state.shape).toBe('')
    expect(state.isReal).toBe(false)
  })
})

describe('whether two curves are drawn over the same range', () => {
  it('is so place for place', () => {
    expect(isSameGrid([0, 10, 20], [0, 10, 20])).toBe(true)
  })

  it('is not so for a range running further', () => {
    expect(isSameGrid([0, 10], [0, 10, 20])).toBe(false)
  })

  it('is not so where one place differs', () => {
    expect(isSameGrid([0, 10, 20], [0, 10, 25])).toBe(false)
  })
})

describe('where the knob stands', () => {
  it('is where the curve says the preset stands, the first place among them', () => {
    expect(getKnobPosition(curve({ now: { at: 0, value: 0, day: '' } }), settings(), today)).toBe(0)
    expect(getKnobPosition(curve({ now: { at: 3, value: 30, day: '' } }), settings(), today)).toBe(
      3,
    )
  })

  it('is the place nearest the preset where the curve does not hold it', () => {
    const off = curve({ now: { at: -1, value: 0, day: '' } })

    expect(getKnobPosition(off, settings({ minutesADay: 21 }), today)).toBe(2)
  })
})

describe('a curve that landed', () => {
  it('is drawn as the application’s own, with what it counted', () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)
    const landed = curve()

    applyCurveAnswer(state, landed)

    expect(state.curve.value).toBe(landed)
    expect(state.material.value).toStrictEqual({ decks: 2, cards: 40, overdue: 3, unbegun: 7 })
    expect(state.isReal).toBe(true)
    expect(state.isWaiting.value).toBe(false)
  })
})

describe('asking for a curve', () => {
  it('draws the sketch meanwhile, the knob where the preset stands on it', async () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)
    state.curve.value = curve()
    const gate = createGate()
    const { core } = createCore(gate.answered)
    const sketch = estimateCurve(DEFAULTS, today, BOUNDS)

    const drawn = updateCurves(
      state,
      PATH,
      () => DEFAULTS,
      core,
      BOUNDS,
      today,
      () => {},
    )
    await nap()

    expect(state.curve.value).toStrictEqual(sketch)
    expect(sketch.now.at).toBeGreaterThan(0)
    expect(state.place.value).toBe(sketch.now.at)
    gate.open(curve())
    await drawn
  })

  it('draws the sketch for a goal other than the one the landed curve was for', async () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)
    applyCurveAnswer(state, curve({ goal: 'retention' }))
    const gate = createGate()
    const { core } = createCore(gate.answered)

    const drawn = updateCurves(
      state,
      PATH,
      () => DEFAULTS,
      core,
      BOUNDS,
      today,
      () => {},
    )
    await nap()

    expect(state.curve.value).toStrictEqual(estimateCurve(DEFAULTS, today, BOUNDS))
    gate.open(curve())
    await drawn
  })

  it('keeps the landed curve for the same goal, marked as not the answer to these settings', async () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)
    applyCurveAnswer(state, curve())
    const gate = createGate()
    const { core } = createCore(gate.answered)

    const drawn = updateCurves(
      state,
      PATH,
      () => DEFAULTS,
      core,
      BOUNDS,
      today,
      () => {},
    )
    await nap()

    expect(state.curve.value).toStrictEqual(curve({ isHonest: false }))
    expect(state.isReal).toBe(false)
    expect(state.isWaiting.value).toBe(true)
    gate.open(curve())
    await drawn
  })

  it('draws an answer already held without asking again', async () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)
    const held = curve({ now: { at: 3, value: 30, day: '' } })
    state.answers.set(getShapeKey(DEFAULTS), held)
    const { core, asked } = createCore()

    await updateCurves(
      state,
      PATH,
      () => DEFAULTS,
      core,
      BOUNDS,
      today,
      () => {},
    )

    expect(asked).toStrictEqual([])
    expect(state.curve.value).toBe(held)
    expect(state.isReal).toBe(true)
    expect(state.place.value).toBe(3)
  })

  it('asks once more for settings changed while a question was out', async () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)
    const gate = createGate()
    const { core, asked } = createCore(gate.answered)
    let now = DEFAULTS

    const drawn = updateCurves(
      state,
      PATH,
      () => now,
      core,
      BOUNDS,
      today,
      () => {},
    )
    now = settings({ minutesADay: 30 })
    await updateCurves(
      state,
      PATH,
      () => now,
      core,
      BOUNDS,
      today,
      () => {},
    )
    gate.open(curve())
    await drawn

    expect(asked).toStrictEqual([DEFAULTS, now])
    expect(state.shouldDrawAgain).toBe(false)
  })
})

describe('an answer nobody is waiting for any more', () => {
  it('is not drawn over an answer already held', async () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)
    const held = curve({ cards: 99 })
    const other = settings({ minutesADay: 30 })
    state.answers.set(getShapeKey(other), held)
    const gate = createGate()
    const { core } = createCore(gate.answered)
    let now = DEFAULTS

    const drawn = updateCurves(
      state,
      PATH,
      () => now,
      core,
      BOUNDS,
      today,
      () => {},
    )
    now = other
    await updateCurves(
      state,
      PATH,
      () => now,
      core,
      BOUNDS,
      today,
      () => {},
    )
    gate.open(curve({ cards: 1 }))
    await drawn

    expect(state.curve.value).toBe(held)
  })

  it('says nothing about a question that failed', async () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)
    const other = settings({ minutesADay: 30 })
    state.answers.set(getShapeKey(other), curve())
    const gate = createGate()
    const { core } = createCore(gate.answered)
    const said: string[] = []
    let now = DEFAULTS

    const drawn = updateCurves(
      state,
      PATH,
      () => now,
      core,
      BOUNDS,
      today,
      (one: string) => void said.push(one),
    )
    now = other
    await updateCurves(
      state,
      PATH,
      () => now,
      core,
      BOUNDS,
      today,
      () => {},
    )
    gate.fail(new Error('gone'))
    await drawn

    expect(said).toStrictEqual([])
  })

  it('is said where the question still stands', async () => {
    const state = createCurveState(DEFAULTS, today, BOUNDS)
    const gate = createGate()
    const { core } = createCore(gate.answered)
    const said: string[] = []

    const drawn = updateCurves(
      state,
      PATH,
      () => DEFAULTS,
      core,
      BOUNDS,
      today,
      (one: string) => void said.push(one),
    )
    gate.fail(new Error('gone'))
    await drawn

    expect(said).toStrictEqual([words.noCurve])
    expect(state.isWaiting.value).toBe(false)
  })
})
