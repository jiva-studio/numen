/**
 * How a preset writes the share of a day's load that each day of the week
 * carries.
 *
 * A day with no entry carries the whole load, so the file names only the days
 * with a smaller share. That is the file's own format, and nothing that draws a
 * week depends on it.
 */
import { describe, expect, it } from 'vitest'
import { DEFAULTS, getLoadOn, setLoadOn, LOADS, WHOLE_LOAD } from './presets'
import type { BudgetUnit, Goal, Rule } from './presets'

/**
 * The defaults are read here and again in the core, which schedules a deck
 * naming no preset by them. Both read this one corpus, and neither owns it.
 */
import corpus from '../../testdata/presets.json'

/**
 * The key a preset file writes each goal word under. Every goal the window has
 * is listed here, so a goal added to it requires a key before this compiles.
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
    expect({
      ...DEFAULTS,
      goal: GOAL_KEYS[DEFAULTS.goal],
      counts: BUDGET_UNIT_KEYS[DEFAULTS.counts],
      learned: RULE_KEYS[DEFAULTS.learned],
    }).toStrictEqual(corpus.defaults)
  })
})

describe('what one day carries', () => {
  it('is the whole of a day for a day nothing was returned about', () => {
    expect(getLoadOn({}, 'mon')).toBe(WHOLE_LOAD)
    expect(getLoadOn({ sat: 50 }, 'mon')).toBe(WHOLE_LOAD)
    expect(getLoadOn({ sat: 50 }, 'sat')).toBe(50)
  })

  it('is nothing where a day is put at nothing, which is not the whole of it', () => {
    expect(getLoadOn({ sun: 0 }, 'sun')).toBe(0)
  })
})

describe('a day put at a share', () => {
  it('names a day standing under the whole, and drops one put back to it', () => {
    expect(setLoadOn({}, 'sat', 50)).toEqual({ sat: 50 })
    expect(setLoadOn({ sat: 50 }, 'sat', 0)).toEqual({ sat: 0 })
    expect(setLoadOn({ sat: 50, sun: 0 }, 'sat', WHOLE_LOAD)).toEqual({ sun: 0 })
    expect(setLoadOn({}, 'sat', WHOLE_LOAD)).toEqual({})
  })

  it('leaves every other day where it stood', () => {
    expect(setLoadOn({ sat: 50, sun: 0 }, 'mon', 25)).toEqual({ sat: 50, sun: 0, mon: 25 })
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
