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

  it('takes the end of a string and boundary code units accurately', () => {
    expect(isLeading(NaN)).toBe(false)
    expect(isTrailing(NaN)).toBe(false)
    expect(isLeading(0xd7ff)).toBe(false)
    expect(isLeading(0xd800)).toBe(true)
    expect(isLeading(0xdbff)).toBe(true)
    expect(isLeading(0xdc00)).toBe(false)
    expect(isTrailing(0xdbff)).toBe(false)
    expect(isTrailing(0xdc00)).toBe(true)
    expect(isTrailing(0xdfff)).toBe(true)
    expect(isTrailing(0xe000)).toBe(false)
  })
})
