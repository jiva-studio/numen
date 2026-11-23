/** The orientation of a side, and how a depth changes it. */
import { describe, expect, it } from 'vitest'
import { getOrientation, findSideOrientation, getCrossOrientation, type Side } from './node'

describe('getCrossOrientation', () => {
  it('turns a row into a column', () => {
    expect(getCrossOrientation('horizontal')).toBe('vertical')
  })

  it('turns a column into a row', () => {
    expect(getCrossOrientation('vertical')).toBe('horizontal')
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
