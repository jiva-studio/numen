/**
 * How a preset writes the share of a day's load each day of the week carries.
 *
 * The whole of a day is what a day nothing was said about carries, so the file
 * names only the days standing under it. That is the file's own way of writing
 * it, and nothing that draws a week knows of it.
 */
import { describe, expect, it } from 'vitest'
import { DEFAULTS, loadOn, loaded, LOADS, WHOLE_LOAD } from './core'
import type { BudgetUnit, Goal, Rule } from './core'

/**
 * The defaults are read here and again in the core, which schedules a deck
 * naming no preset by them. Both read this one corpus, and neither owns it.
 */
import corpus from '../../../../../libs/protocol/testdata/presets.json'

/**
 * The key a preset file writes each word under, which is the words the corpus
 * is written in. Every word the window has stands here, so one added to it has
 * to be given its key before this compiles.
 */
const GOAL_KEYS: Record<Goal, string> = {
  minutes: 'minutes_a_day',
  retention: 'retention',
  date: 'by_date',
}

const BUDGET_UNIT_KEYS: Record<BudgetUnit, string> = { cards: 'cards', shows: 'shows' }

const RULE_KEYS: Record<Rule, string> = { interval: 'interval', retention: 'retention' }

describe('a preset naming nothing', () => {
  it('is scheduled by what the corpus says', () => {
    const { hasEvenLoad: _, ...defaults } = DEFAULTS
    expect({
      ...defaults,
      goal: GOAL_KEYS[DEFAULTS.goal],
      counts: BUDGET_UNIT_KEYS[DEFAULTS.counts],
      learned: RULE_KEYS[DEFAULTS.learned],
    }).toStrictEqual(corpus.defaults)
  })
})

describe('what one day carries', () => {
  it('is the whole of a day for a day nothing was said about', () => {
    expect(loadOn({}, 'mon')).toBe(WHOLE_LOAD)
    expect(loadOn({ sat: 50 }, 'mon')).toBe(WHOLE_LOAD)
    expect(loadOn({ sat: 50 }, 'sat')).toBe(50)
  })

  it('is nothing where a day is put at nothing, which is not the whole of it', () => {
    expect(loadOn({ sun: 0 }, 'sun')).toBe(0)
  })
})

describe('a day put at a share', () => {
  it('names a day standing under the whole, and drops one put back to it', () => {
    expect(loaded({}, 'sat', 50)).toEqual({ sat: 50 })
    expect(loaded({ sat: 50 }, 'sat', 0)).toEqual({ sat: 0 })
    expect(loaded({ sat: 50, sun: 0 }, 'sat', WHOLE_LOAD)).toEqual({ sun: 0 })
    expect(loaded({}, 'sat', WHOLE_LOAD)).toEqual({})
  })

  it('leaves every other day where it stood', () => {
    expect(loaded({ sat: 50, sun: 0 }, 'mon', 25)).toEqual({ sat: 50, sun: 0, mon: 25 })
  })
})

describe('the whole of a day’s load', () => {
  it('is what the corpus says it is', () => {
    expect(WHOLE_LOAD).toBe(corpus.fullLoad)
  })
})

describe('the shares a day is offered', () => {
  it('run from nothing to the whole of a day, in order', () => {
    expect(LOADS[0]).toBe(0)
    expect(LOADS.at(-1)).toBe(WHOLE_LOAD)
    expect([...LOADS].sort((one, two) => one - two)).toEqual([...LOADS])
  })
})
