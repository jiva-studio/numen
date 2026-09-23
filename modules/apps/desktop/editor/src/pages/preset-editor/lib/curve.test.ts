/**
 * What the curve of a preset comes to, and where the knob stands on it: where a
 * value sits on the grid, the value a place reads out, and what a place of the
 * curve produces.
 */
import { describe, expect, it } from 'vitest'

import { DEFAULTS, NOWHERE, type Curve, type Point, type Settings } from '../types'
import { BOUNDS, curve as drawnCurve } from '../fixtures'
import { clamp, goalValue, idle, findNearest, produceSchedule, round, valueAt } from './curve'

/** The review day the window is told, which is what a date is counted from. */
const today = '2026-08-30'

const settings = (over: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...over })

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

describe('where a value stands on a grid', () => {
  it('is the nearest place, whether the value sits on one or between two', () => {
    expect(findNearest([0, 10, 20, 30], 21)).toBe(2)
    expect(findNearest([0, 10, 20, 30], 30)).toBe(3)
    expect(findNearest([0, 10, 20, 30], -5)).toBe(0)
  })

  it('is nowhere at all on a grid with no places', () => {
    expect(findNearest([], 3)).toBe(-1)
  })
})

describe('where the preset itself stands', () => {
  it('is the value under the key its goal names', () => {
    expect(goalValue(settings({ minutesADay: 25 }), today)).toBe(25)
    expect(goalValue(settings({ goal: 'retention', retention: 0.85 }), today)).toBe(0.85)
    expect(goalValue(settings({ goal: 'date', byDate: '2026-09-29' }), today)).toBe(30)
  })
})

// The knob's own readout and the bubble over it say one number, and it is this
// one: the preset's own value is not a place of the grid, and rounding it onto
// one would read out a value nobody set.
describe('the value the knob stands at', () => {
  const riding = drawnCurve({ now: { at: 2, value: 21, day: '' } })

  it('is the preset’s own where the knob has not been moved off it', () => {
    expect(valueAt(riding, 2)).toBe(21)
  })

  it('is the place itself anywhere else, and where the preset falls outside the grid', () => {
    expect(valueAt(riding, 3)).toBe(30)
    expect(valueAt(drawnCurve({ now: NOWHERE }), 2)).toBe(20)
  })

  it('is nothing at all where the grid has no such place', () => {
    expect(valueAt(riding, 9)).toBe(0)
  })
})

describe('a number held inside the bounds of its setting', () => {
  it('is the number where it is inside them, and the end it is past where it is not', () => {
    expect(clamp(0.85, BOUNDS.retention)).toBe(0.85)
    expect(clamp(0.5, BOUNDS.retention)).toBe(0.7)
    expect(clamp(2, BOUNDS.retention)).toBe(0.99)
    expect(clamp(-4, BOUNDS.minutesADay)).toBe(0)
  })

  // A tab draws its fields before the first read lands, and a number typed
  // into one of them is nobody's to bring in until the application has said
  // how far it goes.
  it('is the number itself where the application has said no bound', () => {
    expect(clamp(-4, undefined)).toBe(-4)
    expect(clamp(9_000, undefined)).toBe(9_000)
  })
})

describe('what one place of the curve produces', () => {
  const curve: Curve = {
    goal: 'minutes',
    grid: [0, 10, 20, 30],
    days: [],
    at: [
      point(),
      point({ reviews: 40, retained: 0.8 }),
      point({ reviews: 80, retained: 0.88 }),
      point({ reviews: 120, retained: 0.93 }),
    ],
    now: { at: 2, value: 20, day: '' },
    suggested: NOWHERE,
    decks: 1,
    cards: 400,
    overdue: 0,
    unbegun: 0,
    isHonest: true,
  }

  // The control moves the one value its goal names. What that comes to on this
  // vault is said in a sentence and becomes no setting, so no number the person
  // did not name can close their day.
  it('takes the goal’s own value off the grid, and writes nothing else', () => {
    const was = settings({ newADay: 12, reviewsADay: 0, retention: 0.95 })
    const made = produceSchedule(was, 3, curve, today, BOUNDS)
    expect(made.minutesADay).toBe(30)
    expect(made.newADay).toBe(12)
    expect(made.reviewsADay).toBe(0)
    expect(made.retention).toBe(0.95)
  })

  it('leaves the settings as they are where the grid has no such place', () => {
    expect(produceSchedule(settings(), 9, curve, today, BOUNDS)).toStrictEqual(settings())
  })

  it('names the day of the place for a goal of a date, and nothing else', () => {
    const dated: Curve = {
      ...curve,
      goal: 'date',
      days: ['2026-08-31', '2026-09-09', '2026-09-19', '2026-09-29'],
      at: curve.at.map((one) => ({ ...one, minutes: 45 })),
    }
    const was = settings({ goal: 'date', minutesADay: 20, reviewsADay: 0 })
    const made = produceSchedule(was, 3, dated, today, BOUNDS)
    expect(made.byDate).toBe('2026-09-29')
    expect(made.minutesADay).toBe(20)
    expect(made.reviewsADay).toBe(0)
  })

  it('moves the target alone under a goal of retention', () => {
    const was = settings({ goal: 'retention', newADay: 12, reviewsADay: 30 })
    const made = produceSchedule(was, 3, { ...curve, goal: 'retention' }, today, BOUNDS)
    expect(made.retention).toBe(0.99)
    expect(made.newADay).toBe(12)
    expect(made.reviewsADay).toBe(30)
  })

  it('leaves the knob, the field and what is written at one value', () => {
    const made = produceSchedule(settings(), 2, curve, today, BOUNDS)
    expect(made.minutesADay).toBe(curve.grid[2])
  })
})

describe('a goal with nothing to work on', () => {
  const nothing: Curve = {
    goal: 'minutes',
    grid: [0, 10, 20],
    days: [],
    at: [point(), point(), point()],
    now: NOWHERE,
    suggested: NOWHERE,
    decks: 0,
    cards: 0,
    overdue: 0,
    unbegun: 0,
    isHonest: true,
  }

  it('is a preset no deck points at, where the curve carries no deck', () => {
    expect(idle(nothing)).toBe('unpointed')
  })

  it('is decks holding nothing between them, where they do point here', () => {
    expect(idle({ ...nothing, decks: 1 })).toBe('noCards')
    expect(idle({ ...nothing, decks: 4 })).toBe('noCards')
  })

  // A curve of zeros is a day behind us, a range of nothing, or a question the
  // application would not answer. None of those is an empty preset: the count
  // of cards says that and nothing else does.
  it('is not a preset holding cards, whatever its curve comes to', () => {
    expect(idle({ ...nothing, decks: 4, cards: 900 })).toBe('')
    expect(idle({ ...nothing, decks: 4, cards: 900, goal: 'date', grid: [1, 2, 3] })).toBe('')
  })

  it('is not the line the window guessed, which is nobody’s answer', () => {
    expect(idle({ ...nothing, isHonest: false })).toBe('')
    expect(idle({ ...nothing, decks: 4, isHonest: false })).toBe('')
  })

  // The preset is not stopped: it schedules reviews, and nothing here can ever
  // become one.
  it('is a material nobody has begun that no place of the range begins', () => {
    const all = { ...nothing, decks: 1, cards: 900, unbegun: 900 }
    expect(idle(all)).toBe('beginsNothing')
  })

  it('is not a material a place of the range asks for', () => {
    const all = { ...nothing, decks: 1, cards: 900, unbegun: 900 }
    expect(idle({ ...all, at: [point(), point({ reviews: 12 }), point()] })).toBe('')
  })

  it('is not a material some of which somebody has begun', () => {
    const some = { ...nothing, decks: 1, cards: 900, unbegun: 899 }
    expect(idle(some)).toBe('')
  })
})

describe('a number to that many places', () => {
  it('rounds and does not truncate', () => {
    expect(round(0.876, 2)).toBe(0.88)
    expect(round(0.874, 2)).toBe(0.87)
  })
})
