/** Shares always come back summing to one, whatever went in. */
import { describe, expect, it } from 'vitest'
import { even, fit, insert, remove, spread } from './shares'

const sums = (sizes: readonly number[]) => sizes.reduce((total, size) => total + size, 0)

describe('fit', () => {
  it('leaves shares that already sum to one alone', () => {
    expect(fit([0.25, 0.75], 2)).toStrictEqual([0.25, 0.75])
  })

  it('scales shares that do not', () => {
    expect(fit([1, 3], 2)).toStrictEqual([0.25, 0.75])
  })

  it('gives a child brought in without a size the average of the others', () => {
    const fitted = fit([2, 2], 3)
    expect(sums(fitted)).toBeCloseTo(1)
    expect(fitted[2]).toBeCloseTo(fitted[0] ?? 0)
  })

  it('divides evenly when nothing usable was given', () => {
    expect(fit([], 4)).toStrictEqual([0.25, 0.25, 0.25, 0.25])
    expect(fit([0, -1, Number.NaN], 3)).toStrictEqual(even(3))
  })

  it('has nothing to divide among no children', () => {
    expect(fit([1], 0)).toStrictEqual([])
  })
})

describe('insert', () => {
  it('takes half of the share it is put beside', () => {
    expect(insert([0.5, 0.5], 0, false)).toStrictEqual([0.25, 0.25, 0.5])
  })

  it('goes ahead of it when asked', () => {
    expect(insert([0.5, 0.5], 1, true)).toStrictEqual([0.5, 0.25, 0.25])
  })

  it('leaves the others where they were', () => {
    const after = insert([0.2, 0.8], 1, false)
    expect(after[0]).toBeCloseTo(0.2)
    expect(sums(after)).toBeCloseTo(1)
  })
})

describe('remove', () => {
  it('shares the length out in proportion', () => {
    const after = remove([0.5, 0.25, 0.25], 0)
    expect(after).toStrictEqual([0.5, 0.5])
  })

  it('leaves nothing to divide when the last one goes', () => {
    expect(remove([1], 0)).toStrictEqual([])
  })
})

describe('spread', () => {
  it('divides one share among several, in proportion to them', () => {
    const after = spread([0.5, 0.5], 1, [0.25, 0.75])
    expect(after[0]).toBeCloseTo(0.5)
    expect(after[1]).toBeCloseTo(0.125)
    expect(after[2]).toBeCloseTo(0.375)
    expect(sums(after)).toBeCloseTo(1)
  })
})
