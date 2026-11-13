import { describe, expect, it } from 'vitest'
import { isChangedFromDefault } from './defaults'

describe('isChangedFromDefault', () => {
  it('is changed for a value other than the default', () => {
    expect(isChangedFromDefault(0.7, 0.4)).toBe(true)
    expect(isChangedFromDefault('', 'local')).toBe(true)
    expect(isChangedFromDefault(['a'], ['a', 'b'])).toBe(true)
  })

  it('is not changed for the default itself', () => {
    expect(isChangedFromDefault(0.7, 0.7)).toBe(false)
    expect(isChangedFromDefault({ size: 2 }, { size: 2 })).toBe(false)
  })

  it('is not changed without a default or without a value', () => {
    expect(isChangedFromDefault(undefined, 'local')).toBe(false)
    expect(isChangedFromDefault(true, undefined)).toBe(false)
  })

  it('distinguishes false from a missing value', () => {
    expect(isChangedFromDefault(true, false)).toBe(true)
    expect(isChangedFromDefault(null, false)).toBe(true)
  })
})
