/**
 * The sizes a range reaches, the rungs between its ends, and the number a
 * person typed.
 */
import { describe, expect, it } from 'vitest'
import { isInBounds, ladder, parseSize } from './appearance'

describe('whether a range reaches a size', () => {
  const range = { least: 0.8, most: 2 }

  it('is so for each end and for a size between them', () => {
    expect([range.least, 1, range.most].map((size) => isInBounds(range, size))).toStrictEqual([
      true,
      true,
      true,
    ])
  })

  it('is not so for a size outside either end', () => {
    expect([0.79, 2.01].map((size) => isInBounds(range, size))).toStrictEqual([false, false])
  })

  it('is never so for a range holding nothing', () => {
    expect(isInBounds({ least: 0, most: 0 }, 0)).toBe(false)
    expect(isInBounds({ least: 1, most: 1 }, 1)).toBe(false)
  })

  it('is not so for a range whose far end lies below its near end', () => {
    expect(isInBounds({ least: 2, most: 1 }, 1.5)).toBe(false)
  })

  it('is not so for a range reaching down to nothing or below', () => {
    expect(isInBounds({ least: 0, most: 1 }, 0.5)).toBe(false)
    expect(isInBounds({ least: -1, most: 1 }, 0.5)).toBe(false)
  })
})

describe('the sizes standing between the ends of a range', () => {
  it('are every step inside it, with each end itself', () => {
    expect(ladder({ least: 0.8, most: 1.2 })).toStrictEqual([0.8, 0.9, 1, 1.1, 1.2])
  })

  it('offer an end that falls between two steps', () => {
    expect(ladder({ least: 0.85, most: 1.15 })).toStrictEqual([0.85, 0.9, 1, 1.1, 1.15])
  })

  it('are each end once where the ends fall on steps', () => {
    expect(ladder({ least: 1, most: 1.1 })).toStrictEqual([1, 1.1])
  })

  it('are none for a range holding nothing', () => {
    expect(ladder({ least: 0, most: 0 })).toStrictEqual([])
    expect(ladder({ least: 2, most: 1 })).toStrictEqual([])
  })
})

describe('the size a typed number names', () => {
  it('is the percent typed, as a multiplier', () => {
    expect(parseSize('150')).toBe(1.5)
    expect(parseSize(' 120 % ')).toBe(1.2)
  })

  it('is nothing where what was typed is not digits', () => {
    expect(parseSize('big')).toBeNull()
    expect(parseSize('')).toBeNull()
  })
})
