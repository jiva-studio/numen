/** The words a seat is counted in. */
import { describe, expect, it } from 'vitest'
import { countOf, seatWord } from './seat'

describe('counting a seat', () => {
  it('uses the singular for one and the plural for every other count', () => {
    expect(countOf('child', 1)).toBe('1 child')
    expect(countOf('child', 0)).toBe('0 children')
    expect(countOf('child', 3)).toBe('3 children')
    expect(countOf('jump', 1)).toBe('1 jump')
    expect(countOf('jump', 2)).toBe('2 jumps')
  })

  it('names one seat by its singular', () => {
    expect(seatWord('parent')).toBe('parent')
  })
})
