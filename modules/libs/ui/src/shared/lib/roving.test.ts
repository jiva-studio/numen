/**
 * Where the keyboard goes in a list, whatever the list holds.
 */
import { describe, expect, it } from 'vitest'
import { stepTo } from './roving'

const isOn = (item: { readonly off?: boolean }): boolean => !item.off

const ITEMS = [{}, { off: true }, {}]

describe('where the keyboard goes', () => {
  it('lands on the first from nowhere, and on the last counting back', () => {
    expect(stepTo(ITEMS, -1, 1, isOn)).toBe(0)
    expect(stepTo(ITEMS, 0, -1, isOn)).toBe(2)
  })

  it('passes over what cannot be landed on', () => {
    expect(stepTo(ITEMS, 0, 1, isOn)).toBe(2)
    expect(stepTo(ITEMS, 2, -1, isOn)).toBe(0)
  })

  it('wraps at either end', () => {
    expect(stepTo(ITEMS, 2, 1, isOn)).toBe(0)
    expect(stepTo(ITEMS, 0, -1, isOn)).toBe(2)
  })

  it('answers nothing when there is nothing to land on', () => {
    expect(stepTo([], -1, 1, isOn)).toBe(-1)
    expect(stepTo([{ off: true }], -1, 1, isOn)).toBe(-1)
  })

  it('stays where it is when one item is all there is to be on', () => {
    expect(stepTo([{}], 0, 1, isOn)).toBe(0)
    expect(stepTo([{}], 0, -1, isOn)).toBe(0)
  })
})
