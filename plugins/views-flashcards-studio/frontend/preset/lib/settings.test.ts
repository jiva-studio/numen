/**
 * Settings calculation, bound clamping, and reconciliation for preset tabs.
 */
import { describe, expect, it } from 'vitest'

import { DEFAULTS, NO_BOUNDS, type Settings } from '../types'
import {
  applyGoal,
  applyTypedSetting,
  clampShares,
  findGridIndex,
  isLoad,
  reconcileSettings,
} from './settings'

const today = '2026-08-30'

const settings = (overrides: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...overrides })

describe('whether a value is a load', () => {
  it('is true for a record of daily numeric shares', () => {
    expect(isLoad({ mon: 50, tue: 75 })).toBe(true)
  })

  it('is true for an empty record', () => {
    expect(isLoad({})).toBe(true)
  })

  it('is false for null, which typeof reports as an object', () => {
    expect(isLoad(null as unknown as Settings['load'])).toBe(false)
  })

  it('is false for a record holding a share that is not a number', () => {
    expect(isLoad({ mon: 50, tue: 'half' } as unknown as Settings['load'])).toBe(false)
  })

  it('is false for a string, a number, and a boolean', () => {
    expect(isLoad('monday')).toBe(false)
    expect(isLoad(50)).toBe(false)
    expect(isLoad(true)).toBe(false)
  })
})

describe('applying a typed setting', () => {
  it('leaves the load unchanged where the value given for it is null', () => {
    const was = settings({ load: { mon: 50 } })
    const value = null as unknown as Settings['load']
    expect(applyTypedSetting(was, { field: 'load', value }, NO_BOUNDS)).toBe(was)
  })

  it('sets the load where the value given for it is a genuine record', () => {
    const was = settings({ load: {} })
    const after = applyTypedSetting(was, { field: 'load', value: { mon: 50 } }, NO_BOUNDS)
    expect(after.load).toStrictEqual({ mon: 50 })
  })
})

describe('holding the shares of a load to bounds', () => {
  it('leaves the load as it is where no bound is answered', () => {
    const load = { mon: 50.4 }
    expect(clampShares(load, undefined)).toBe(load)
  })

  it('rounds each share and holds it inside the bound', () => {
    expect(clampShares({ mon: 50.4, tue: 140 }, { min: 0, max: 100 })).toStrictEqual({
      mon: 50,
      tue: 100,
    })
  })
})

describe('reconciling a read with the edits not yet written', () => {
  it('is the read itself where nothing is unwritten', () => {
    const readItem = settings({ newADay: 3 })
    expect(reconcileSettings(readItem, settings(), new Set())).toBe(readItem)
  })

  it('keeps each unwritten field and takes the rest from the read', () => {
    const readItem = settings({ newADay: 3, interval: 40 })
    const was = settings({ newADay: 8 })
    expect(reconcileSettings(readItem, was, new Set(['newADay']))).toStrictEqual(
      settings({ newADay: 8, interval: 40 }),
    )
  })
})

describe('applying a typed choice', () => {
  it('counts the budget in either unit it names', () => {
    expect(
      applyTypedSetting(settings(), { field: 'counts', value: 'shows' }, NO_BOUNDS).counts,
    ).toBe('shows')
    const shows = settings({ counts: 'shows' })
    expect(applyTypedSetting(shows, { field: 'counts', value: 'cards' }, NO_BOUNDS).counts).toBe(
      'cards',
    )
  })

  it('leaves the budget where the unit given is none of its own', () => {
    const was = settings()
    expect(applyTypedSetting(was, { field: 'counts', value: 'minutes' }, NO_BOUNDS)).toBe(was)
  })

  it('takes either rule for what is learned', () => {
    const ruled = settings({ learned: 'retention' })
    expect(
      applyTypedSetting(settings(), { field: 'learned', value: 'retention' }, NO_BOUNDS).learned,
    ).toBe('retention')
    expect(
      applyTypedSetting(ruled, { field: 'learned', value: 'interval' }, NO_BOUNDS).learned,
    ).toBe('interval')
  })

  it('leaves the rule where the value given is none of its own', () => {
    const was = settings()
    expect(applyTypedSetting(was, { field: 'learned', value: 'shows' }, NO_BOUNDS)).toBe(was)
  })

  it('takes a day for the day aimed at, and no number', () => {
    const was = settings()
    expect(applyTypedSetting(was, { field: 'byDate', value: '2026-09-29' }, NO_BOUNDS).byDate).toBe(
      '2026-09-29',
    )
    expect(applyTypedSetting(was, { field: 'byDate', value: 5 }, NO_BOUNDS)).toBe(was)
  })

  it('takes a yes or a no for an even load, and nothing else', () => {
    const was = settings()
    expect(
      applyTypedSetting(was, { field: 'hasEvenLoad', value: false }, NO_BOUNDS).hasEvenLoad,
    ).toBe(false)
    expect(applyTypedSetting(was, { field: 'hasEvenLoad', value: 'no' }, NO_BOUNDS)).toBe(was)
    expect(applyTypedSetting(was, { field: 'hasEvenLoad', value: 0 }, NO_BOUNDS)).toBe(was)
  })

  it('sets no field from a value that belongs to another', () => {
    const was = settings()
    expect(applyTypedSetting(was, { field: 'newADay', value: 'shows' }, NO_BOUNDS)).toBe(was)
    expect(applyTypedSetting(was, { field: 'interval', value: 'retention' }, NO_BOUNDS)).toBe(was)
    expect(applyTypedSetting(was, { field: 'counts', value: true }, NO_BOUNDS)).toBe(was)
    expect(applyTypedSetting(was, { field: 'minutesADay', value: { mon: 50 } }, NO_BOUNDS)).toBe(
      was,
    )
  })
})

describe('where a typed value falls on the grid', () => {
  const grid = [10, 20, 30]

  it('is the place nearest a number', () => {
    expect(findGridIndex(grid, 22, today)).toBe(1)
  })

  it('is the place nearest the days to a day', () => {
    expect(findGridIndex(grid, '2026-09-28', today)).toBe(2)
  })

  it('is nowhere for a value that is neither', () => {
    expect(findGridIndex(grid, 'soon', today)).toBe(-1)
    expect(findGridIndex(grid, true, today)).toBe(-1)
  })
})

describe('aiming at a goal', () => {
  it('names a day a month off where a date is aimed at and the file names none', () => {
    expect(applyGoal(settings({ byDate: '' }), 'date', today)).toStrictEqual(
      settings({ goal: 'date', byDate: '2026-09-29' }),
    )
  })

  it('names no day for a goal that is not a date', () => {
    expect(applyGoal(settings({ byDate: '' }), 'retention', today)).toStrictEqual(
      settings({ goal: 'retention', byDate: '' }),
    )
  })
})
