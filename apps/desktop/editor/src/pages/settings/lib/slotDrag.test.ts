import { describe, expect, it } from 'vitest'
import {
  clampRowOffset,
  getDropIndex,
  getLandingShift,
  getRowShifts,
  type RowSpan,
} from './slotDrag'

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

const spans: readonly RowSpan[] = [
  { top: 100, height: 24 },
  { top: 124, height: 24 },
  { top: 148, height: 24 },
]

describe('the place a held row is over', () => {
  it('stays on the own place for a small move', () => {
    expect(getDropIndex(spans, 1, 5)).toBe(1)
  })

  it('moves down once the centre crosses the next row', () => {
    expect(getDropIndex(spans, 0, 30)).toBe(1)
    expect(getDropIndex(spans, 0, 50)).toBe(2)
  })

  it('moves up once the centre crosses the row above', () => {
    expect(getDropIndex(spans, 2, -30)).toBe(1)
  })

  it('stays inside the list however far the pointer goes', () => {
    expect(getDropIndex(spans, 1, 900)).toBe(2)
    expect(getDropIndex(spans, 1, -900)).toBe(0)
    expect(clampRowOffset(spans, 1, 900)).toBe(24)
    expect(clampRowOffset(spans, 1, -900)).toBe(-24)
  })

  it('gives the same place for the same pointer after the rows have shifted', () => {
    const first = getDropIndex(spans, 0, 30)
    expect(getRowShifts(3, 0, first ?? 0, 24)).toStrictEqual([0, -24, 0])
    expect(getDropIndex(spans, 0, 30)).toBe(first)
  })

  it('knows no place in an empty list', () => {
    expect(getDropIndex([], 0, 10)).toBeNull()
  })
})

describe('getLandingShift', () => {
  const spans = [
    { top: 0, height: 10 },
    { top: 10, height: 10 },
    { top: 20, height: 10 },
  ]

  it('is the distance to the new place going down', () => {
    expect(getLandingShift(spans, 0, 2)).toBe(20)
  })

  it('is the distance to the new place going up', () => {
    expect(getLandingShift(spans, 2, 0)).toBe(-20)
  })

  it('is nothing when the row stays', () => {
    expect(getLandingShift(spans, 1, 1)).toBe(0)
  })
})
