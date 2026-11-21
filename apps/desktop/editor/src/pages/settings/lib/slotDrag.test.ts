import { describe, expect, it } from 'vitest'
import { getRowShifts } from './slotDrag'

describe('the shifts of the rows while one is held', () => {
  it('leave every row in place while the held row is over its own place', () => {
    expect(getRowShifts(3, 1, 1, 24)).toStrictEqual([0, 0, 0])
  })

  it('lift the rows passed when the held row goes down', () => {
    expect(getRowShifts(4, 0, 2, 24)).toStrictEqual([0, -24, -24, 0])
  })

  it('lower the rows passed when the held row goes up', () => {
    expect(getRowShifts(4, 3, 1, 24)).toStrictEqual([0, 24, 24, 0])
  })
})
