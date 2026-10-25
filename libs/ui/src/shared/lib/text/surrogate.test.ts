import { describe, expect, it } from 'vitest'
import { isLeading, isTrailing } from './surrogate'

const PAIR = '😀'

describe('the halves of a pair', () => {
  it('notifies the leading half from the trailing one', () => {
    expect(isLeading(PAIR.charCodeAt(0))).toBe(true)
    expect(isTrailing(PAIR.charCodeAt(0))).toBe(false)
    expect(isTrailing(PAIR.charCodeAt(1))).toBe(true)
    expect(isLeading(PAIR.charCodeAt(1))).toBe(false)
  })

  it('takes a plain letter for neither', () => {
    expect(isLeading('a'.charCodeAt(0))).toBe(false)
    expect(isTrailing('a'.charCodeAt(0))).toBe(false)
  })

  it('takes the end of a string for neither', () => {
    expect(isLeading(NaN)).toBe(false)
    expect(isTrailing(NaN)).toBe(false)
  })
})
