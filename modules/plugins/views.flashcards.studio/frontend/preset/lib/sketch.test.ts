/**
 * The line the window draws while the honest one is on its way.
 */
import { describe, expect, it } from 'vitest'

import { DEFAULTS, NO_BOUNDS, NO_CURVE_POINT, type Settings } from '../types'
import { BOUNDS } from '../fixtures'
import { getPointCost } from './curve'
import { estimateCurve } from './sketch'

/**
 * The sketch is drawn across the span the core holds a target to, before any
 * answer from it has landed. Both read this one corpus, and neither owns it.
 */
import corpus from '../../../../../libs/protocol/testdata/presets.json'

const today = '2026-08-30'

const settings = (over: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...over })

describe('the line the window draws in the answer’s place', () => {
  it('says it is not the application’s', () => {
    expect(estimateCurve(settings(), today, BOUNDS).isHonest).toBe(false)
  })

  it('runs over the whole range of minutes, and marks where the preset remains', () => {
    const guess = estimateCurve(settings({ minutesADay: 20 }), today, BOUNDS)
    expect(guess.grid[0]).toBe(0)
    expect(guess.grid[guess.grid.length - 1]).toBe(60)
    expect(guess.grid[guess.now.at]).toBe(20)
    expect(guess.suggested).toStrictEqual(NO_CURVE_POINT)
  })

  it('brings back more of the material the longer the day runs', () => {
    const guess = estimateCurve(settings(), today, BOUNDS)
    const costs = guess.at.map((one) => getPointCost('minutes', one))
    expect(costs.every((cost, at) => at === 0 || cost >= (costs[at - 1] ?? 0))).toBe(true)
  })

  it('costs more of the day the more of the material is requested back', () => {
    const guess = estimateCurve(settings({ goal: 'retention', retention: 0.85 }), today, BOUNDS)
    expect(guess.grid[0]).toBe(0.7)
    expect(guess.grid[guess.grid.length - 1]).toBe(0.99)
    const costs = guess.at.map((one) => getPointCost('retention', one))
    expect(costs.every((cost, at) => at === 0 || cost >= (costs[at - 1] ?? 0))).toBe(true)
  })

  it('names a day at every place of a goal of a date', () => {
    const guess = estimateCurve(settings({ goal: 'date', byDate: '2026-09-29' }), today, BOUNDS)
    expect(guess.days).toHaveLength(guess.grid.length)
    expect(guess.days[guess.now.at]).toBe('2026-09-29')
  })
})

describe('the span the sketch of a target is drawn across', () => {
  it('is the one the application answered with, and not one kept here', () => {
    const response = { ...BOUNDS, retention: { least: 0.75, most: 0.95 } }
    const guess = estimateCurve(settings({ goal: 'retention' }), today, response)
    expect(guess.grid[0]).toBe(0.75)
    expect(guess.grid.at(-1)).toBe(0.95)
  })

  it('answers the corpus, which is where that span is written down', () => {
    const guess = estimateCurve(settings({ goal: 'retention' }), today, BOUNDS)
    expect({ least: guess.grid[0], most: guess.grid.at(-1) }).toStrictEqual(corpus.retentionBounds)
  })

  it('has no places at all until the application has returned how far a target goes', () => {
    expect(estimateCurve(settings({ goal: 'retention' }), today, NO_BOUNDS).grid).toStrictEqual([])
  })
})

/** A place of the sketch with nothing on it but what the test names. */
const flat = {
  retained: 0,
  owed: 0,
  through: 0,
  canLearnEveryCard: true,
  closed: [],
  clears: 0,
  learned: 0,
  short: 0,
  backlog: [],
}

describe('where a sketch says the preset remains', () => {
  it('names no day under a goal that is not a date', () => {
    const guess = estimateCurve(settings({ minutesADay: 20 }), today, BOUNDS)
    expect(guess.days).toStrictEqual([])
    expect(guess.now).toStrictEqual({ at: 8, value: 20, day: '' })
  })

  it('names the day the preset aims at under a goal of a date', () => {
    const guess = estimateCurve(settings({ goal: 'date', byDate: '2026-09-29' }), today, BOUNDS)
    expect(guess.now.day).toBe('2026-09-29')
  })
})

describe('the days a sketch of a date runs over', () => {
  it('runs from tomorrow to the day aimed at', () => {
    const guess = estimateCurve(settings({ goal: 'date', byDate: '2026-11-28' }), today, BOUNDS)
    expect(guess.grid[0]).toBe(1)
    expect(guess.grid.at(-1)).toBe(90)
  })

  it('runs a month at the least, however near the day aimed at is', () => {
    const guess = estimateCurve(settings({ goal: 'date', byDate: '2026-09-09' }), today, BOUNDS)
    expect(guess.grid[0]).toBe(1)
    expect(guess.grid.at(-1)).toBe(30)
  })
})

describe('one place of a sketch of minutes', () => {
  it('answers a card every eight seconds and keeps more the longer the day', () => {
    const guess = estimateCurve(settings({ minutesADay: 20 }), today, BOUNDS)
    expect(guess.grid[4]).toBe(10)
    expect(guess.at[4]).toStrictEqual({
      ...flat,
      minutes: 10,
      reviews: 75,
      retained: expect.closeTo(0.392, 9),
    })
  })
})

describe('one place of a sketch of a target', () => {
  it('costs the day’s minutes at ninety percent, and more the more is requested back', () => {
    const guess = estimateCurve(
      settings({ goal: 'retention', minutesADay: 30, retention: 0.8 }),
      today,
      BOUNDS,
    )
    expect(guess.grid[0]).toBe(0.7)
    expect(guess.at[0]).toStrictEqual({
      ...flat,
      minutes: expect.closeTo(10, 9),
      reviews: expect.closeTo(75, 9),
      retained: 0.7,
    })
  })

  it('reads a day of no minutes as the standing budget', () => {
    const guess = estimateCurve(settings({ goal: 'retention', minutesADay: 0 }), today, BOUNDS)
    expect(guess.at[0]?.minutes).toBeCloseTo(20 / 3, 9)
  })
})

describe('one place of a sketch of a date', () => {
  const dated = (minutesADay: number) =>
    estimateCurve(settings({ goal: 'date', byDate: '2026-10-29', minutesADay }), today, BOUNDS)

  it('learns every card on the day aimed at within the day’s minutes', () => {
    const guess = dated(30)
    expect(guess.grid.at(-1)).toBe(60)
    expect(guess.at.at(-1)).toStrictEqual({
      ...flat,
      minutes: 30,
      reviews: 225,
      through: 1,
      canLearnEveryCard: true,
    })
  })

  it('costs more than the day holds to be through by tomorrow', () => {
    const guess = dated(30)
    expect(guess.grid[0]).toBe(1)
    expect(guess.at[0]).toStrictEqual({
      ...flat,
      minutes: 1800,
      reviews: 13500,
      through: expect.closeTo(1 / 60, 9),
      canLearnEveryCard: false,
    })
  })

  it('reads a day of no minutes as the standing budget', () => {
    const guess = dated(0)
    expect(guess.at.at(-1)?.minutes).toBe(DEFAULTS.minutesADay)
    expect(guess.at.at(-1)?.canLearnEveryCard).toBe(true)
  })
})
