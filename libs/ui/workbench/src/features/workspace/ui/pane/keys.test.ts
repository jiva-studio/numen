/** What a key means along a strip of tabs. */
import { describe, expect, it } from 'vitest'
import { stepTo } from './keys'

describe('the arrows along a strip', () => {
  it('step one along, and the ends meet', () => {
    expect(stepTo('ArrowRight', 0, 3)).toBe(1)
    expect(stepTo('ArrowRight', 2, 3)).toBe(0)
    expect(stepTo('ArrowLeft', 2, 3)).toBe(1)
    expect(stepTo('ArrowLeft', 0, 3)).toBe(2)
  })

  it('stay on the only tab of a strip', () => {
    expect(stepTo('ArrowRight', 0, 1)).toBe(0)
    expect(stepTo('Home', 0, 1)).toBe(0)
    expect(stepTo('End', 0, 1)).toBe(0)
  })
})

describe('Home and End', () => {
  it('go to the two ends', () => {
    expect(stepTo('Home', 2, 4)).toBe(0)
    expect(stepTo('End', 1, 4)).toBe(3)
  })
})

describe('a key that means nothing along a strip', () => {
  it('goes nowhere', () => {
    expect(stepTo('a', 1, 3)).toBeNull()
    expect(stepTo('Enter', 1, 3)).toBeNull()
  })

  it('goes nowhere from a tab the strip does not hold', () => {
    expect(stepTo('Home', -1, 3)).toBeNull()
    expect(stepTo('ArrowRight', -1, 3)).toBeNull()
    expect(stepTo('Home', 3, 3)).toBeNull()
    expect(stepTo('End', 3, 3)).toBeNull()
    expect(stepTo('ArrowLeft', 4, 3)).toBeNull()
  })

  it('goes nowhere along a strip of no tabs', () => {
    expect(stepTo('Home', 0, 0)).toBeNull()
  })
})
