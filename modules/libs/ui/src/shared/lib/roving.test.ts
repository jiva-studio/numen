/**
 * Where the keyboard goes in a list, whatever the list holds.
 */
import { describe, expect, it } from 'vitest'
import { stepTo } from './roving'

const isOn = (item: { readonly off?: boolean }): boolean => !item.off

const ITEMS = [{}, { off: true }, {}]

describe('where the keyboard goes', () => {
  it('lands on the first from nowhere, and on the last counting back', () => {
    expect(stepTo(ITEMS, { from: -1, by: 1 }, isOn)).toBe(0)
    expect(stepTo(ITEMS, { from: 0, by: -1 }, isOn)).toBe(2)
  })

  it('passes over what cannot be landed on', () => {
    expect(stepTo(ITEMS, { from: 0, by: 1 }, isOn)).toBe(2)
    expect(stepTo(ITEMS, { from: 2, by: -1 }, isOn)).toBe(0)
  })

  it('wraps at either end', () => {
    expect(stepTo(ITEMS, { from: 2, by: 1 }, isOn)).toBe(0)
    expect(stepTo(ITEMS, { from: 0, by: -1 }, isOn)).toBe(2)
  })

  it('answers nothing when there is nothing to land on', () => {
    expect(stepTo([], { from: -1, by: 1 }, isOn)).toBe(-1)
    expect(stepTo([{ off: true }], { from: -1, by: 1 }, isOn)).toBe(-1)
  })

  it('stays where it is when one item is all there is to be on', () => {
    expect(stepTo([{}], { from: 0, by: 1 }, isOn)).toBe(0)
    expect(stepTo([{}], { from: 0, by: -1 }, isOn)).toBe(0)
  })
})
