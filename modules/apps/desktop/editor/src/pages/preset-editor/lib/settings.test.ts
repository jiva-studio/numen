/**
 * Settings calculation, bound clamping, and reconciliation for preset tabs.
 */
import { describe, expect, it } from 'vitest'

import { DEFAULTS, NO_BOUNDS, type Settings } from '../types'
import { applyTypedSetting, isLoad } from './settings'

const settings = (over: Partial<Settings> = {}): Settings => ({ ...DEFAULTS, ...over })

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
    expect(applyTypedSetting(was, 'load', value, NO_BOUNDS)).toBe(was)
  })

  it('sets the load where the value given for it is a genuine record', () => {
    const was = settings({ load: {} })
    const after = applyTypedSetting(was, 'load', { mon: 50 }, NO_BOUNDS)
    expect(after.load).toStrictEqual({ mon: 50 })
  })
})
