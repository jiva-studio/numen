/** The orientation a side asks for, and how a depth turns it. */
import { describe, expect, it } from 'vitest'
import { orientationAt, orientationOf, orthogonal, type Side } from './node'

describe('orthogonal', () => {
  it('turns a row into a column', () => {
    expect(orthogonal('horizontal')).toBe('vertical')
  })

  it('turns a column into a row', () => {
    expect(orthogonal('vertical')).toBe('horizontal')
  })
})

describe('orientationAt', () => {
  it('alternates with the depth', () => {
    expect(orientationAt('horizontal', 0)).toBe('horizontal')
    expect(orientationAt('horizontal', 1)).toBe('vertical')
    expect(orientationAt('vertical', 0)).toBe('vertical')
    expect(orientationAt('vertical', 1)).toBe('horizontal')
  })
})

describe('orientationOf', () => {
  it.each<[Side, string | null]>([
    ['left', 'horizontal'],
    ['right', 'horizontal'],
    ['top', 'vertical'],
    ['bottom', 'vertical'],
    ['center', null],
  ])('asks %s for %s', (side, expected) => {
    expect(orientationOf(side)).toBe(expected)
  })
})
