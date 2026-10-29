/** The orientation of a side, and how a depth changes it. */
import { describe, expect, it } from 'vitest'
import { getOrientation, findSideOrientation, orthogonal, type Side } from './node'

describe('orthogonal', () => {
  it('turns a row into a column', () => {
    expect(orthogonal('horizontal')).toBe('vertical')
  })

  it('turns a column into a row', () => {
    expect(orthogonal('vertical')).toBe('horizontal')
  })
})

describe('getOrientation', () => {
  it('alternates with the depth', () => {
    expect(getOrientation('horizontal', 0)).toBe('horizontal')
    expect(getOrientation('horizontal', 1)).toBe('vertical')
    expect(getOrientation('vertical', 0)).toBe('vertical')
    expect(getOrientation('vertical', 1)).toBe('horizontal')
  })
})

describe('findSideOrientation', () => {
  it.each<[Side, string | null]>([
    ['left', 'horizontal'],
    ['right', 'horizontal'],
    ['top', 'vertical'],
    ['bottom', 'vertical'],
    ['center', null],
  ])('asks %s for %s', (side, expected) => {
    expect(findSideOrientation(side)).toBe(expected)
  })
})
