import { describe, expect, it } from 'vitest'
import { createTimeSelector, getSelectorStart } from './place'

describe('the place in a recording', () => {
  it('is the start of the time range a selector names, in milliseconds', () => {
    expect(getSelectorStart({ t: [12.5, 14] })).toBe(12_500)
    expect(getSelectorStart(createTimeSelector(61_000))).toBe(61_000)
  })

  it('is nothing for a selector without a time range', () => {
    expect(getSelectorStart(null)).toBeNull()
    expect(getSelectorStart({ page: 3 })).toBeNull()
  })
})
