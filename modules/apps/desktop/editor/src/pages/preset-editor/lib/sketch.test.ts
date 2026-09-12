/**
 * The line the window draws while the honest one is on its way.
 */
import { describe, expect, it } from 'vitest'

import { DEFAULTS, NO_BOUNDS, NOWHERE, type Settings } from '../types'
import { BOUNDS } from '../fixtures'
import { costOf } from './curve'
import { approximate } from './sketch'

/**
 * The sketch is drawn across the span the core holds a target to, before any
 * answer from it has landed. Both read this one corpus, and neither owns it.
 */
import corpus from '../../../../../../../libs/protocol/testdata/presets.json'

/** The review day the window is told, which is what a date is counted from. */
const today = '2026-08-30'

const settings = (over: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...over })

describe('the line the window draws in the answer’s place', () => {
  it('says it is not the application’s', () => {
    expect(approximate(settings(), today, BOUNDS).honest).toBe(false)
  })

  it('runs over the whole range of minutes, and marks where the preset stands', () => {
    const guess = approximate(settings({ minutesADay: 20 }), today, BOUNDS)
    expect(guess.grid[0]).toBe(0)
    expect(guess.grid[guess.grid.length - 1]).toBe(60)
    expect(guess.grid[guess.now.at]).toBe(20)
    expect(guess.suggested).toStrictEqual(NOWHERE)
  })

  it('brings back more of the material the longer the day runs', () => {
    const guess = approximate(settings(), today, BOUNDS)
    const costs = guess.at.map((one) => costOf('minutes', one))
    expect(costs.every((cost, at) => at === 0 || cost >= (costs[at - 1] ?? 0))).toBe(true)
  })

  it('costs more of the day the more of the material is asked back', () => {
    const guess = approximate(settings({ goal: 'retention', retention: 0.85 }), today, BOUNDS)
    expect(guess.grid[0]).toBe(0.7)
    expect(guess.grid[guess.grid.length - 1]).toBe(0.99)
    const costs = guess.at.map((one) => costOf('retention', one))
    expect(costs.every((cost, at) => at === 0 || cost >= (costs[at - 1] ?? 0))).toBe(true)
  })

  it('names a day at every place of a goal of a date', () => {
    const guess = approximate(settings({ goal: 'date', byDate: '2026-09-29' }), today, BOUNDS)
    expect(guess.days).toHaveLength(guess.grid.length)
    expect(guess.days[guess.now.at]).toBe('2026-09-29')
  })
})

describe('the span the sketch of a target is drawn across', () => {
  it('is the one the application answered with, and not one kept here', () => {
    const said = { ...BOUNDS, retention: { least: 0.75, most: 0.95 } }
    const guess = approximate(settings({ goal: 'retention' }), today, said)
    expect(guess.grid[0]).toBe(0.75)
    expect(guess.grid.at(-1)).toBe(0.95)
  })

  it('answers the corpus, which is where that span is written down', () => {
    const guess = approximate(settings({ goal: 'retention' }), today, BOUNDS)
    expect({ least: guess.grid[0], most: guess.grid.at(-1) }).toStrictEqual(corpus.retentionBounds)
  })

  it('has no places at all until the application has said how far a target goes', () => {
    expect(approximate(settings({ goal: 'retention' }), today, NO_BOUNDS).grid).toStrictEqual([])
  })
})
