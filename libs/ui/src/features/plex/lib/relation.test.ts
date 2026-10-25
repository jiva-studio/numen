/** The words a relation is counted in. */
import { describe, expect, it } from 'vitest'
import { formatCount, getRelationName } from './relation'

describe('counting a relation', () => {
  it('uses the singular for one and the plural for every other count', () => {
    expect(formatCount('child', 1)).toBe('1 child')
    expect(formatCount('child', 0)).toBe('0 children')
    expect(formatCount('child', 3)).toBe('3 children')
    expect(formatCount('jump', 1)).toBe('1 jump')
    expect(formatCount('jump', 2)).toBe('2 jumps')
  })

  it('names one relation by its singular', () => {
    expect(getRelationName('parent')).toBe('parent')
  })
})
