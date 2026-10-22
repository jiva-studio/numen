/**
 * What the curve of a preset evaluates to, and where the knob relies on it: where a
 * curve produces.
 */
import { describe, expect, it } from 'vitest'

import { DEFAULTS, NO_CURVE_POINT, type Curve, type Point, type Settings } from '../types'
import { BOUNDS, curve as drawnCurve } from '../fixtures'
import {
  clamp,
  clearBacklog,
  getPointCost,
  findNearest,
  getGoalValue,
  getIdleReason,
  produceSchedule,
  round,
  getKnobValue,
} from './curve'

const today = '2026-08-30'

const settings = (overrides: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...overrides })

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

describe('where a value relies on a grid', () => {
  it('snaps to nearest point', () => {
    expect(findNearest([0, 10, 20, 30], 21)).toBe(2)
    expect(findNearest([0, 10, 20, 30], 30)).toBe(3)
    expect(findNearest([0, 10, 20, 30], -5)).toBe(0)
  })

  it('is nowhere at all on a grid with no places', () => {
    expect(findNearest([], 3)).toBe(-1)
  })
})

describe('where the preset itself remains', () => {
  it('is the value under the key its goal names', () => {
    expect(getGoalValue(settings({ minutesADay: 25 }), today)).toBe(25)
    expect(getGoalValue(settings({ goal: 'retention', retention: 0.85 }), today)).toBe(0.85)
    expect(getGoalValue(settings({ goal: 'date', byDate: '2026-09-29' }), today)).toBe(30)
  })
})

// The knob's own readout and the bubble over it say one number, and it is this
// one: the preset's own value is not a place of the grid, and rounding it onto
// one would read out a value nobody set.
describe('the value the knob remains at', () => {
  const riding = drawnCurve({ now: { at: 2, value: 21, day: '' } })

  it('is the preset’s own where the knob has not been moved off it', () => {
    expect(getKnobValue(riding, 2)).toBe(21)
  })

  it('is the place itself anywhere else, and where the preset falls outside the grid', () => {
    expect(getKnobValue(riding, 3)).toBe(30)
    expect(getKnobValue(drawnCurve({ now: NO_CURVE_POINT }), 2)).toBe(20)
  })

  it('is nothing at all where the grid has no such place', () => {
    expect(getKnobValue(riding, 9)).toBe(0)
  })
})

describe('a number retained inside the bounds of its setting', () => {
  it('is the number where it is inside them, and the end it is past where it is not', () => {
    expect(clamp(0.85, BOUNDS.retention)).toBe(0.85)
    expect(clamp(0.5, BOUNDS.retention)).toBe(0.7)
    expect(clamp(2, BOUNDS.retention)).toBe(0.99)
    expect(clamp(-4, BOUNDS.minutesADay)).toBe(0)
  })

  // A tab draws its fields before the first read lands, and a number typed
  // into one of them is nobody's to bring in until the application has said
  // how far it goes.
  it('is the number itself where the application has returned no bound', () => {
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
    suggested: NO_CURVE_POINT,
    decks: 1,
    cards: 400,
    overdue: 0,
    unbegun: 0,
    isHonest: true,
  }

  // The control moves the one value its goal names. What that evaluates to on this
  // vault is said in a sentence and becomes no setting, so no number the person
  // did not name can close their day.
  it('takes the goal’s own value off the grid, and writes nothing else', () => {
    const was = settings({ newADay: 12, reviewsADay: 0, retention: 0.95 })
    const createdItem = produceSchedule(was, 3, { curve, today, within: BOUNDS })
    expect(createdItem.minutesADay).toBe(30)
    expect(createdItem.newADay).toBe(12)
    expect(createdItem.reviewsADay).toBe(0)
    expect(createdItem.retention).toBe(0.95)
  })

  it('leaves the settings as they are where the grid has no such place', () => {
    expect(produceSchedule(settings(), 9, { curve, today, within: BOUNDS })).toStrictEqual(
      settings(),
    )
  })

  it('names the day of the place for a goal of a date, and nothing else', () => {
    const dated: Curve = {
      ...curve,
      goal: 'date',
      days: ['2026-08-31', '2026-09-09', '2026-09-19', '2026-09-29'],
      at: curve.at.map((one) => ({ ...one, minutes: 45 })),
    }
    const was = settings({ goal: 'date', minutesADay: 20, reviewsADay: 0 })
    const createdItem = produceSchedule(was, 3, { curve: dated, today, within: BOUNDS })
    expect(createdItem.byDate).toBe('2026-09-29')
    expect(createdItem.minutesADay).toBe(20)
    expect(createdItem.reviewsADay).toBe(0)
  })

  it('moves the target alone under a goal of retention', () => {
    const was = settings({ goal: 'retention', newADay: 12, reviewsADay: 30 })
    const createdItem = produceSchedule(was, 3, {
      curve: { ...curve, goal: 'retention' },
      today,
      within: BOUNDS,
    })
    expect(createdItem.retention).toBe(0.99)
    expect(createdItem.newADay).toBe(12)
    expect(createdItem.reviewsADay).toBe(30)
  })

  it('leaves the knob, the field and what is written at one value', () => {
    const createdItem = produceSchedule(settings(), 2, { curve, today, within: BOUNDS })
    expect(createdItem.minutesADay).toBe(curve.grid[2])
  })
})

describe('a goal with nothing to work on', () => {
  const nothing: Curve = {
    goal: 'minutes',
    grid: [0, 10, 20],
    days: [],
    at: [point(), point(), point()],
    now: NO_CURVE_POINT,
    suggested: NO_CURVE_POINT,
    decks: 0,
    cards: 0,
    overdue: 0,
    unbegun: 0,
    isHonest: true,
  }

  it('is a preset no deck points at, where the curve carries no deck', () => {
    expect(getIdleReason(nothing)).toBe('unpointed')
  })

  it('is decks holding nothing between them, where they do point here', () => {
    expect(getIdleReason({ ...nothing, decks: 1 })).toBe('noCards')
    expect(getIdleReason({ ...nothing, decks: 4 })).toBe('noCards')
  })

  // A curve of zeros is a day behind us, a range of nothing, or a question the
  // application would not answer. None of those is an empty preset: the count
  // of cards says that and nothing else does.
  it('is not a preset holding cards, whatever its curve evaluates to', () => {
    expect(getIdleReason({ ...nothing, decks: 4, cards: 900 })).toBe('')
    expect(getIdleReason({ ...nothing, decks: 4, cards: 900, goal: 'date', grid: [1, 2, 3] })).toBe(
      '',
    )
  })

  it('is not the line the window guessed, which is nobody’s answer', () => {
    expect(getIdleReason({ ...nothing, isHonest: false })).toBe('')
    expect(getIdleReason({ ...nothing, decks: 4, isHonest: false })).toBe('')
  })

  // The preset is not stopped: it schedules reviews, and nothing here can ever
  // become one.
  it('is a material nobody has begun that no place of the range begins', () => {
    const all = { ...nothing, decks: 1, cards: 900, unbegun: 900 }
    expect(getIdleReason(all)).toBe('beginsNothing')
  })

  it('is not a material a place of the range asks for', () => {
    const all = { ...nothing, decks: 1, cards: 900, unbegun: 900 }
    expect(getIdleReason({ ...all, at: [point(), point({ reviews: 12 }), point()] })).toBe('')
  })

  it('is not a material some of which somebody has begun', () => {
    const some = { ...nothing, decks: 1, cards: 900, unbegun: 899 }
    expect(getIdleReason(some)).toBe('')
  })
})

describe('a number to that many places', () => {
  it('rounds and does not truncate', () => {
    expect(round(0.876, 2)).toBe(0.88)
    expect(round(0.874, 2)).toBe(0.87)
  })
})

describe('what a curve is read in', () => {
  it('is the cards a day answers under a goal of minutes, and the minutes it costs under the others', () => {
    const one = point({ reviews: 40, minutes: 15 })

    expect(getPointCost('minutes', one)).toBe(40)
    expect(getPointCost('retention', one)).toBe(15)
    expect(getPointCost('date', one)).toBe(15)
  })
})

describe('the day the overdue pile is gone', () => {
  it('is nothing where nothing is overdue', () => {
    expect(clearBacklog([0, 0, 0])).toBeNull()
    expect(clearBacklog([])).toBeNull()
  })

  it('counts days from one, so a pile gone on the first day is day one', () => {
    expect(clearBacklog([0, 0])).toBeNull()
    expect(clearBacklog([5, 0])).toBe(2)
    expect(clearBacklog([5, 3, 0, 0])).toBe(3)
  })

  it('is -1 where the pile relies on the last day projected', () => {
    expect(clearBacklog([5, 3, 1])).toBe(-1)
  })

  it('is day one where the first place of the projection is already empty', () => {
    expect(clearBacklog([0, 4, 0])).toBe(1)
  })
})

describe('where a value relies on a grid, at the edges', () => {
  it('is the first place of a tie between two', () => {
    expect(findNearest([0, 10], 5)).toBe(0)
  })

  it('is the first place for a value nearer the start than any place after it', () => {
    expect(findNearest([10, 20], 0)).toBe(0)
  })
})

describe('the value the knob remains at, at the edges', () => {
  it('is the preset’s own at the first place', () => {
    expect(getKnobValue(drawnCurve({ now: { at: 0, value: 7, day: '' } }), 0)).toBe(7)
  })

  it('is nothing at no place, where the preset falls outside the grid', () => {
    expect(getKnobValue(drawnCurve({ now: { at: -1, value: 7, day: '' } }), -1)).toBe(0)
  })
})

describe('a material nobody has begun, where the range has no place', () => {
  it('is not a reason to say the preset begins nothing', () => {
    const none: Curve = {
      goal: 'minutes',
      grid: [],
      days: [],
      at: [],
      now: NO_CURVE_POINT,
      suggested: NO_CURVE_POINT,
      decks: 1,
      cards: 900,
      overdue: 0,
      unbegun: 900,
      isHonest: true,
    }

    expect(getIdleReason(none)).toBe('')
  })
})

describe('the day a place of a curve of a date produces', () => {
  const dated: Curve = {
    goal: 'date',
    grid: [0, 10, 20, 30],
    days: ['2027-01-01', '2027-01-02', '2027-01-03', '2027-01-04'],
    at: [point(), point(), point(), point()],
    now: NO_CURVE_POINT,
    suggested: NO_CURVE_POINT,
    decks: 1,
    cards: 400,
    overdue: 0,
    unbegun: 0,
    isHonest: true,
  }
  const was = settings({ goal: 'date', byDate: '2026-09-01' })

  it('is the day the curve names for it, whatever the grid says', () => {
    expect(produceSchedule(was, 3, { curve: dated, today, within: BOUNDS }).byDate).toBe(
      '2027-01-04',
    )
  })

  it('is the day the grid’s value evaluates to where the curve names none', () => {
    expect(
      produceSchedule(was, 3, { curve: { ...dated, days: [] }, today, within: BOUNDS }).byDate,
    ).toBe('2026-09-29')
  })
})
