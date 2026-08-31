/**
 * The arithmetic behind the one control: where a value stands on the grid, the
 * line the window draws while the honest one is on its way, and what a place of
 * the curve produces.
 */
import { describe, expect, it } from 'vitest'

import { DEFAULTS, NOWHERE, type Curve, type Point, type Settings } from './core'
import {
  approximate,
  closed,
  costOf,
  dayAfter,
  daysUntil,
  FIELDS,
  held,
  nearest,
  paused,
  placeAt,
  fieldsUnder,
  idle,
  producing,
  round,
  spent,
  standing,
} from './curve'

const today = new Date('2026-08-30T00:00:00Z')

const settings = (over: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...over })

const point = (over: Partial<Point> = {}): Point => ({
  reviews: 0,
  minutes: 0,
  retained: 0,
  owed: 0,
  through: 0,
  enough: true,
  met: true,
  clears: 0,
  ...over,
})

describe('where a value stands on a grid', () => {
  it('is the nearest place, whether the value sits on one or between two', () => {
    expect(nearest([0, 10, 20, 30], 21)).toBe(2)
    expect(nearest([0, 10, 20, 30], 30)).toBe(3)
    expect(nearest([0, 10, 20, 30], -5)).toBe(0)
  })

  it('is nowhere at all on a grid with no places', () => {
    expect(nearest([], 3)).toBe(-1)
    expect(placeAt([], 0.5)).toBe(-1)
  })

  it('is held inside the grid wherever a share of the way along falls', () => {
    expect(placeAt([0, 1, 2, 3, 4], 0.5)).toBe(2)
    expect(placeAt([0, 1, 2, 3, 4], -1)).toBe(0)
    expect(placeAt([0, 1, 2, 3, 4], 2)).toBe(4)
  })
})

describe('the days a goal of a date counts', () => {
  it('counts forward from today and back again', () => {
    expect(dayAfter(today, 30)).toBe('2026-09-29')
    expect(daysUntil(today, '2026-09-29')).toBe(30)
    expect(daysUntil(today, '2026-08-01')).toBe(-29)
  })

  it('counts nothing for a day that is not one', () => {
    expect(daysUntil(today, '')).toBe(0)
    expect(daysUntil(today, 'the day after tomorrow')).toBe(0)
  })
})

describe('a preset that schedules nothing', () => {
  it('is one past the day it aimed at', () => {
    const gone = settings({ goal: 'date', byDate: '2026-08-01' })
    expect(spent(gone, today)).toBe(true)
    expect(paused(gone, today)).toBe(true)
  })

  it('is one holding no cards a day', () => {
    expect(paused(settings({ newADay: 0, reviewsADay: 0 }), today)).toBe(true)
  })

  it('is not one aiming at a day still ahead', () => {
    expect(spent(settings({ goal: 'date', byDate: '2026-09-29' }), today)).toBe(false)
    expect(paused(settings({ goal: 'date', byDate: '2026-09-29' }), today)).toBe(false)
  })
})

describe('the line the window draws in the answer’s place', () => {
  it('says it is not the application’s', () => {
    expect(approximate(settings(), today).honest).toBe(false)
  })

  it('runs over the whole range of minutes, and marks where the preset stands', () => {
    const guess = approximate(settings({ minutesADay: 20 }), today)
    expect(guess.grid[0]).toBe(0)
    expect(guess.grid[guess.grid.length - 1]).toBe(60)
    expect(guess.grid[guess.now.at]).toBe(20)
    expect(guess.suggested).toStrictEqual(NOWHERE)
  })

  it('brings back more of the material the longer the day runs', () => {
    const guess = approximate(settings(), today)
    const costs = guess.at.map((one) => costOf('minutes', one))
    expect(costs.every((cost, at) => at === 0 || cost >= (costs[at - 1] ?? 0))).toBe(true)
  })

  it('costs more of the day the more of the material is asked back', () => {
    const guess = approximate(settings({ goal: 'retention', retention: 0.85 }), today)
    expect(guess.grid[0]).toBe(0.7)
    expect(guess.grid[guess.grid.length - 1]).toBe(0.99)
    const costs = guess.at.map((one) => costOf('retention', one))
    expect(costs.every((cost, at) => at === 0 || cost >= (costs[at - 1] ?? 0))).toBe(true)
  })

  it('names a day at every place of a goal of a date', () => {
    const guess = approximate(settings({ goal: 'date', byDate: '2026-09-29' }), today)
    expect(guess.days).toHaveLength(guess.grid.length)
    expect(guess.days[guess.now.at]).toBe('2026-09-29')
  })
})

describe('where the preset itself stands', () => {
  it('is the value under the key its goal names', () => {
    expect(standing(settings({ minutesADay: 25 }), today)).toBe(25)
    expect(standing(settings({ goal: 'retention', retention: 0.85 }), today)).toBe(0.85)
    expect(standing(settings({ goal: 'date', byDate: '2026-09-29' }), today)).toBe(30)
  })
})

describe('a number held inside the bounds of its setting', () => {
  it('is the number where it is inside them, and the end it is past where it is not', () => {
    expect(held(0.85, 'retention')).toBe(0.85)
    expect(held(0.5, 'retention')).toBe(0.7)
    expect(held(2, 'retention')).toBe(0.99)
    expect(held(-4, 'minutesADay')).toBe(0)
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
    honest: true,
  }

  // The control moves the one value its goal names. What that comes to on this
  // vault is said in a sentence and becomes no setting, so no number the person
  // did not name can close their day.
  it('takes the goal’s own value off the grid, and writes nothing else', () => {
    const was = settings({ newADay: 12, reviewsADay: 0, retention: 0.95 })
    const made = producing(was, 3, curve, today)
    expect(made.minutesADay).toBe(30)
    expect(made.newADay).toBe(12)
    expect(made.reviewsADay).toBe(0)
    expect(made.retention).toBe(0.95)
  })

  it('leaves the settings as they are where the grid has no such place', () => {
    expect(producing(settings(), 9, curve, today)).toStrictEqual(settings())
  })

  it('names the day of the place for a goal of a date, and nothing else', () => {
    const dated: Curve = {
      ...curve,
      goal: 'date',
      days: ['2026-08-31', '2026-09-09', '2026-09-19', '2026-09-29'],
      at: curve.at.map((one) => ({ ...one, minutes: 45 })),
    }
    const was = settings({ goal: 'date', minutesADay: 20, reviewsADay: 0 })
    const made = producing(was, 3, dated, today)
    expect(made.byDate).toBe('2026-09-29')
    expect(made.minutesADay).toBe(20)
    expect(made.reviewsADay).toBe(0)
  })

  it('moves the target alone under a goal of retention', () => {
    const was = settings({ goal: 'retention', newADay: 12, reviewsADay: 30 })
    const made = producing(was, 3, { ...curve, goal: 'retention' }, today)
    expect(made.retention).toBe(0.99)
    expect(made.newADay).toBe(12)
    expect(made.reviewsADay).toBe(30)
  })

  it('leaves the knob, the field and what is written at one value', () => {
    const made = producing(settings(), 2, curve, today)
    expect(made.minutesADay).toBe(curve.grid[2])
  })
})

describe('a day a longer one buys nothing on', () => {
  const over = (cards: readonly number[]): Curve => ({
    goal: 'minutes',
    grid: cards.map((_, at) => at * 10),
    days: [],
    at: cards.map((one) => point({ reviews: one })),
    now: NOWHERE,
    suggested: NOWHERE,
    decks: 1,
    cards: 400,
    overdue: 0,
    honest: true,
  })

  it('is a curve holding the same cards at every place, which the counts close', () => {
    expect(closed(over([13, 13, 13]))).toBe(true)
  })

  it('is not a curve a longer day answers more cards on', () => {
    expect(closed(over([13, 40, 90]))).toBe(false)
  })

  it('is not said of a goal read in minutes, nor of a line the window guessed', () => {
    expect(closed({ ...over([13, 13, 13]), goal: 'retention' })).toBe(false)
    expect(closed({ ...over([13, 13, 13]), honest: false })).toBe(false)
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
    honest: true,
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
    expect(idle({ ...nothing, honest: false })).toBe('')
    expect(idle({ ...nothing, decks: 4, honest: false })).toBe('')
  })
})

describe('the settings a goal schedules by', () => {
  // A goal names one budget, and the budgets of the other two take no part in
  // it: they are not drawn, so nothing on the screen offers to cut the day
  // short by a measure the person did not name.
  it('is the minutes alone under a goal of minutes', () => {
    const under = fieldsUnder('minutes')
    expect(under).toContain('minutesADay')
    expect(under).not.toContain('newADay')
    expect(under).not.toContain('reviewsADay')
    expect(under).not.toContain('retention')
    expect(under).not.toContain('byDate')
  })

  it('is the target and the counts that close a day under a goal of retention', () => {
    const under = fieldsUnder('retention')
    expect(under).toContain('retention')
    expect(under).toContain('newADay')
    expect(under).toContain('reviewsADay')
    expect(under).not.toContain('minutesADay')
    expect(under).not.toContain('byDate')
  })

  it('is the day alone under a goal of a date, which nothing else may cut short', () => {
    const under = fieldsUnder('date')
    expect(under).toContain('byDate')
    expect(under).not.toContain('minutesADay')
    expect(under).not.toContain('newADay')
    expect(under).not.toContain('reviewsADay')
  })

  it('draws what stands under no goal in particular under all three', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      expect(fieldsUnder(goal)).toContain('counts')
      expect(fieldsUnder(goal)).toContain('lightDays')
      expect(fieldsUnder(goal)).toContain('evenLoad')
    }
  })

  it('draws them in the order the receipt keeps them in', () => {
    for (const goal of ['minutes', 'retention', 'date'] as const) {
      const under = fieldsUnder(goal)
      const places = under.map((field) => FIELDS.indexOf(field))
      expect(places).toStrictEqual([...places].sort((a, b) => a - b))
    }
  })
})

describe('a number to that many places', () => {
  it('rounds and does not truncate', () => {
    expect(round(0.876, 2)).toBe(0.88)
    expect(round(0.874, 2)).toBe(0.87)
  })
})
