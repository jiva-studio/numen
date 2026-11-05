import { describe, expect, it } from 'vitest'
import { INTERFACE_SCALE, onto } from './appearanceValues'

describe('the pair of sizes', () => {
  const both = { interfaceScale: 1, textScale: 1.5 }

  it('puts one in its place and leaves the other where it was', () => {
    expect(onto(both, INTERFACE_SCALE, 1.25)).toStrictEqual({
      interfaceScale: 1.25,
      textScale: 1.5,
    })
    expect(both.interfaceScale).toBe(1)
  })
})
